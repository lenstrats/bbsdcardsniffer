package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"unicode/utf8"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"bbsdcardsniffer/internal/device"
	"bbsdcardsniffer/internal/i18n"
	"bbsdcardsniffer/internal/volume"
)

// App is the bridge between the frontend and the disk layer. It holds at most
// one open device at a time, which matches how the UI works: pick a card, then
// browse it.
type App struct {
	ctx context.Context
	// autoOpen is a device or image path given on the command line, which the
	// frontend opens as soon as it starts.
	autoOpen string

	mu      sync.Mutex
	session *volume.Session

	transfers transfers
	assistant assistantState

	// backedUp maps an opened medium to where its backup was written this
	// session, so the editing gate knows whether one exists.
	backedUp map[string]string
}

func NewApp(autoOpen string) *App {
	return &App{autoOpen: autoOpen}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// shutdown releases the device so the card can be ejected cleanly.
func (a *App) shutdown(context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session != nil {
		a.session.Close()
		a.session = nil
	}
}

// Environment tells the frontend what it needs to explain the situation to the
// user before anything goes wrong.
type Environment struct {
	OS string `json:"os"`
	// Elevated reports whether we can expect raw device reads to be permitted.
	Elevated bool `json:"elevated"`
	// CanUnmount reports whether UnmountDisk does anything on this platform.
	CanUnmount bool `json:"canUnmount"`
	// AutoOpen is the device or image path passed on the command line, if any.
	AutoOpen string `json:"autoOpen"`
}

func (a *App) Environment() Environment {
	return Environment{
		OS:         runtime.GOOS,
		Elevated:   os.Geteuid() == 0,
		CanUnmount: runtime.GOOS == "darwin",
		AutoOpen:   a.autoOpen,
	}
}

// ListDevices returns the physical disks attached to the machine.
func (a *App) ListDevices() ([]device.Device, error) {
	return device.List()
}

// DiskInfo is the description of an opened disk handed to the frontend.
type DiskInfo struct {
	Path      string          `json:"path"`
	Size      int64           `json:"size"`
	SizeHuman string          `json:"sizeHuman"`
	Table     string          `json:"table"`
	Volumes   []volume.Volume `json:"volumes"`
	// Writable reports whether this disk was opened for editing.
	Writable bool `json:"writable"`
}

// OpenDevice opens a disk read-only and describes its volumes. Opening a second
// device closes the first.
func (a *App) OpenDevice(devPath string) (*DiskInfo, error) {
	return a.open(devPath)
}

// OpenImage opens a disk image file instead of a device, so a dump made with
// dd can be inspected exactly like the card it came from. It returns nil if
// the user cancels the dialog.
func (a *App) OpenImage() (*DiskInfo, error) {
	imgPath, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Open disk image",
		Filters: []wruntime.FileFilter{
			{DisplayName: "Disk images (*.img, *.iso, *.bin, *.dmg)", Pattern: "*.img;*.iso;*.bin;*.dmg"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return nil, err
	}
	if imgPath == "" {
		return nil, nil
	}
	return a.open(imgPath)
}

// open replaces the current session with one for path.
func (a *App) open(path string) (*DiskInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.session != nil {
		a.session.Close()
		a.session = nil
	}

	s, err := volume.Open(path)
	if err != nil {
		return nil, describeOpenError(path, err)
	}
	a.session = s

	return &DiskInfo{
		Path:      s.Path,
		Size:      s.Size,
		SizeHuman: s.SizeHuman,
		Table:     s.Table,
		Volumes:   s.Volumes,
		Writable:  s.Writable,
	}, nil
}

// describeOpenError turns the two failure modes users actually hit into
// something actionable. The underlying error is kept as detail: it comes from
// the operating system, is not translated, and is what a search engine will
// recognise.
func describeOpenError(devPath string, err error) error {
	switch {
	case errors.Is(err, os.ErrPermission):
		return fmt.Errorf("%s (%w)", i18n.T().NoPermission(devPath), err)
	case errors.Is(err, os.ErrExist), strings.Contains(err.Error(), "resource busy"):
		return fmt.Errorf("%s (%w)", i18n.T().DeviceBusy(devPath), err)
	}
	return err
}

// SetLocale switches the language the backend phrases its messages in. The
// frontend calls this on start and whenever the user picks another language,
// so both halves of the application speak the same one.
func (a *App) SetLocale(locale string) {
	i18n.Set(i18n.Locale(locale))
}

// CloseDevice releases the currently open disk.
func (a *App) CloseDevice() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return nil
	}
	err := a.session.Close()
	a.session = nil
	return err
}

// UnmountDisk asks the OS to unmount every volume of a whole disk. On macOS a
// mounted card cannot be read raw, and this is the supported way to free it.
func (a *App) UnmountDisk(id string) error {
	if runtime.GOOS != "darwin" {
		return errors.New(i18n.T().UnmountUnsupported())
	}
	out, err := exec.Command("diskutil", "unmountDisk", id).CombinedOutput()
	if err != nil {
		return errors.New(i18n.T().UnmountFailed(id, strings.TrimSpace(string(out))))
	}
	return nil
}

// current returns the open session, or a clear error if there is none.
func (a *App) current() (*volume.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return nil, errors.New(i18n.T().NoDeviceOpen())
	}
	return a.session, nil
}

// ListDir lists a directory inside a volume. dir is rooted at "/".
func (a *App) ListDir(volIndex int, dir string) ([]volume.Entry, error) {
	s, err := a.current()
	if err != nil {
		return nil, err
	}
	return s.ReadDir(volIndex, dir)
}

// Preview is a window onto a file's bytes, for the hex/text pane.
type Preview struct {
	// Data is base64 so it survives the JSON bridge untouched.
	Data       string `json:"data"`
	Offset     int64  `json:"offset"`
	Length     int    `json:"length"`
	Total      int64  `json:"total"`
	TotalHuman string `json:"totalHuman"`
	// IsText reports whether the chunk decodes as UTF-8 without control
	// characters, so the UI knows whether a text view makes sense.
	IsText bool `json:"isText"`
}

// maxPreview caps a single preview request; the UI pages through larger files.
const maxPreview = 256 * 1024

// PreviewFile reads at most length bytes at offset from a file on a volume.
func (a *App) PreviewFile(volIndex int, name string, offset int64, length int) (*Preview, error) {
	s, err := a.current()
	if err != nil {
		return nil, err
	}
	if length <= 0 || length > maxPreview {
		length = 4096
	}
	if offset < 0 {
		offset = 0
	}

	data, total, err := s.ReadChunk(volIndex, name, offset, length)
	if err != nil {
		return nil, err
	}
	return &Preview{
		Data:       base64.StdEncoding.EncodeToString(data),
		Offset:     offset,
		Length:     len(data),
		Total:      total,
		TotalHuman: device.HumanSize(total),
		IsText:     looksLikeText(data),
	}, nil
}

// looksLikeText is deliberately conservative: valid UTF-8 with no NUL bytes and
// only the usual whitespace among the control characters.
func looksLikeText(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	if !utf8.Valid(b) {
		return false
	}
	for _, c := range b {
		if c == 0 {
			return false
		}
		if c < 0x20 && c != '\t' && c != '\n' && c != '\r' {
			return false
		}
	}
	return true
}

// ExportFile copies one file off the card to a location the user picks.
// It returns the destination path, or "" if the user cancelled.
func (a *App) ExportFile(volIndex int, name string) (string, error) {
	s, err := a.current()
	if err != nil {
		return "", err
	}

	dest, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Save file",
		DefaultFilename: path.Base(name),
	})
	if err != nil {
		return "", err
	}
	if dest == "" {
		return "", nil
	}

	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	defer f.Close()

	if _, err := s.CopyFile(volIndex, name, f); err != nil {
		return "", err
	}
	return dest, nil
}

// ExportResult reports what a recursive export actually managed to copy.
type ExportResult struct {
	Dest  string `json:"dest"`
	Files int    `json:"files"`
	Bytes int64  `json:"bytes"`
	// Skipped names entries that could not be copied, with the reason. Broken
	// symlinks and unreadable inodes are common on a card being recovered, and
	// aborting the whole export over one of them would be unhelpful.
	Skipped []string `json:"skipped"`
}

// ExportTree copies a directory and everything under it to a chosen folder.
func (a *App) ExportTree(volIndex int, dir string) (*ExportResult, error) {
	s, err := a.current()
	if err != nil {
		return nil, err
	}

	target, err := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Choose a destination folder",
	})
	if err != nil {
		return nil, err
	}
	if target == "" {
		return nil, nil
	}

	base := path.Base(path.Clean("/" + dir))
	if base == "/" || base == "." {
		base = fmt.Sprintf("volume%d", volIndex+1)
	}
	root := filepath.Join(target, base)

	res := &ExportResult{Dest: root}
	if err := a.exportInto(s, volIndex, dir, root, res); err != nil {
		return nil, err
	}
	return res, nil
}

// exportInto walks one directory level and recurses, accumulating into res.
func (a *App) exportInto(s *volume.Session, volIndex int, src, dst string, res *ExportResult) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := s.ReadDir(volIndex, src)
	if err != nil {
		return err
	}

	for _, e := range entries {
		outPath := filepath.Join(dst, e.Name)
		switch {
		case e.IsSymlink:
			// Following symlinks off a foreign filesystem risks writing
			// outside the destination, so record them and move on.
			res.Skipped = append(res.Skipped, e.Path+": symlink")
		case e.IsDir:
			if err := a.exportInto(s, volIndex, e.Path, outPath, res); err != nil {
				res.Skipped = append(res.Skipped, e.Path+": "+err.Error())
			}
		default:
			n, err := a.exportFileTo(s, volIndex, e.Path, outPath)
			if err != nil {
				res.Skipped = append(res.Skipped, e.Path+": "+err.Error())
				continue
			}
			res.Files++
			res.Bytes += n
		}
	}
	return nil
}

func (a *App) exportFileTo(s *volume.Session, volIndex int, src, dst string) (int64, error) {
	f, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	n, err := s.CopyFile(volIndex, src, f)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, fs.ErrNotExist) {
		return n, err
	}
	return n, nil
}
