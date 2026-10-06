package imaging

import (
	"fmt"
	"os/exec"
	"strings"

	"bbsdcardsniffer/internal/device"
)

// prepareForWrite unmounts every volume of the card. macOS refuses to open a
// mounted disk for writing, and this is the supported way to free it.
//
// The returned function ejects the card once the transfer is done. That both
// guarantees the write has landed before it can be pulled and avoids the
// "disk not readable" dialog macOS raises when it re-scans a card carrying a
// filesystem it does not understand.
func prepareForWrite(d *device.Device) (func(), error) {
	out, err := exec.Command("diskutil", "unmountDisk", d.ID).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("could not unmount %s: %s", d.Node, strings.TrimSpace(string(out)))
	}
	return func() {
		// Best effort: a failure here costs the user a dialog, not their data.
		_ = exec.Command("diskutil", "eject", d.ID).Run()
	}, nil
}
