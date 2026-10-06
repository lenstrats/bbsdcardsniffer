package volume

import "testing"

// blob builds a header with the given magic bytes written at fixed offsets, so
// each probe can be exercised without a real filesystem behind it.
func blob(parts map[int]string) []byte {
	buf := make([]byte, headerSize)
	for off, s := range parts {
		copy(buf[off:], s)
	}
	return buf
}

func TestProbeIdentifiesKinds(t *testing.T) {
	tests := []struct {
		name      string
		buf       []byte
		wantKind  string
		wantLabel string
	}{
		{
			name: "ext4 via the extents feature flag",
			buf: blob(map[int]string{
				extMagicOff:        "\x53\xef",
				extFeatureIncompat: "\x40\x00\x00\x00",
				extLabelOff:        "rootfs\x00",
			}),
			wantKind:  "ext4",
			wantLabel: "rootfs",
		},
		{
			name: "ext3 has a journal but no ext4 features",
			buf: blob(map[int]string{
				extMagicOff:      "\x53\xef",
				extFeatureCompat: "\x04\x00\x00\x00",
			}),
			wantKind: "ext3",
		},
		{
			name:     "ext2 has neither",
			buf:      blob(map[int]string{extMagicOff: "\x53\xef"}),
			wantKind: "ext2",
		},
		{
			name:      "btrfs superblock sits at 0x10000",
			buf:       blob(map[int]string{0x10040: "_BHRfS_M", 0x1012B: "media\x00"}),
			wantKind:  "btrfs",
			wantLabel: "media",
		},
		{
			name:     "LUKS2 is told apart from LUKS1 by its version field",
			buf:      blob(map[int]string{0: "LUKS\xba\xbe\x00\x02", 0x18: "secret\x00"}),
			wantKind: "luks2", wantLabel: "secret",
		},
		{
			name:     "LUKS1",
			buf:      blob(map[int]string{0: "LUKS\xba\xbe\x00\x01"}),
			wantKind: "luks1",
		},
		{
			name:     "swap signature lives at the end of the first page",
			buf:      blob(map[int]string{4096 - 10: "SWAPSPACE2"}),
			wantKind: "swap",
		},
		{
			name:      "fat32 needs its boot signature",
			buf:       blob(map[int]string{0x52: "FAT32   ", 0x47: "BOOT\x00", 510: "\x55\xaa"}),
			wantKind:  "fat32",
			wantLabel: "BOOT",
		},
		{
			name:     "xfs",
			buf:      blob(map[int]string{0: "XFSB"}),
			wantKind: "xfs",
		},
		{
			name:     "lvm2 physical volume",
			buf:      blob(map[int]string{512: "LABELONE"}),
			wantKind: "lvm2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got FSInfo
			for _, check := range checks {
				if info, ok := check(tt.buf); ok {
					got = info
					break
				}
			}
			if got.Kind != tt.wantKind {
				t.Errorf("kind = %q, want %q", got.Kind, tt.wantKind)
			}
			if tt.wantLabel != "" && got.Label != tt.wantLabel {
				t.Errorf("label = %q, want %q", got.Label, tt.wantLabel)
			}
		})
	}
}

// A blank region must not be claimed by any probe, or the UI would report a
// bogus filesystem for unallocated space.
func TestProbeRejectsBlankRegion(t *testing.T) {
	for _, check := range checks {
		if info, ok := check(make([]byte, headerSize)); ok {
			t.Errorf("blank region matched as %q", info.Kind)
		}
	}
}

// FAT text without the 0x55AA boot signature is not a FAT volume.
func TestProbeFATNeedsBootSignature(t *testing.T) {
	if info, ok := probeFAT(blob(map[int]string{0x36: "FAT16   "})); ok {
		t.Errorf("matched %q without a boot signature", info.Kind)
	}
}
