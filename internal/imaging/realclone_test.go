package imaging

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRealCloneAtScale clones a slice off a real card or image, to check the
// copy holds up on actual hardware rather than only on synthetic data. Point
// REAL_IMAGE at a device or image to run it.
func TestRealCloneAtScale(t *testing.T) {
	src := os.Getenv("REAL_IMAGE")
	if src == "" {
		t.Skip("set REAL_IMAGE to a card image or device to run this")
	}

	const limit = 2 << 30 // 2 GiB is enough to be a real transfer, not a unit test
	dest := filepath.Join(t.TempDir(), "scale.img")

	var updates int
	start := time.Now()
	res, err := Clone(context.Background(), CloneOptions{
		Source: src, Dest: dest, Limit: limit, Verify: true,
	}, func(Progress) { updates++ })
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	if res.Bytes != limit {
		t.Errorf("copied %d bytes, want %d", res.Bytes, limit)
	}
	if !res.Verified {
		t.Error("verification did not run")
	}
	if updates < 2 {
		t.Errorf("only %d progress updates over a %d byte copy", updates, limit)
	}

	// Compare against the source independently of the clone's own hashing.
	f, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	digest := sha256.New()
	if _, err := io.CopyN(digest, f, limit); err != nil {
		t.Fatalf("hash source: %v", err)
	}
	if want := hex.EncodeToString(digest.Sum(nil)); res.SHA256 != want {
		t.Errorf("clone hashed to %s, source is %s", res.SHA256, want)
	}

	mbps := float64(res.Bytes) / res.Seconds / 1e6
	t.Logf("cloned and verified %d bytes in %s (%.0f MB/s including verification)",
		res.Bytes, time.Since(start).Round(time.Millisecond), mbps)
}
