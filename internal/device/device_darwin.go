package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// plistToJSON converts a plist blob using plutil, which ships with macOS. It
// saves us a plist parser dependency for the handful of fields we need.
func plistToJSON(data []byte) ([]byte, error) {
	cmd := exec.Command("plutil", "-convert", "json", "-o", "-", "-")
	cmd.Stdin = bytes.NewReader(data)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("plutil: %v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return out.Bytes(), nil
}

// diskutilJSON runs a diskutil subcommand that supports -plist and decodes the
// result into v.
func diskutilJSON(v any, args ...string) error {
	var stderr bytes.Buffer
	cmd := exec.Command("diskutil", args...)
	cmd.Stderr = &stderr
	raw, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("diskutil %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	js, err := plistToJSON(raw)
	if err != nil {
		return err
	}
	return json.Unmarshal(js, v)
}

type duList struct {
	WholeDisks            []string `json:"WholeDisks"`
	AllDisksAndPartitions []struct {
		DeviceIdentifier string `json:"DeviceIdentifier"`
		Partitions       []struct {
			MountPoint string `json:"MountPoint"`
		} `json:"Partitions"`
		APFSVolumes []struct {
			MountPoint string `json:"MountPoint"`
		} `json:"APFSVolumes"`
	} `json:"AllDisksAndPartitions"`
}

type duInfo struct {
	DeviceNode     string `json:"DeviceNode"`
	MediaName      string `json:"MediaName"`
	TotalSize      int64  `json:"TotalSize"`
	Internal       bool   `json:"Internal"`
	Removable      bool   `json:"Removable"`
	RemovableMedia bool   `json:"RemovableMedia"`
	BusProtocol    string `json:"BusProtocol"`
	Content        string `json:"Content"`
}

func list() ([]Device, error) {
	var l duList
	if err := diskutilJSON(&l, "list", "-plist", "physical"); err != nil {
		return nil, err
	}

	mounted := map[string]bool{}
	for _, d := range l.AllDisksAndPartitions {
		for _, p := range d.Partitions {
			if p.MountPoint != "" {
				mounted[d.DeviceIdentifier] = true
			}
		}
		for _, v := range d.APFSVolumes {
			if v.MountPoint != "" {
				mounted[d.DeviceIdentifier] = true
			}
		}
	}

	devs := make([]Device, 0, len(l.WholeDisks))
	for _, id := range l.WholeDisks {
		var info duInfo
		if err := diskutilJSON(&info, "info", "-plist", id); err != nil {
			// A card pulled between the list and the info call is normal;
			// skip it rather than failing the whole enumeration.
			continue
		}
		node := info.DeviceNode
		if node == "" {
			node = "/dev/" + id
		}
		devs = append(devs, Device{
			ID: id,
			// The raw character device skips the buffer cache and is an order
			// of magnitude faster for the large sequential reads we do.
			Path:      "/dev/r" + id,
			Node:      node,
			Name:      info.MediaName,
			Size:      info.TotalSize,
			SizeHuman: HumanSize(info.TotalSize),
			Removable: info.Removable || info.RemovableMedia,
			Internal:  info.Internal,
			Bus:       info.BusProtocol,
			Content:   info.Content,
			Mounted:   mounted[id],
		})
	}
	return devs, nil
}
