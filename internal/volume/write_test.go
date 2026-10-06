package volume

import (
	"errors"
	"strings"
	"testing"
)

// TestWriteFileRoundTrip is the check that decides whether the assistant may
// offer editing at all: go-diskfs's ext4 write path is far less exercised than
// its read path, so it is verified against a real image rather than assumed.
func TestWriteFileRoundTrip(t *testing.T) {
	img := buildImage(t)

	s, err := OpenWritable(img)
	if err != nil {
		t.Fatalf("open writable: %v", err)
	}
	if !s.Writable {
		t.Fatal("session does not report itself as writable")
	}

	const content = "PARTUUID=deadbeef-02 / ext4 defaults,noatime 0 1\n"
	if err := s.WriteFile(1, "/etc/fstab", []byte(content)); err != nil {
		t.Fatalf("write /etc/fstab: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// Reopen from scratch so the check goes through the on-disk bytes rather
	// than any state the writing session still held.
	r, err := Open(img)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer r.Close()

	got, _, err := r.ReadChunk(1, "/etc/fstab", 0, 4096)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != content {
		t.Errorf("read back %q, wrote %q", got, content)
	}

	entries, err := r.ReadDir(1, "/etc")
	if err != nil {
		t.Fatalf("read /etc after write: %v", err)
	}
	if !hasEntry(entries, "fstab") {
		t.Errorf("fstab missing from /etc: %v", names(entries))
	}
	// The directory that already existed must not have been damaged.
	if !hasEntry(entries, "hostname") {
		t.Errorf("writing fstab lost hostname: %v", names(entries))
	}
}

// Overwriting with something shorter must not leave the old tail behind.
func TestWriteFileTruncates(t *testing.T) {
	img := buildImage(t)

	s, err := OpenWritable(img)
	if err != nil {
		t.Fatalf("open writable: %v", err)
	}
	long := strings.Repeat("x", 8000)
	if err := s.WriteFile(1, "/etc/motd", []byte(long)); err != nil {
		t.Fatalf("write long: %v", err)
	}
	if err := s.WriteFile(1, "/etc/motd", []byte("short\n")); err != nil {
		t.Fatalf("write short: %v", err)
	}
	s.Close()

	r, err := Open(img)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer r.Close()

	got, total, err := r.ReadChunk(1, "/etc/motd", 0, 16384)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if string(got) != "short\n" {
		t.Errorf("read back %q, want %q", got, "short\n")
	}
	if total != int64(len("short\n")) {
		t.Errorf("size is %d, want %d", total, len("short\n"))
	}
}

// A read-only session must refuse every mutating call.
func TestReadOnlySessionRefusesWrites(t *testing.T) {
	s, err := Open(buildImage(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if err := s.WriteFile(1, "/etc/hostname", []byte("nope")); !errors.Is(err, ErrReadOnly) {
		t.Errorf("WriteFile: err = %v, want ErrReadOnly", err)
	}
	if err := s.Mkdir(1, "/nope"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Mkdir: err = %v, want ErrReadOnly", err)
	}
	if err := s.Remove(1, "/etc/hostname"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("Remove: err = %v, want ErrReadOnly", err)
	}
}
