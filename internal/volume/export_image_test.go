package volume

import (
	"os"
	"testing"
)

// TestWriteSampleImage is a helper, not a check: with SAMPLE_IMAGE set it
// writes the same GPT + FAT32 + ext4 layout the other tests use to that path,
// so the GUI can be pointed at a realistic card without one being inserted.
func TestWriteSampleImage(t *testing.T) {
	dest := os.Getenv("SAMPLE_IMAGE")
	if dest == "" {
		t.Skip("set SAMPLE_IMAGE to write a sample card image")
	}
	src := buildImage(t)
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read built image: %v", err)
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dest, err)
	}
	t.Logf("wrote %s (%d bytes)", dest, len(data))
}
