package blockdev

import (
	"encoding/binary"
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

const ioctlDiskGetLengthInfo = 0x0007405C

// Size returns the size in bytes of an opened physical drive.
func Size(f *os.File) (int64, error) {
	var out [8]byte
	var returned uint32
	err := windows.DeviceIoControl(
		windows.Handle(f.Fd()),
		ioctlDiskGetLengthInfo,
		nil, 0,
		&out[0], uint32(len(out)),
		&returned, nil,
	)
	if err != nil {
		return 0, fmt.Errorf("IOCTL_DISK_GET_LENGTH_INFO: %w", err)
	}
	return int64(binary.LittleEndian.Uint64(out[:])), nil
}
