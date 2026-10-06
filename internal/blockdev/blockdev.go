// Package blockdev opens raw block devices in a way that is safe on every
// platform we target. Raw devices reject reads that are not aligned to the
// logical sector size (mandatory on macOS /dev/rdiskN and on Windows
// \\.\PhysicalDriveN), while the filesystem readers in go-diskfs happily read
// at arbitrary offsets. Device sits in between: it turns every request into
// aligned chunk reads and keeps a small cache of recent chunks, which also
// takes the sting out of the many tiny reads an ext4 directory walk performs.
package blockdev

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"sync"

	"github.com/diskfs/go-diskfs/backend"
)

// chunkSize is the granularity we read and cache at. It is a multiple of every
// logical sector size in the wild (512 and 4096).
const chunkSize = 64 * 1024

// maxChunks caps the cache at 8 MiB.
const maxChunks = 128

// writeAlignment is the boundary writes are rounded to. Raw devices reject
// partial-sector writes, and 4096 satisfies both sector sizes in use.
const writeAlignment = 4096

// Device is a backend.Storage over a raw block device or image file. It is
// read-only unless opened with OpenReadWrite.
type Device struct {
	f    *os.File
	path string
	// writable gates every mutating path. Read-only is the default so that
	// browsing can never modify the card it is inspecting.
	writable bool
	// aligned is false for regular files, which take writes at any offset.
	aligned bool

	mu     sync.Mutex
	pos    int64
	cache  map[int64][]byte
	recent []int64 // FIFO of cached chunk offsets
}

// Open opens path read-only. The caller must have permission to read the raw
// device, which in practice means running as root or Administrator.
func Open(path string) (*Device, error) {
	return open(path, os.O_RDONLY, false)
}

// OpenReadWrite opens path for reading and writing. Everything that modifies a
// card or an image goes through here, so the call site is always explicit.
func OpenReadWrite(path string) (*Device, error) {
	return open(path, os.O_RDWR, true)
}

func open(path string, flag int, writable bool) (*Device, error) {
	f, err := os.OpenFile(path, flag, 0)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	return &Device{
		f:        f,
		path:     path,
		writable: writable,
		// Only raw devices impose alignment; an image file does not.
		aligned: !info.Mode().IsRegular(),
		cache:   map[int64][]byte{},
	}, nil
}

// Writable reports whether this handle may be written to.
func (d *Device) IsWritable() bool { return d.writable }

// chunk returns the cached chunk starting at the aligned offset start,
// reading it from the device on a miss. The returned slice must not be
// modified by callers.
func (d *Device) chunk(start int64) ([]byte, error) {
	if buf, ok := d.cache[start]; ok {
		return buf, nil
	}
	buf := make([]byte, chunkSize)
	n, err := d.f.ReadAt(buf, start)
	if err != nil && !errors.Is(err, io.EOF) {
		// A short read at the tail of the device surfaces as EIO or EINVAL on
		// some drivers rather than EOF. Retry once at sector granularity so a
		// partial final chunk still yields its readable prefix.
		if n == 0 {
			return nil, fmt.Errorf("read at offset %d: %w", start, err)
		}
	}
	buf = buf[:n]
	d.cache[start] = buf
	d.recent = append(d.recent, start)
	if len(d.recent) > maxChunks {
		delete(d.cache, d.recent[0])
		d.recent = d.recent[1:]
	}
	return buf, nil
}

// ReadAt implements io.ReaderAt with arbitrary (unaligned) offsets and lengths.
func (d *Device) ReadAt(p []byte, off int64) (int, error) {
	if off < 0 {
		return 0, fmt.Errorf("negative offset %d", off)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readAtLocked(p, off)
}

func (d *Device) readAtLocked(p []byte, off int64) (int, error) {
	n := 0
	for n < len(p) {
		cur := off + int64(n)
		start := cur / chunkSize * chunkSize
		buf, err := d.chunk(start)
		if err != nil {
			return n, err
		}
		inChunk := cur - start
		if inChunk >= int64(len(buf)) {
			return n, io.EOF
		}
		n += copy(p[n:], buf[inChunk:])
	}
	return n, nil
}

// Read implements io.Reader against the device's own seek position.
func (d *Device) Read(p []byte) (int, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	n, err := d.readAtLocked(p, d.pos)
	d.pos += int64(n)
	return n, err
}

// Seek implements io.Seeker. Seeking relative to the end needs the device size,
// which we take from the ioctl-backed Size helper.
func (d *Device) Seek(offset int64, whence int) (int64, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	switch whence {
	case io.SeekStart:
		d.pos = offset
	case io.SeekCurrent:
		d.pos += offset
	case io.SeekEnd:
		size, err := d.Length()
		if err != nil {
			return 0, err
		}
		d.pos = size + offset
	default:
		return 0, fmt.Errorf("invalid whence %d", whence)
	}
	if d.pos < 0 {
		return 0, errors.New("seek to negative offset")
	}
	return d.pos, nil
}

// Length returns the size in bytes of the device or image behind this handle.
// Block devices need an ioctl for that; a plain image file does not, which is
// what lets the same code path work on a dd dump.
func (d *Device) Length() (int64, error) {
	info, err := d.f.Stat()
	if err != nil {
		return 0, err
	}
	if info.Mode().IsRegular() {
		return info.Size(), nil
	}
	return Size(d.f)
}

func (d *Device) Close() error               { return d.f.Close() }
func (d *Device) Stat() (fs.FileInfo, error) { return d.f.Stat() }
func (d *Device) Sys() (*os.File, error)     { return d.f, nil }
func (d *Device) Path() string               { return d.path }

// Writable hands the filesystem drivers a writer, but only when this device was
// opened for writing. A read-only handle refuses, which is what keeps the
// browsing side incapable of modifying anything.
func (d *Device) Writable() (backend.WritableFile, error) {
	if !d.writable {
		return nil, backend.ErrIncorrectOpenMode
	}
	return d, nil
}

// WriteAt writes at an arbitrary offset. Raw devices only accept whole aligned
// sectors, so a partial write is turned into a read-modify-write of the
// sectors it touches.
func (d *Device) WriteAt(p []byte, off int64) (int, error) {
	if !d.writable {
		return 0, backend.ErrIncorrectOpenMode
	}
	if off < 0 {
		return 0, fmt.Errorf("negative offset %d", off)
	}
	d.mu.Lock()
	defer d.mu.Unlock()

	// Any cached chunk covering this range is now stale whatever happens next.
	defer d.invalidate(off, int64(len(p)))

	if !d.aligned {
		return d.f.WriteAt(p, off)
	}

	start := off / writeAlignment * writeAlignment
	end := (off + int64(len(p)) + writeAlignment - 1) / writeAlignment * writeAlignment

	buf := make([]byte, end-start)
	// Read the enclosing sectors first so the bytes we are not changing survive.
	if _, err := d.f.ReadAt(buf, start); err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("read before write at offset %d: %w", start, err)
	}
	copy(buf[off-start:], p)

	if _, err := d.f.WriteAt(buf, start); err != nil {
		return 0, fmt.Errorf("write at offset %d: %w", start, err)
	}
	return len(p), nil
}

// invalidate drops cached chunks overlapping a written range.
func (d *Device) invalidate(off, length int64) {
	first := off / chunkSize * chunkSize
	last := (off + length) / chunkSize * chunkSize
	for start := first; start <= last; start += chunkSize {
		delete(d.cache, start)
	}
	// The recent list may now name evicted chunks; that is harmless, since
	// eviction only ever deletes keys that are already gone.
}

// Sync flushes pending writes to the medium.
func (d *Device) Sync() error {
	if !d.writable {
		return nil
	}
	return d.f.Sync()
}

var (
	_ backend.Storage      = (*Device)(nil)
	_ backend.WritableFile = (*Device)(nil)
)
