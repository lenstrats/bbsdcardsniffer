// Package device enumerates the physical disks attached to the machine, so the
// UI can offer a list to pick the inserted card from. Each platform has its own
// source of truth (diskutil, lsblk, Get-Disk); the shared part is the shape we
// hand to the frontend.
package device

import "fmt"

// Device is one whole physical disk, never a partition.
type Device struct {
	// ID is the platform's short identifier: disk4, sda, or the drive number.
	ID string `json:"id"`
	// Path is what we actually open for reading. On macOS this is the raw
	// character device, which is far faster than the buffered one.
	Path string `json:"path"`
	// Node is the path a user recognises, shown in the UI.
	Node string `json:"node"`
	// Name is the model or media name reported by the OS.
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"sizeHuman"`
	Removable bool   `json:"removable"`
	Internal  bool   `json:"internal"`
	// Bus is the transport: "Secure Digital", "USB", "nvme", ...
	Bus string `json:"bus"`
	// Content is the partition scheme as the OS sees it, useful as a sanity
	// check against what we parse ourselves.
	Content string `json:"content"`
	// Mounted reports whether the OS currently has volumes of this disk
	// mounted. On macOS reading a mounted disk raw needs it unmounted first.
	Mounted bool `json:"mounted"`
}

// List returns the physical disks, removable ones first, since those are what
// this tool is pointed at in practice.
func List() ([]Device, error) {
	devs, err := list()
	if err != nil {
		return nil, err
	}
	// Stable partition: removable before fixed, order within each preserved.
	out := make([]Device, 0, len(devs))
	for _, d := range devs {
		if d.Removable || !d.Internal {
			out = append(out, d)
		}
	}
	for _, d := range devs {
		if !(d.Removable || !d.Internal) {
			out = append(out, d)
		}
	}
	return out, nil
}

// HumanSize formats a byte count the way disk sizes are normally quoted.
func HumanSize(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
