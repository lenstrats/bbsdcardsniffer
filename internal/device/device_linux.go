package device

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type lsblkDevice struct {
	Name       string        `json:"name"`
	Path       string        `json:"path"`
	Size       int64         `json:"size"`
	Model      string        `json:"model"`
	Tran       string        `json:"tran"`
	Rm         bool          `json:"rm"`
	Hotplug    bool          `json:"hotplug"`
	Type       string        `json:"type"`
	Pttype     string        `json:"pttype"`
	Mountpoint string        `json:"mountpoint"`
	Children   []lsblkDevice `json:"children"`
}

func (d lsblkDevice) anyMounted() bool {
	if d.Mountpoint != "" {
		return true
	}
	for _, c := range d.Children {
		if c.anyMounted() {
			return true
		}
	}
	return false
}

func list() ([]Device, error) {
	// -b gives sizes in bytes, -J gives JSON, and the explicit column list
	// keeps the output stable across util-linux versions.
	cmd := exec.Command("lsblk", "-J", "-b", "-o",
		"NAME,PATH,SIZE,MODEL,TRAN,RM,HOTPLUG,TYPE,PTTYPE,MOUNTPOINT")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("lsblk: %v: %s", err, strings.TrimSpace(stderr.String()))
	}

	var parsed struct {
		Blockdevices []lsblkDevice `json:"blockdevices"`
	}
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parse lsblk output: %w", err)
	}

	var devs []Device
	for _, d := range parsed.Blockdevices {
		if d.Type != "disk" {
			continue
		}
		path := d.Path
		if path == "" {
			path = "/dev/" + d.Name
		}
		name := strings.TrimSpace(d.Model)
		if name == "" {
			name = d.Name
		}
		devs = append(devs, Device{
			ID:        d.Name,
			Path:      path,
			Node:      path,
			Name:      name,
			Size:      d.Size,
			SizeHuman: HumanSize(d.Size),
			Removable: d.Rm || d.Hotplug,
			Internal:  !(d.Rm || d.Hotplug),
			Bus:       d.Tran,
			Content:   d.Pttype,
			Mounted:   d.anyMounted(),
		})
	}
	return devs, nil
}
