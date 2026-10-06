package blockdev

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// Size returns the size in bytes of an opened block device.
func Size(f *os.File) (int64, error) {
	size, err := unix.IoctlGetInt(int(f.Fd()), unix.BLKGETSIZE64)
	if err != nil {
		return 0, fmt.Errorf("BLKGETSIZE64: %w", err)
	}
	return int64(size), nil
}
