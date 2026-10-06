package device

import (
	"encoding/json"
	"os"
	"testing"
)

// TestListPrints is a smoke test for the platform enumeration: it asserts only
// that listing succeeds and returns plausible disks, and prints what it found
// under -v so a real card can be eyeballed.
func TestListPrints(t *testing.T) {
	devs, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(devs) == 0 {
		t.Fatal("no disks found; every machine has at least one")
	}
	for _, d := range devs {
		if d.Path == "" {
			t.Errorf("disk %s has no readable path", d.ID)
		}
		if d.Size <= 0 {
			t.Errorf("disk %s has size %d", d.ID, d.Size)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if testing.Verbose() {
		enc.Encode(devs)
	}
}
