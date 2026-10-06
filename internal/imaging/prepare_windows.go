package imaging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/sys/windows"

	"bbsdcardsniffer/internal/device"
)

// Windows will not let a process write to a physical drive while the volumes
// on it are mounted: the writes are silently discarded or the handle is
// invalidated partway. The documented sequence is to open each volume, lock
// it, dismount it, and hold those handles open for the duration of the write.
const (
	fsctlLockVolume     = 0x00090018
	fsctlDismountVolume = 0x00090020
)

// prepareForWrite locks and dismounts every volume of the target disk. The
// returned function closes the handles, which releases the locks.
func prepareForWrite(d *device.Device) (func(), error) {
	letters, err := driveLetters(d.ID)
	if err != nil {
		return nil, err
	}

	var handles []windows.Handle
	release := func() {
		for _, h := range handles {
			windows.CloseHandle(h)
		}
	}

	for _, letter := range letters {
		h, err := lockVolume(letter)
		if err != nil {
			release()
			return nil, err
		}
		handles = append(handles, h)
	}
	return release, nil
}

// lockVolume opens \\.\X: and takes it out of service for the write.
func lockVolume(letter string) (windows.Handle, error) {
	path, err := windows.UTF16PtrFromString(`\\.\` + letter + `:`)
	if err != nil {
		return 0, err
	}
	h, err := windows.CreateFile(path,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		return 0, fmt.Errorf("open volume %s: %w", letter, err)
	}

	var returned uint32
	if err := windows.DeviceIoControl(h, fsctlLockVolume, nil, 0, nil, 0, &returned, nil); err != nil {
		windows.CloseHandle(h)
		return 0, fmt.Errorf("lock volume %s (close anything using it and try again): %w", letter, err)
	}
	if err := windows.DeviceIoControl(h, fsctlDismountVolume, nil, 0, nil, 0, &returned, nil); err != nil {
		windows.CloseHandle(h)
		return 0, fmt.Errorf("dismount volume %s: %w", letter, err)
	}
	return h, nil
}

// driveLetters lists the drive letters belonging to one physical disk.
func driveLetters(diskNumber string) ([]string, error) {
	script := fmt.Sprintf(
		`@(Get-Partition -DiskNumber %s -ErrorAction SilentlyContinue | `+
			`Where-Object DriveLetter | Select-Object -ExpandProperty DriveLetter) | ConvertTo-Json -Compress`,
		diskNumber)

	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list volumes on disk %s: %v: %s", diskNumber, err, strings.TrimSpace(stderr.String()))
	}

	trimmed := bytes.TrimSpace(out)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		// A card with no mounted volumes needs no locking.
		return nil, nil
	}

	var letters []string
	if err := json.Unmarshal(trimmed, &letters); err != nil {
		// A single letter comes back as a bare JSON string rather than an array.
		var one string
		if err2 := json.Unmarshal(trimmed, &one); err2 != nil {
			return nil, fmt.Errorf("parse volume list: %w", err)
		}
		letters = []string{one}
	}
	return letters, nil
}
