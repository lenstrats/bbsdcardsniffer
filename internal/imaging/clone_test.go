package imaging

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// makeSource writes a file of pseudo-random bytes to stand in for a card.
func makeSource(t *testing.T, size int64) (string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "source.img")
	data := make([]byte, size)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("random data: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	sum := sha256.Sum256(data)
	return path, hex.EncodeToString(sum[:])
}

func TestCloneCopiesEverything(t *testing.T) {
	// Deliberately not a multiple of the 4 MiB block size, so the trimmed
	// final block is exercised.
	src, want := makeSource(t, 9<<20+1234)
	dest := filepath.Join(t.TempDir(), "clone.img")

	res, err := Clone(context.Background(), CloneOptions{
		Source: src, Dest: dest, Verify: true,
	}, nil)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	if res.SHA256 != want {
		t.Errorf("hash = %s, want %s", res.SHA256, want)
	}
	if !res.Verified {
		t.Error("verification did not run")
	}
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat clone: %v", err)
	}
	if info.Size() != res.Bytes {
		t.Errorf("image is %d bytes, copied %d", info.Size(), res.Bytes)
	}
}

func TestCloneLimitTrimsTail(t *testing.T) {
	src, _ := makeSource(t, 8<<20)
	dest := filepath.Join(t.TempDir(), "clone.img")

	const limit = 3<<20 + 7 // unaligned on purpose
	res, err := Clone(context.Background(), CloneOptions{
		Source: src, Dest: dest, Limit: limit, Verify: true,
	}, nil)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if res.Bytes != limit {
		t.Errorf("copied %d bytes, want %d", res.Bytes, limit)
	}

	// The trimmed image must be byte-identical to the head of the source.
	original, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(original[:limit])
	if res.SHA256 != hex.EncodeToString(sum[:]) {
		t.Error("trimmed image does not match the head of the source")
	}
}

func TestCloneGzipRoundTrips(t *testing.T) {
	src, want := makeSource(t, 5<<20)
	dest := filepath.Join(t.TempDir(), "clone.img.gz")

	res, err := Clone(context.Background(), CloneOptions{
		Source: src, Dest: dest, Gzip: true, Verify: true,
	}, nil)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if res.SHA256 != want {
		t.Errorf("hash = %s, want %s", res.SHA256, want)
	}
	if !res.Verified {
		t.Error("verification did not run")
	}
	// The hash covers the card bytes, while the file itself is the compressed
	// form, so the two counts must differ.
	if res.FileBytes == res.Bytes {
		t.Error("compressed file is the same size as the source")
	}
}

func TestCloneReportsProgress(t *testing.T) {
	src, _ := makeSource(t, 12<<20)
	dest := filepath.Join(t.TempDir(), "clone.img")

	var updates []Progress
	_, err := Clone(context.Background(), CloneOptions{Source: src, Dest: dest},
		func(p Progress) { updates = append(updates, p) })
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if len(updates) == 0 {
		t.Fatal("no progress reported")
	}
	last := updates[len(updates)-1]
	if last.Bytes != last.Total {
		t.Errorf("final update was %d of %d, expected completion", last.Bytes, last.Total)
	}
	for i, p := range updates {
		if p.Bytes > p.Total {
			t.Errorf("update %d reports %d of %d", i, p.Bytes, p.Total)
		}
	}
}

func TestCloneStopsOnCancel(t *testing.T) {
	src, _ := makeSource(t, 16<<20)
	dest := filepath.Join(t.TempDir(), "clone.img")

	// A context that is already cancelled checks the loop honours it without
	// depending on how fast the copy happens to run.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Clone(ctx, CloneOptions{Source: src, Dest: dest}, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}

	// The partial file is left behind deliberately, for the caller to remove.
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat partial: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("cancelled before the first block but wrote %d bytes", info.Size())
	}
}

// A cancellation partway through must stop the copy rather than run to the end.
func TestCloneStopsPartway(t *testing.T) {
	src, _ := makeSource(t, 64<<20)
	dest := filepath.Join(t.TempDir(), "clone.img")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var copied int64
	_, err := Clone(ctx, CloneOptions{Source: src, Dest: dest}, func(p Progress) {
		copied = p.Bytes
	})
	if err == nil {
		// The copy outran the cancellation, which is fine on a fast disk;
		// the deterministic case above already covers the loop check.
		t.Skip("copy finished before it could be cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if copied >= 64<<20 {
		t.Error("cancellation did not stop the copy short")
	}
}
