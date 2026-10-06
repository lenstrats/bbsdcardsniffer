package blockdev

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

// These ioctls are not (yet) exported by golang.org/x/sys/unix.
const (
	dkiocGetBlockSize  = 0x40046418
	dkiocGetBlockCount = 0x40086419
)

// Size returns the size in bytes of an opened block or character device.
func Size(f *os.File) (int64, error) {
	fd := int(f.Fd())
	blockSize, err := unix.IoctlGetInt(fd, dkiocGetBlockSize)
	if err != nil {
		return 0, fmt.Errorf("DKIOCGETBLOCKSIZE: %w", err)
	}
	blockCount, err := unix.IoctlGetInt(fd, dkiocGetBlockCount)
	if err != nil {
		return 0, fmt.Errorf("DKIOCGETBLOCKCOUNT: %w", err)
	}
	return int64(blockSize) * int64(blockCount), nil
}
