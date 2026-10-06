package main

import (
	"encoding/base64"
	"os"
	"testing"
)

// sampleImage builds the GPT + FAT32 + ext4 test card via the volume package's
// helper, exposed through SAMPLE_IMAGE, so the bound API can be exercised
// end to end without a real device.
func sampleImage(t *testing.T) string {
	t.Helper()
	path := os.Getenv("SAMPLE_IMAGE")
	if path == "" {
		t.Skip("set SAMPLE_IMAGE to a card image to run the API test")
	}
	if _, err := os.Stat(path); err != nil {
		t.Skipf("SAMPLE_IMAGE %s: %v", path, err)
	}
	return path
}

// TestAPIOnImage walks the same calls the frontend makes: open, list a
// directory, preview a file.
func TestAPIOnImage(t *testing.T) {
	app := NewApp("")
	defer app.shutdown(nil)

	info, err := app.OpenDevice(sampleImage(t))
	if err != nil {
		t.Fatalf("OpenDevice: %v", err)
	}
	if info.Table != "gpt" || len(info.Volumes) != 2 {
		t.Fatalf("got table %q with %d volumes", info.Table, len(info.Volumes))
	}

	root := info.Volumes[1]
	if root.FS.Kind != "ext4" || !root.Supported {
		t.Fatalf("volume 2 = %+v, want a browsable ext4", root.FS)
	}

	entries, err := app.ListDir(root.Index, "/etc")
	if err != nil {
		t.Fatalf("ListDir /etc: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("/etc is empty")
	}

	preview, err := app.PreviewFile(root.Index, "/etc/hostname", 0, 4096)
	if err != nil {
		t.Fatalf("PreviewFile: %v", err)
	}
	if !preview.IsText {
		t.Error("hostname should be detected as text")
	}
	data, err := base64.StdEncoding.DecodeString(preview.Data)
	if err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if string(data) != "raspberrypi\n" {
		t.Errorf("preview = %q, want %q", data, "raspberrypi\n")
	}
}

// Opening a second disk must release the first, or the card cannot be ejected.
func TestOpenDeviceReplacesSession(t *testing.T) {
	img := sampleImage(t)
	app := NewApp("")
	defer app.shutdown(nil)

	if _, err := app.OpenDevice(img); err != nil {
		t.Fatalf("first open: %v", err)
	}
	if _, err := app.OpenDevice(img); err != nil {
		t.Fatalf("second open: %v", err)
	}
	if err := app.CloseDevice(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, err := app.ListDir(0, "/"); err == nil {
		t.Error("ListDir after close should fail")
	}
}
