package volume

import (
	"os"
	"path/filepath"
	"testing"

	diskfs "github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/disk"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/diskfs/go-diskfs/partition/gpt"
)

const (
	sectorSize = 512
	imageSize  = 512 << 20 // 512 MiB, comfortably above the ext4 minimum
	bootStart  = 2048
	bootEnd    = bootStart + (128<<20)/sectorSize - 1
	rootStart  = bootEnd + 1
	rootEnd    = imageSize/sectorSize - 34 // leave room for the backup GPT
)

// buildImage writes a GPT image with a FAT32 boot partition and an ext4 root,
// which is the layout of every Raspberry Pi card this tool will be pointed at.
func buildImage(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "card.img")

	d, err := diskfs.Create(path, imageSize, diskfs.SectorSizeDefault)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}

	table := &gpt.Table{
		LogicalSectorSize:  sectorSize,
		PhysicalSectorSize: sectorSize,
		Partitions: []*gpt.Partition{
			{Index: 1, Start: bootStart, End: bootEnd, Type: gpt.EFISystemPartition, Name: "boot"},
			{Index: 2, Start: rootStart, End: rootEnd, Type: gpt.LinuxFilesystem, Name: "root"},
		},
	}
	if err := d.Partition(table); err != nil {
		t.Fatalf("write partition table: %v", err)
	}

	bootFS, err := d.CreateFilesystem(disk.FilesystemSpec{
		Partition: 1, FSType: filesystem.TypeFat32, VolumeLabel: "BOOT",
	})
	if err != nil {
		t.Fatalf("mkfs fat32: %v", err)
	}
	writeFile(t, bootFS, "config.txt", "dtparam=audio=on\n")

	rootFS, err := d.CreateFilesystem(disk.FilesystemSpec{
		Partition: 2, FSType: filesystem.TypeExt4, VolumeLabel: "rootfs",
	})
	if err != nil {
		t.Fatalf("mkfs ext4: %v", err)
	}
	if err := rootFS.Mkdir("etc"); err != nil {
		t.Fatalf("mkdir etc: %v", err)
	}
	writeFile(t, rootFS, "etc/hostname", "raspberrypi\n")

	if err := d.Close(); err != nil {
		t.Fatalf("close image: %v", err)
	}
	return path
}

func writeFile(t *testing.T, fs filesystem.FileSystem, name, content string) {
	t.Helper()
	f, err := fs.OpenFile(name, os.O_CREATE|os.O_RDWR)
	if err != nil {
		t.Fatalf("create %s: %v", name, err)
	}
	defer f.Close()
	if _, err := f.Write([]byte(content)); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestOpenIdentifiesVolumes(t *testing.T) {
	s, err := Open(buildImage(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	if s.Table != "gpt" {
		t.Errorf("table = %q, want gpt", s.Table)
	}
	if len(s.Volumes) != 2 {
		t.Fatalf("got %d volumes, want 2", len(s.Volumes))
	}

	boot, root := s.Volumes[0], s.Volumes[1]
	if boot.FS.Kind != "fat32" {
		t.Errorf("boot kind = %q, want fat32", boot.FS.Kind)
	}
	if !boot.Supported {
		t.Error("boot should be browsable")
	}
	if root.FS.Kind != "ext4" {
		t.Errorf("root kind = %q, want ext4", root.FS.Kind)
	}
	if !root.Supported {
		t.Error("root should be browsable")
	}
	if root.FS.Label != "rootfs" {
		t.Errorf("root label = %q, want rootfs", root.FS.Label)
	}
	if root.Name != "root" {
		t.Errorf("root partition name = %q, want root", root.Name)
	}
}

func TestBrowseAndRead(t *testing.T) {
	s, err := Open(buildImage(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	entries, err := s.ReadDir(1, "/")
	if err != nil {
		t.Fatalf("read root of ext4: %v", err)
	}
	if !hasEntry(entries, "etc") {
		t.Fatalf("no etc in %v", names(entries))
	}

	// The API takes display paths, so a nested lookup must work with a
	// leading slash even though the driver underneath wants io/fs form.
	sub, err := s.ReadDir(1, "/etc")
	if err != nil {
		t.Fatalf("read /etc: %v", err)
	}
	if !hasEntry(sub, "hostname") {
		t.Fatalf("no hostname in %v", names(sub))
	}

	data, total, err := s.ReadChunk(1, "/etc/hostname", 0, 4096)
	if err != nil {
		t.Fatalf("read /etc/hostname: %v", err)
	}
	if got := string(data); got != "raspberrypi\n" {
		t.Errorf("content = %q, want %q", got, "raspberrypi\n")
	}
	if total != int64(len("raspberrypi\n")) {
		t.Errorf("total = %d, want %d", total, len("raspberrypi\n"))
	}
}

func TestReadChunkOffset(t *testing.T) {
	s, err := Open(buildImage(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer s.Close()

	data, _, err := s.ReadChunk(1, "/etc/hostname", 7, 16)
	if err != nil {
		t.Fatalf("read at offset: %v", err)
	}
	if got := string(data); got != "rypi\n" {
		t.Errorf("content at offset 7 = %q, want %q", got, "rypi\n")
	}
}

func hasEntry(entries []Entry, name string) bool {
	for _, e := range entries {
		if e.Name == name {
			return true
		}
	}
	return false
}

func names(entries []Entry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Name)
	}
	return out
}
