// Package volume opens a raw disk read-only, describes the volumes on it, and
// browses the ones we have a filesystem driver for. Nothing here ever writes to
// the device under inspection.
package volume

import (
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/filesystem/ext4"
	"github.com/diskfs/go-diskfs/filesystem/fat12"
	"github.com/diskfs/go-diskfs/filesystem/fat16"
	"github.com/diskfs/go-diskfs/filesystem/fat32"
	"github.com/diskfs/go-diskfs/filesystem/iso9660"
	"github.com/diskfs/go-diskfs/filesystem/squashfs"

	"bbsdcardsniffer/internal/blockdev"
	"bbsdcardsniffer/internal/device"
	"bbsdcardsniffer/internal/i18n"
)

// Volume is one browsable (or at least identifiable) region of the disk.
type Volume struct {
	// Index addresses this volume in the session's API calls.
	Index int `json:"index"`
	// Name is the GPT partition name, empty for MBR.
	Name      string `json:"name"`
	UUID      string `json:"uuid"`
	Start     int64  `json:"start"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"sizeHuman"`
	// FS is what the signature scan found.
	FS FSInfo `json:"fs"`
	// Supported reports whether this tool can browse the volume's contents.
	Supported bool `json:"supported"`
	// Label prefers the filesystem label over the partition name, since that
	// is what the user recognises.
	Label string `json:"label"`
}

// Session is an opened disk. It is safe for concurrent use; the Wails runtime
// can call into it from several frontend requests at once.
type Session struct {
	Path      string   `json:"path"`
	Size      int64    `json:"size"`
	SizeHuman string   `json:"sizeHuman"`
	Table     string   `json:"table"`
	Volumes   []Volume `json:"volumes"`

	// Writable reports whether this session may modify the medium.
	Writable bool `json:"writable"`

	dev  *blockdev.Device
	disk *disk.Disk

	mu      sync.Mutex
	mounted map[int]filesystem.FileSystem
}

// Open opens the disk at path read-only and describes its volumes. It does not
// yet parse any filesystem: that happens lazily, per volume, on first browse.
func Open(path string) (*Session, error) {
	return openSession(path, false)
}

// OpenWritable opens the disk so its filesystems can be modified. Reserved for
// the assistant's editing tools; everything else uses Open.
func OpenWritable(path string) (*Session, error) {
	return openSession(path, true)
}

func openSession(path string, writable bool) (*Session, error) {
	open := blockdev.Open
	if writable {
		open = blockdev.OpenReadWrite
	}
	dev, err := open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}

	d, err := diskfs.OpenBackend(dev, diskfs.WithOpenMode(diskfs.ReadOnly))
	if err != nil {
		dev.Close()
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	s := &Session{
		Path:      path,
		Size:      d.Size,
		SizeHuman: device.HumanSize(d.Size),
		Writable:  writable,
		dev:       dev,
		disk:      d,
		mounted:   map[int]filesystem.FileSystem{},
	}

	table, err := d.GetPartitionTable()
	if err != nil || table == nil || len(table.GetPartitions()) == 0 {
		// No partition table is normal for a card written with a single
		// filesystem straight to the device. Treat the disk as one volume.
		s.Table = "none"
		s.Volumes = []Volume{s.describe(0, "", "", 0, d.Size)}
		return s, nil
	}

	s.Table = table.Type()
	for _, p := range table.GetPartitions() {
		if p.GetSize() == 0 {
			// GPT tables carry unused slots; they are not volumes.
			continue
		}
		s.Volumes = append(s.Volumes, s.describe(
			len(s.Volumes), p.Label(), p.UUID(), p.GetStart(), p.GetSize()))
	}
	return s, nil
}

// describe fills in everything we can learn about a volume without mounting it.
func (s *Session) describe(index int, name, uuid string, start, size int64) Volume {
	v := Volume{
		Index:     index,
		Name:      name,
		UUID:      uuid,
		Start:     start,
		Size:      size,
		SizeHuman: device.HumanSize(size),
	}
	if info, err := probe(io.NewSectionReader(s.dev, start, size)); err == nil {
		v.FS = info
		v.Supported = readable[info.Kind]
	}
	v.Label = v.FS.Label
	if v.Label == "" {
		v.Label = name
	}
	return v
}

// DataEnd returns the offset just past the last partition: the point where a
// clone can stop without losing any filesystem. It falls back to the whole
// device when there is no partition table to reason about.
//
// Note for GPT: the backup partition table lives in the last sectors of the
// device, so an image cut off here does not contain it. Everything else is
// intact and tools such as gdisk can rebuild the backup header, but callers
// should say so rather than trim silently.
func (s *Session) DataEnd() int64 {
	if len(s.Volumes) == 0 {
		return s.Size
	}
	var end int64
	for _, v := range s.Volumes {
		if e := v.Start + v.Size; e > end {
			end = e
		}
	}
	if end <= 0 || end > s.Size {
		return s.Size
	}
	return end
}

// ErrUnsupported is returned when a volume was identified but we have no
// driver to read it with. It stays a bare sentinel so errors.Is keeps working;
// the text a user sees is composed around it.
var ErrUnsupported = errors.New("filesystem not supported for browsing")

// mount parses the volume's filesystem, caching the result. Rather than letting
// go-diskfs try every driver in turn, we dispatch on what the probe found: it
// avoids five failed parses per volume and rules out misdetection.
func (s *Session) mount(index int) (filesystem.FileSystem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if fs, ok := s.mounted[index]; ok {
		return fs, nil
	}
	if index < 0 || index >= len(s.Volumes) {
		return nil, errors.New(i18n.T().NoSuchVolume(index))
	}
	v := s.Volumes[index]
	if !v.Supported {
		kind := v.FS.Kind
		if kind == "" {
			kind = "unknown"
		}
		return nil, fmt.Errorf("%s: %w", i18n.T().FSUnsupported(kind), ErrUnsupported)
	}

	// ext4 and the read-only formats want the default sector size; the FAT
	// drivers validate the value they are handed, so pass the real one.
	lbs := s.disk.LogicalBlocksize

	var (
		fs  filesystem.FileSystem
		err error
	)
	switch v.FS.Kind {
	case "ext2", "ext3", "ext4":
		fs, err = ext4.Read(s.dev, v.Size, v.Start, 0)
	case "fat32":
		fs, err = fat32.Read(s.dev, v.Size, v.Start, lbs)
	case "fat16":
		fs, err = fat16.Read(s.dev, v.Size, v.Start, lbs)
	case "fat12":
		fs, err = fat12.Read(s.dev, v.Size, v.Start, lbs)
	case "iso9660":
		fs, err = iso9660.Read(s.dev, v.Size, v.Start, 0)
	case "squashfs":
		fs, err = squashfs.Read(s.dev, v.Size, v.Start, 0)
	default:
		return nil, fmt.Errorf("%s: %w", i18n.T().FSUnsupported(v.FS.Kind), ErrUnsupported)
	}
	if err != nil {
		return nil, fmt.Errorf("mount %s volume %d: %w", v.FS.Kind, index+1, err)
	}

	s.mounted[index] = fs
	return fs, nil
}

// Entry is one directory entry as the frontend sees it.
type Entry struct {
	Name string `json:"name"`
	// Path is rooted at "/" for display and for passing back into the API.
	Path      string `json:"path"`
	IsDir     bool   `json:"isDir"`
	IsSymlink bool   `json:"isSymlink"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"sizeHuman"`
	Mode      string `json:"mode"`
	ModTime   string `json:"modTime"`
}

// toFS converts a display path ("/etc/fstab", "/") to the io/fs form the
// filesystem drivers expect ("etc/fstab", ".").
func toFS(p string) string {
	p = path.Clean("/" + strings.TrimSpace(p))
	p = strings.TrimPrefix(p, "/")
	if p == "" {
		return "."
	}
	return p
}

// ReadDir lists a directory on a volume. dir is a display path.
func (s *Session) ReadDir(index int, dir string) ([]Entry, error) {
	fs, err := s.mount(index)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := fs.ReadDir(toFS(dir))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	display := path.Clean("/" + dir)
	out := make([]Entry, 0, len(entries))
	for _, e := range entries {
		item := Entry{
			Name:  e.Name(),
			Path:  path.Join(display, e.Name()),
			IsDir: e.IsDir(),
		}
		if info, err := e.Info(); err == nil {
			item.Size = info.Size()
			item.SizeHuman = device.HumanSize(info.Size())
			item.Mode = info.Mode().String()
			item.IsSymlink = info.Mode()&iofs.ModeSymlink != 0
			if t := info.ModTime(); !t.IsZero() {
				item.ModTime = t.UTC().Format(time.RFC3339)
			}
		}
		out = append(out, item)
	}
	return out, nil
}

// ReadChunk reads at most length bytes at offset from a file, for the preview
// pane. It returns the bytes read and the file's total size.
func (s *Session) ReadChunk(index int, name string, offset int64, length int) ([]byte, int64, error) {
	fs, err := s.mount(index)
	if err != nil {
		return nil, 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := fs.Open(toFS(name))
	if err != nil {
		return nil, 0, fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()

	var total int64
	if info, err := f.Stat(); err == nil {
		total = info.Size()
	}

	if offset > 0 {
		seeker, ok := f.(io.Seeker)
		if !ok {
			return nil, total, fmt.Errorf("%s: cannot seek in this filesystem", name)
		}
		if _, err := seeker.Seek(offset, io.SeekStart); err != nil {
			return nil, total, fmt.Errorf("seek %s: %w", name, err)
		}
	}

	buf := make([]byte, length)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, total, fmt.Errorf("read %s: %w", name, err)
	}
	return buf[:n], total, nil
}

// CopyFile streams a file from the volume into w.
func (s *Session) CopyFile(index int, name string, w io.Writer) (int64, error) {
	fs, err := s.mount(index)
	if err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := fs.Open(toFS(name))
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", name, err)
	}
	defer f.Close()
	return io.Copy(w, f)
}

// Close releases the filesystem drivers and the device handle.
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, fs := range s.mounted {
		fs.Close()
	}
	s.mounted = map[int]filesystem.FileSystem{}
	return s.dev.Close()
}

// ErrReadOnly is returned when a modifying call is made on a session that was
// not opened for writing.
var ErrReadOnly = errors.New("disk opened read-only")

// readOnly wraps the sentinel with the translated explanation.
func readOnly() error {
	return fmt.Errorf("%s: %w", i18n.T().DiskReadOnly(), ErrReadOnly)
}

// truncater is implemented by the filesystems that can shorten a file. ext4
// can; the FAT drivers cannot.
type truncater interface {
	Truncate(path string, size int64) error
}

// WriteFile replaces the contents of a file, creating it if needed.
//
// go-diskfs ignores O_TRUNC, so overwriting a long file with a short one would
// otherwise leave the tail of the old contents in place — a config file that
// still ends in whatever it used to say. The size is therefore corrected
// explicitly afterwards, and a filesystem that cannot do that refuses the
// write rather than producing a mangled file.
func (s *Session) WriteFile(index int, name string, data []byte) error {
	if !s.Writable {
		return readOnly()
	}
	fs, err := s.mount(index)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	fsPath := toFS(name)

	// Note the current size before touching anything: it decides whether the
	// file has to be shortened afterwards.
	existing := int64(-1)
	if info, err := fs.Stat(fsPath); err == nil {
		existing = info.Size()
	}

	if existing > int64(len(data)) {
		if _, ok := fs.(truncater); !ok {
			return errors.New(i18n.T().CannotShorten(name, len(data)))
		}
	}

	f, err := fs.OpenFile(fsPath, os.O_RDWR|os.O_CREATE)
	if err != nil {
		return fmt.Errorf("open %s for writing: %w", name, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("write %s: %w", name, err)
	}
	// Close before truncating: the driver re-reads the inode from disk.
	if err := f.Close(); err != nil {
		return fmt.Errorf("close %s: %w", name, err)
	}

	if existing > int64(len(data)) {
		if err := fs.(truncater).Truncate(fsPath, int64(len(data))); err != nil {
			return fmt.Errorf("shorten %s: %w", name, err)
		}
	}
	return s.dev.Sync()
}

// Mkdir creates a directory and any missing parents.
func (s *Session) Mkdir(index int, name string) error {
	if !s.Writable {
		return readOnly()
	}
	fs, err := s.mount(index)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := fs.Mkdir(toFS(name)); err != nil {
		return fmt.Errorf("create %s: %w", name, err)
	}
	return s.dev.Sync()
}

// Remove deletes a file or an empty directory.
func (s *Session) Remove(index int, name string) error {
	if !s.Writable {
		return readOnly()
	}
	fs, err := s.mount(index)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := fs.Remove(toFS(name)); err != nil {
		return fmt.Errorf("remove %s: %w", name, err)
	}
	return s.dev.Sync()
}
