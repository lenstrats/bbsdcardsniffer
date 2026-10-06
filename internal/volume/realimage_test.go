package volume

import (
	"os"
	"testing"
)

// TestRealImage opens an image from disk, for checking this tool against cards
// it was not developed with. Point REAL_IMAGE at one to run it.
//
//	REAL_IMAGE=~/card.img go test -v -run TestRealImage ./internal/volume/
func TestRealImage(t *testing.T) {
	path := os.Getenv("REAL_IMAGE")
	if path == "" {
		t.Skip("set REAL_IMAGE to a card image or device to run this")
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer s.Close()

	t.Logf("%s: %s, %s table", s.Path, s.SizeHuman, s.Table)
	for _, v := range s.Volumes {
		t.Logf("  volume %d: %-8s label=%-12q %8s supported=%v",
			v.Index+1, v.FS.Kind, v.Label, v.SizeHuman, v.Supported)
	}

	browsed := 0
	for _, v := range s.Volumes {
		if !v.Supported {
			continue
		}
		entries, err := s.ReadDir(v.Index, "/")
		if err != nil {
			t.Errorf("volume %d (%s): read root: %v", v.Index+1, v.FS.Kind, err)
			continue
		}
		browsed++
		t.Logf("  volume %d root has %d entries: %v", v.Index+1, len(entries), names(entries))
	}
	if browsed == 0 {
		t.Fatal("no volume could be browsed")
	}
}

// TestRealImageDeepRead walks further than the root directory and reads file
// contents, since a working root inode says nothing about the extent tree or
// the hashed-directory path underneath it.
func TestRealImageDeepRead(t *testing.T) {
	path := os.Getenv("REAL_IMAGE")
	if path == "" {
		t.Skip("set REAL_IMAGE to a card image or device to run this")
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer s.Close()

	// Find the volume holding a Linux root filesystem.
	var rootVol = -1
	for _, v := range s.Volumes {
		if !v.Supported {
			continue
		}
		if entries, err := s.ReadDir(v.Index, "/"); err == nil && hasEntry(entries, "etc") {
			rootVol = v.Index
			break
		}
	}
	if rootVol < 0 {
		t.Skip("no volume with an /etc on it")
	}

	etc, err := s.ReadDir(rootVol, "/etc")
	if err != nil {
		t.Fatalf("read /etc: %v", err)
	}
	t.Logf("/etc has %d entries", len(etc))
	if len(etc) < 10 {
		t.Errorf("/etc has only %d entries, expected a populated directory", len(etc))
	}

	// Read every small regular file directly in /etc. Anything that fails
	// points at the extent or inode path rather than at one odd file.
	var read, failed int
	for _, e := range etc {
		if e.IsDir || e.IsSymlink || e.Size == 0 || e.Size > 64*1024 {
			continue
		}
		data, total, err := s.ReadChunk(rootVol, e.Path, 0, 4096)
		if err != nil {
			t.Errorf("read %s: %v", e.Path, err)
			failed++
			continue
		}
		if total != e.Size {
			t.Errorf("%s: stat size %d but open reports %d", e.Path, e.Size, total)
		}
		if len(data) == 0 {
			t.Errorf("%s: read 0 bytes of %d", e.Path, e.Size)
		}
		read++
	}
	t.Logf("read %d files in /etc, %d failed", read, failed)
	if read == 0 {
		t.Fatal("no file in /etc could be read")
	}

	// A deeper path exercises directory traversal more than one level down.
	if _, err := s.ReadDir(rootVol, "/usr/bin"); err != nil {
		t.Errorf("read /usr/bin: %v", err)
	}
}

// TestRealImageDataEnd reports where a trimmed clone of this card would stop.
func TestRealImageDataEnd(t *testing.T) {
	path := os.Getenv("REAL_IMAGE")
	if path == "" {
		t.Skip("set REAL_IMAGE to a card image or device to run this")
	}

	s, err := Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer s.Close()

	end := s.DataEnd()
	if end <= 0 || end > s.Size {
		t.Fatalf("DataEnd = %d, outside the device size %d", end, s.Size)
	}
	// It must sit past the last partition, never inside one.
	for _, v := range s.Volumes {
		if v.Start+v.Size > end {
			t.Errorf("volume %d ends at %d, past DataEnd %d", v.Index+1, v.Start+v.Size, end)
		}
	}
	t.Logf("device %d bytes, data ends at %d, trimming saves %d bytes",
		s.Size, end, s.Size-end)
}
