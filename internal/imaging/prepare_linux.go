package imaging

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"bbsdcardsniffer/internal/device"
)

// prepareForWrite unmounts every partition of the card that is currently
// mounted. Writing underneath a mounted filesystem corrupts it and confuses
// the page cache, so this is not optional.
func prepareForWrite(d *device.Device) (func(), error) {
	points, err := mountPointsFor(d.Path)
	if err != nil {
		return nil, err
	}
	for _, mp := range points {
		if out, err := exec.Command("umount", mp).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("could not unmount %s: %s", mp, strings.TrimSpace(string(out)))
		}
	}
	// Nothing to undo: the kernel re-reads the partition table on close and
	// any automounter will pick the card up again by itself.
	return func() {}, nil
}

// mountPointsFor lists the mount points of every partition of a whole disk.
func mountPointsFor(diskPath string) ([]string, error) {
	f, err := os.Open("/proc/self/mounts")
	if err != nil {
		return nil, fmt.Errorf("cannot check what is mounted: %w", err)
	}
	defer f.Close()

	var points []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		// Partitions are the disk path with a suffix: /dev/sda -> /dev/sda1,
		// /dev/mmcblk0 -> /dev/mmcblk0p1. The disk itself may be mounted too.
		if fields[0] == diskPath || strings.HasPrefix(fields[0], diskPath) {
			points = append(points, fields[1])
		}
	}
	return points, scanner.Err()
}
