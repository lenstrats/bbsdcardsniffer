package imaging

import (
	"archive/zip"
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/ulikunitz/xz"
)

// countingReader tracks how far through the image file we have read, which is
// what drives the progress bar: the uncompressed length of a compressed stream
// is not reliably known in advance, but the file's own size always is.
type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// Format is how an image file is stored.
type Format string

const (
	FormatRaw   Format = "raw"
	FormatGzip  Format = "gzip"
	FormatXZ    Format = "xz"
	FormatBzip2 Format = "bzip2"
	FormatZip   Format = "zip"
)

// magics maps a leading byte sequence to the format it identifies. Sniffing
// the content beats trusting the extension: images get renamed, and a
// mislabelled file would otherwise be written to a card verbatim.
var magics = []struct {
	prefix []byte
	format Format
}{
	{[]byte{0x1f, 0x8b}, FormatGzip},
	{[]byte{0xfd, '7', 'z', 'X', 'Z', 0x00}, FormatXZ},
	{[]byte{'B', 'Z', 'h'}, FormatBzip2},
	{[]byte{'P', 'K', 0x03, 0x04}, FormatZip},
}

// DetectFormat reads the first bytes of a file to work out how it is stored.
func DetectFormat(f *os.File) (Format, error) {
	head := make([]byte, 8)
	n, err := f.ReadAt(head, 0)
	if err != nil && n == 0 {
		return "", fmt.Errorf("read image header: %w", err)
	}
	head = head[:n]

	for _, m := range magics {
		if bytes.HasPrefix(head, m.prefix) {
			return m.format, nil
		}
	}
	return FormatRaw, nil
}

// source is a decompressed image ready to be streamed, along with what the
// progress bar should measure against.
type source struct {
	reader io.Reader
	closer io.Closer
	// total is the progress denominator.
	total int64
	// consumed returns the numerator, given how many bytes have reached the
	// card so far. Sequential formats count bytes read out of the file, since
	// that is known exactly; zip counts bytes written, because the archive
	// records the uncompressed length but not how far into it we are.
	consumed func(written int64) int64
}

// openDecompressed returns a reader over the image contents. The returned
// closer releases whatever the decompressor holds; the file itself stays the
// caller's to close.
func openDecompressed(f *os.File, format Format, size int64) (*source, error) {
	// Sequential formats read through this, so progress tracks the file.
	counter := &countingReader{r: f}
	fromFile := func(r io.Reader, c io.Closer) *source {
		return &source{
			reader:   r,
			closer:   c,
			total:    size,
			consumed: func(int64) int64 { return counter.n },
		}
	}

	switch format {
	case FormatRaw:
		return fromFile(counter, io.NopCloser(nil)), nil

	case FormatGzip:
		gz, err := gzip.NewReader(bufio.NewReaderSize(counter, 1<<20))
		if err != nil {
			return nil, fmt.Errorf("read gzip image: %w", err)
		}
		return fromFile(gz, gz), nil

	case FormatXZ:
		r, err := xz.NewReader(bufio.NewReaderSize(counter, 1<<20))
		if err != nil {
			return nil, fmt.Errorf("read xz image: %w", err)
		}
		return fromFile(r, io.NopCloser(nil)), nil

	case FormatBzip2:
		return fromFile(bzip2.NewReader(bufio.NewReaderSize(counter, 1<<20)), io.NopCloser(nil)), nil

	case FormatZip:
		// zip needs random access, so it reads the file directly and the
		// counter above never advances.
		zr, err := zip.NewReader(f, size)
		if err != nil {
			return nil, fmt.Errorf("read zip image: %w", err)
		}
		entry, err := singleImageInZip(zr)
		if err != nil {
			return nil, err
		}
		rc, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s inside the zip: %w", entry.Name, err)
		}
		return &source{
			reader:   rc,
			closer:   rc,
			total:    int64(entry.UncompressedSize64),
			consumed: func(written int64) int64 { return written },
		}, nil
	}
	return nil, fmt.Errorf("unsupported image format %q", format)
}

// singleImageInZip picks the file to write out of an archive. A zip holding
// more than one candidate is ambiguous, and guessing which to burn onto a card
// is not a decision to make silently.
func singleImageInZip(zr *zip.Reader) (*zip.File, error) {
	var candidates []*zip.File
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		candidates = append(candidates, f)
	}
	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf("the zip archive is empty")
	case 1:
		return candidates[0], nil
	}

	// More than one file: accept it only when exactly one looks like an image.
	var images []*zip.File
	for _, f := range candidates {
		if hasImageExtension(f.Name) {
			images = append(images, f)
		}
	}
	if len(images) == 1 {
		return images[0], nil
	}
	return nil, fmt.Errorf("the zip archive holds %d files; unpack the image first", len(candidates))
}

func hasImageExtension(name string) bool {
	for _, ext := range []string{".img", ".iso", ".bin", ".raw", ".dmg"} {
		if len(name) > len(ext) && name[len(name)-len(ext):] == ext {
			return true
		}
	}
	return false
}

// UncompressedSize reports how many bytes an image will occupy on a card, or 0
// when that cannot be known without decompressing the whole thing.
//
// Knowing it lets the interface refuse an image that will not fit before the
// write starts rather than halfway through. A raw image is its own size and a
// zip records the length of its entries; xz and gzip do not carry a figure we
// can trust (gzip stores it modulo 4 GiB, which is exactly the range where the
// answer matters), so those return 0 and the check happens while writing.
func UncompressedSize(f *os.File, format Format, size int64) int64 {
	switch format {
	case FormatRaw:
		return size
	case FormatZip:
		zr, err := zip.NewReader(f, size)
		if err != nil {
			return 0
		}
		entry, err := singleImageInZip(zr)
		if err != nil {
			return 0
		}
		return int64(entry.UncompressedSize64)
	}
	return 0
}
