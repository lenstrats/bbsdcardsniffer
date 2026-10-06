package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// psScript asks PowerShell for the physical disks. ConvertTo-Json collapses a
// single result to an object rather than an array, so the depth and the array
// subexpression keep the shape predictable.
const psScript = `@(Get-Disk | Select-Object Number,FriendlyName,Size,BusType,PartitionStyle,IsBoot,IsSystem,` +
	`@{n='Mounted';e={[bool](Get-Partition -DiskNumber $_.Number -ErrorAction SilentlyContinue | ` +
	`Where-Object DriveLetter)}}) | ConvertTo-Json -Depth 3 -Compress`

type psDisk struct {
	Number         int    `json:"Number"`
	FriendlyName   string `json:"FriendlyName"`
	Size           int64  `json:"Size"`
	BusType        string `json:"BusType"`
	PartitionStyle string `json:"PartitionStyle"`
	IsBoot         bool   `json:"IsBoot"`
	IsSystem       bool   `json:"IsSystem"`
	Mounted        bool   `json:"Mounted"`
}

func list() ([]Device, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", psScript)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("Get-Disk: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	var disks []psDisk
	if err := json.Unmarshal(bytes.TrimSpace(out), &disks); err != nil {
		return nil, fmt.Errorf("parse Get-Disk output: %w", err)
	}

	devs := make([]Device, 0, len(disks))
	for _, d := range disks {
		path := fmt.Sprintf(`\\.\PhysicalDrive%d`, d.Number)
		removable := strings.EqualFold(d.BusType, "USB") || strings.EqualFold(d.BusType, "SD") ||
			strings.EqualFold(d.BusType, "MMC")
		devs = append(devs, Device{
			ID:        fmt.Sprintf("%d", d.Number),
			Path:      path,
			Node:      path,
			Name:      d.FriendlyName,
			Size:      d.Size,
			SizeHuman: HumanSize(d.Size),
			Removable: removable,
			Internal:  d.IsBoot || d.IsSystem || !removable,
			Bus:       d.BusType,
			Content:   d.PartitionStyle,
			Mounted:   d.Mounted,
		})
	}
	return devs, nil
}
