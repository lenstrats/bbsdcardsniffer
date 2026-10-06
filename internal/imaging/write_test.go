package imaging

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"bbsdcardsniffer/internal/device"
	"github.com/ulikunitz/xz"
)

// The safety gate is the whole point of this package's write half, so it gets
// checked before anything else.

func TestWriteRefusesUnknownTarget(t *testing.T) {
	img := filepath.Join(t.TempDir(), "img")
	if err := os.WriteFile(img, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Write(context.Background(), WriteOptions{
		Source: img, Dest: "/dev/definitely-not-a-disk",
	}, nil)
	if !errors.Is(err, ErrNotRemovable) {
		t.Fatalf("err = %v, want ErrNotRemovable", err)
	}
}

func TestWriteRefusesFixedDisks(t *testing.T) {
	devices, err := device.List()
	if err != nil {
		t.Skipf("cannot list devices here: %v", err)
	}

	var fixed *device.Device
	for i := range devices {
		if !devices[i].Removable {
			fixed = &devices[i]
			break
		}
	}
	if fixed == nil {
		t.Skip("no fixed disk attached to test the refusal against")
	}

	img := filepath.Join(t.TempDir(), "img")
	if err := os.WriteFile(img, []byte("payload"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Both the raw path and the friendly node must be refused, since either
	// could arrive from the frontend.
	for _, target := range []string{fixed.Path, fixed.Node} {
		if _, err := Write(context.Background(), WriteOptions{Source: img, Dest: target}, nil); !errors.Is(err, ErrNotRemovable) {
			t.Errorf("writing to %s: err = %v, want ErrNotRemovable", target, err)
		}
	}
}

func TestResolveRemovableRejectsSystemDisk(t *testing.T) {
	if _, err := resolveRemovable(""); !errors.Is(err, ErrNotRemovable) {
		t.Errorf("empty path: err = %v, want ErrNotRemovable", err)
	}
}

// --- format detection and decompression ---

func payload(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

func writeTemp(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestDetectFormatAndRoundTrip(t *testing.T) {
	raw := payload(t, 512*1024)

	var gzBuf bytes.Buffer
	gw := gzip.NewWriter(&gzBuf)
	gw.Write(raw)
	gw.Close()

	var xzBuf bytes.Buffer
	xw, err := xz.NewWriter(&xzBuf)
	if err != nil {
		t.Fatal(err)
	}
	xw.Write(raw)
	xw.Close()

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	entry, err := zw.Create("card.img")
	if err != nil {
		t.Fatal(err)
	}
	entry.Write(raw)
	zw.Close()

	tests := []struct {
		name string
		data []byte
		want Format
	}{
		{"raw.img", raw, FormatRaw},
		{"card.img.gz", gzBuf.Bytes(), FormatGzip},
		{"card.img.xz", xzBuf.Bytes(), FormatXZ},
		{"card.zip", zipBuf.Bytes(), FormatZip},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTemp(t, tt.name, tt.data)
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close()

			format, err := DetectFormat(f)
			if err != nil {
				t.Fatalf("DetectFormat: %v", err)
			}
			if format != tt.want {
				t.Fatalf("format = %q, want %q", format, tt.want)
			}

			info, _ := f.Stat()
			src, err := openDecompressed(f, format, info.Size())
			if err != nil {
				t.Fatalf("openDecompressed: %v", err)
			}
			defer src.closer.Close()

			got, err := io.ReadAll(src.reader)
			if err != nil {
				t.Fatalf("read image: %v", err)
			}
			if !bytes.Equal(got, raw) {
				t.Errorf("decompressed %d bytes, want the original %d", len(got), len(raw))
			}
			if src.total <= 0 {
				t.Errorf("progress total is %d, which would leave the bar stuck", src.total)
			}
		})
	}
}

// Detection must go by content, since a renamed file would otherwise be
// written to a card verbatim.
func TestDetectFormatIgnoresExtension(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	gw.Write([]byte("compressed all the same"))
	gw.Close()

	f, err := os.Open(writeTemp(t, "misleading.img", buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	format, err := DetectFormat(f)
	if err != nil {
		t.Fatal(err)
	}
	if format != FormatGzip {
		t.Errorf("format = %q, want gzip despite the .img name", format)
	}
}

// An archive with several files is ambiguous; guessing which to burn is not
// a decision to make silently.
func TestZipWithSeveralFilesIsRefused(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range []string{"a.img", "b.img"} {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte("x"))
	}
	zw.Close()

	f, err := os.Open(writeTemp(t, "two.zip", buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	info, _ := f.Stat()
	if _, err := openDecompressed(f, FormatZip, info.Size()); err == nil {
		t.Error("expected a zip with two images to be refused")
	}
}

// One image beside incidental files (a readme, a checksum) is unambiguous.
func TestZipPicksTheOnlyImage(t *testing.T) {
	raw := payload(t, 4096)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	readme, _ := zw.Create("README.txt")
	readme.Write([]byte("flash me"))
	entry, _ := zw.Create("card.img")
	entry.Write(raw)
	zw.Close()

	f, err := os.Open(writeTemp(t, "mixed.zip", buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	info, _ := f.Stat()
	src, err := openDecompressed(f, FormatZip, info.Size())
	if err != nil {
		t.Fatalf("openDecompressed: %v", err)
	}
	defer src.closer.Close()

	got, err := io.ReadAll(src.reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, raw) {
		t.Error("did not pick the image out of the archive")
	}
}
