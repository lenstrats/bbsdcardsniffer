package imaging

import (
	"io"
	"os"
	"testing"
)

// TestRealCompressedImage runs a real distribution image through the write
// path's front half. Distributions ship as .img.xz, so this is the shape most
// images arriving at this tool actually have.
//
//	COMPRESSED_IMAGE=~/Downloads/Armbian_...img.xz go test -run TestRealCompressedImage ./internal/imaging/
func TestRealCompressedImage(t *testing.T) {
	path := os.Getenv("COMPRESSED_IMAGE")
	if path == "" {
		t.Skip("set COMPRESSED_IMAGE to a compressed distribution image to run this")
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}

	format, err := DetectFormat(f)
	if err != nil {
		t.Fatalf("DetectFormat: %v", err)
	}
	if format == FormatRaw {
		t.Skipf("%s is not compressed", path)
	}

	src, err := openDecompressed(f, format, info.Size())
	if err != nil {
		t.Fatalf("openDecompressed: %v", err)
	}
	defer src.closer.Close()

	// The progress bar divides by this, so a zero would leave it stuck at 0%.
	if src.total <= 0 {
		t.Errorf("progress total is %d", src.total)
	}

	n, err := io.Copy(io.Discard, src.reader)
	if err != nil {
		t.Fatalf("decompress %s: %v", format, err)
	}
	if n == 0 {
		t.Fatal("the image decompressed to nothing")
	}
	t.Logf("%s: %s, %d bytes on disk, %d bytes written to a card", path, format, info.Size(), n)
}
