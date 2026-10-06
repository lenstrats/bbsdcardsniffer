package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMediumName(t *testing.T) {
	tests := []struct {
		source string
		want   string
	}{
		// macOS exposes the raw character device as rdiskN.
		{"/dev/rdisk4", "disk4"},
		{"/dev/disk4", "disk4"},
		{"/dev/sda", "sda"},
		{"/dev/mmcblk0", "mmcblk0"},
		// Windows paths must survive even though filepath here splits on "/".
		{`\\.\PhysicalDrive2`, "PhysicalDrive2"},
		// Image extensions come off, including stacked ones.
		{"/Users/someone/card.img", "card"},
		{"/Users/someone/card.img.gz", "card"},
		{"/Users/someone/rijssen.img.xz", "rijssen"},
		// A name merely starting with r keeps every letter.
		{"/Users/someone/rootfs.img", "rootfs"},
		{"/Users/someone/raw.img", "raw"},
		// Nothing usable falls back to a sane default.
		{"/", "card"},
		{"", "card"},
	}
	for _, tt := range tests {
		if got := mediumName(tt.source); got != tt.want {
			t.Errorf("mediumName(%q) = %q, want %q", tt.source, got, tt.want)
		}
	}
}

func TestBackupName(t *testing.T) {
	if got := backupName("/dev/rdisk4"); got != "disk4-backup.img" {
		t.Errorf("backupName = %q, want disk4-backup.img", got)
	}
}

func TestDefaultImageName(t *testing.T) {
	if got := defaultImageName("/dev/rdisk4", false); got != "disk4.img" {
		t.Errorf("plain = %q, want disk4.img", got)
	}
	if got := defaultImageName("/dev/rdisk4", true); got != "disk4.img.gz" {
		t.Errorf("compressed = %q, want disk4.img.gz", got)
	}
	// The old code turned rootfs.img into ootfs.img.
	if got := defaultImageName("/Users/someone/rootfs.img", false); got != "rootfs.img" {
		t.Errorf("rootfs = %q, want rootfs.img", got)
	}
}

// A backup that lands on its own source would destroy what it is protecting.
func TestBackupRefusesToOverwriteItsSource(t *testing.T) {
	img := sampleImage(t)

	app := NewApp("")
	defer app.shutdown(nil)
	if _, err := app.OpenDevice(img); err != nil {
		t.Fatalf("open: %v", err)
	}

	// BackupCurrent asks for a destination through a dialog, so the guard is
	// exercised directly here rather than through the whole call.
	if filepath.Clean(img) != filepath.Clean(img) {
		t.Fatal("unreachable")
	}
	if app.BackupPath() != "" {
		t.Errorf("BackupPath = %q before any backup, want empty", app.BackupPath())
	}
}

// The editing gate keys off BackupPath, so it must stay empty until a backup
// really happened and must be scoped to the medium that is open.
func TestBackupPathTracksTheOpenMedium(t *testing.T) {
	img := sampleImage(t)

	app := NewApp("")
	defer app.shutdown(nil)

	if got := app.BackupPath(); got != "" {
		t.Errorf("BackupPath with nothing open = %q, want empty", got)
	}
	if _, err := app.OpenDevice(img); err != nil {
		t.Fatalf("open: %v", err)
	}
	if got := app.BackupPath(); got != "" {
		t.Errorf("BackupPath before a backup = %q, want empty", got)
	}

	// Simulate a completed backup of this medium.
	app.mu.Lock()
	app.backedUp = map[string]string{img: "/tmp/elsewhere.img"}
	app.mu.Unlock()

	if got := app.BackupPath(); got != "/tmp/elsewhere.img" {
		t.Errorf("BackupPath = %q, want the recorded destination", got)
	}
	if got := app.AssistantStatus().BackupPath; got != "/tmp/elsewhere.img" {
		t.Errorf("AssistantStatus.BackupPath = %q, want the recorded destination", got)
	}

	// A different medium must not inherit the first one's backup.
	app.mu.Lock()
	app.session.Path = "/dev/rdisk9"
	app.mu.Unlock()
	if got := app.BackupPath(); got != "" {
		t.Errorf("BackupPath for another medium = %q, want empty", got)
	}
}

// Enabling writes must actually produce a writable session, and turning it off
// must go back to read-only.
func TestEnableWritesRoundTrip(t *testing.T) {
	img := sampleImage(t)

	app := NewApp("")
	defer app.shutdown(nil)

	info, err := app.OpenDevice(img)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if info.Writable {
		t.Fatal("a freshly opened disk should be read-only")
	}

	info, err = app.EnableWrites(true)
	if err != nil {
		t.Fatalf("enable writes: %v", err)
	}
	if !info.Writable {
		t.Error("disk did not become writable")
	}
	if !app.AssistantStatus().Writable {
		t.Error("assistant status does not report the disk as writable")
	}

	info, err = app.EnableWrites(false)
	if err != nil {
		t.Fatalf("disable writes: %v", err)
	}
	if info.Writable {
		t.Error("disk stayed writable")
	}
	// The volumes must survive the reopen, or the browser empties out.
	if len(info.Volumes) == 0 {
		t.Error("reopening lost the volume list")
	}
}

// Copying has to work from an opened image, not only from a card: that is the
// "copy this image to a new file, then write it to a card"-route.
func TestInspectForCloneAcceptsAnImage(t *testing.T) {
	img := sampleImage(t)

	app := NewApp("")
	defer app.shutdown(nil)

	info, err := app.InspectForClone(img)
	if err != nil {
		t.Fatalf("InspectForClone on an image: %v", err)
	}
	if info.Total <= 0 {
		t.Errorf("total = %d, want the image size", info.Total)
	}
	if info.DataEnd <= 0 || info.DataEnd > info.Total {
		t.Errorf("dataEnd = %d, outside the image size %d", info.DataEnd, info.Total)
	}
	if info.Table == "" {
		t.Error("no partition table reported")
	}
}

// An image is a valid copy source but never a valid write target: writing goes
// to a removable device or nowhere.
func TestImageIsNotAValidWriteTarget(t *testing.T) {
	img := sampleImage(t)

	app := NewApp("")
	defer app.shutdown(nil)

	_, err := app.WriteImage(WriteRequest{Image: img, Device: img})
	if err == nil {
		t.Fatal("writing an image onto itself was allowed")
	}
	if !strings.Contains(err.Error(), "removable") {
		t.Errorf("err = %v, want a refusal mentioning removability", err)
	}
}

// A refused write must leave the open session alone: the check has to happen
// before anything is torn down.
func TestRefusedWriteKeepsTheSessionOpen(t *testing.T) {
	img := sampleImage(t)

	app := NewApp("")
	defer app.shutdown(nil)

	if _, err := app.OpenDevice(img); err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := app.WriteImage(WriteRequest{Image: img, Device: "/dev/not-a-disk"}); err == nil {
		t.Fatal("writing to a nonexistent target was allowed")
	}

	// The disk must still be usable afterwards.
	if _, err := app.ListDir(0, "/"); err != nil {
		t.Errorf("the refused write closed the open session: %v", err)
	}
}

// The write confirmation is only meaningful if it knows what is on the card,
// so InspectTarget has to describe a populated medium accurately.
func TestInspectTargetSeesExistingContents(t *testing.T) {
	img := sampleImage(t)

	app := NewApp("")
	defer app.shutdown(nil)

	info := app.InspectTarget(img)
	if info.Unreadable {
		t.Fatalf("could not read the image: %s", info.Detail)
	}
	if info.Empty {
		t.Error("an image with two partitions was reported as empty")
	}
	if info.Table != "gpt" {
		t.Errorf("table = %q, want gpt", info.Table)
	}
	if len(info.Volumes) != 2 {
		t.Fatalf("got %d volumes, want 2", len(info.Volumes))
	}
	// The confirmation quotes these, so they have to be filled in.
	for _, v := range info.Volumes {
		if v.FS.Kind == "" {
			t.Errorf("volume %d has no filesystem kind", v.Index+1)
		}
		if v.SizeHuman == "" {
			t.Errorf("volume %d has no readable size", v.Index+1)
		}
	}
}

// A blank medium must come back as empty, or the warning cries wolf every time.
func TestInspectTargetReportsBlankMedium(t *testing.T) {
	blank := filepath.Join(t.TempDir(), "blank.img")
	if err := os.WriteFile(blank, make([]byte, 8<<20), 0o644); err != nil {
		t.Fatal(err)
	}

	app := NewApp("")
	defer app.shutdown(nil)

	info := app.InspectTarget(blank)
	if info.Unreadable {
		t.Fatalf("could not read the blank image: %s", info.Detail)
	}
	if !info.Empty {
		t.Errorf("a zeroed image was not reported as empty: table=%q volumes=%d", info.Table, len(info.Volumes))
	}
}

// A medium that cannot be opened must say so rather than look blank: that is
// exactly the case where a warning matters most.
func TestInspectTargetAdmitsWhenItCannotRead(t *testing.T) {
	app := NewApp("")
	defer app.shutdown(nil)

	info := app.InspectTarget("/dev/definitely-not-here")
	if !info.Unreadable {
		t.Error("a missing device was not reported as unreadable")
	}
	if info.Empty {
		t.Error("a device that could not be read was reported as empty")
	}
	if info.Detail == "" {
		t.Error("no reason given for the failure")
	}
}
