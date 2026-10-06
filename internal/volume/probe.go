package volume

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"strings"
)

// headerSize is how much of a partition we read to identify it. The furthest
// signature we look at is the btrfs label at 0x1012B, so 128 KiB is ample.
const headerSize = 128 * 1024

// FSInfo is what a signature scan can tell us about a partition without
// needing a filesystem driver for it.
type FSInfo struct {
	// Kind is a lowercase filesystem identifier: ext4, btrfs, luks, ...
	// Empty when nothing recognisable was found.
	Kind string `json:"kind"`
	// Label is the volume label, when the format stores one where we can see it.
	Label string `json:"label"`
}

// readable lists the kinds go-diskfs can actually mount and browse. Everything
// else is identified for the user's benefit but cannot be opened.
var readable = map[string]bool{
	"ext2":     true,
	"ext3":     true,
	"ext4":     true,
	"fat12":    true,
	"fat16":    true,
	"fat32":    true,
	"iso9660":  true,
	"squashfs": true,
}

// probe identifies the filesystem at the start of r by its on-disk signature.
func probe(r io.ReaderAt) (FSInfo, error) {
	buf := make([]byte, headerSize)
	n, err := r.ReadAt(buf, 0)
	if err != nil && !errors.Is(err, io.EOF) && n == 0 {
		return FSInfo{}, err
	}
	buf = buf[:n]

	for _, check := range checks {
		if info, ok := check(buf); ok {
			return info, nil
		}
	}
	return FSInfo{}, nil
}

// at returns the size bytes starting at off, or nil if the buffer is too short.
func at(buf []byte, off, size int) []byte {
	if off < 0 || size < 0 || off+size > len(buf) {
		return nil
	}
	return buf[off : off+size]
}

// magic reports whether the bytes at off equal want.
func magic(buf []byte, off int, want string) bool {
	return bytes.Equal(at(buf, off, len(want)), []byte(want))
}

// cstr trims a fixed-width label field to its printable content.
func cstr(b []byte) string {
	if i := bytes.IndexByte(b, 0); i >= 0 {
		b = b[:i]
	}
	return strings.TrimSpace(string(b))
}

// checks are tried in order; the first match wins. Container formats (LUKS,
// LVM) come first because what they wrap must not be matched instead.
var checks = []func([]byte) (FSInfo, bool){
	probeLUKS,
	probeLVM2,
	probeExt,
	probeBtrfs,
	probeXFS,
	probeF2FS,
	probeSwap,
	probeSquashfs,
	probeXNU,
	probeNTFSFamily,
	probeFAT,
	probeISO9660,
}

func probeLUKS(buf []byte) (FSInfo, bool) {
	if !magic(buf, 0, "LUKS\xba\xbe") {
		return FSInfo{}, false
	}
	// LUKS2 keeps an optional label; LUKS1 has none.
	version := binary.BigEndian.Uint16(at(buf, 6, 2))
	info := FSInfo{Kind: "luks"}
	if version == 2 {
		info.Kind = "luks2"
		info.Label = cstr(at(buf, 0x18, 48))
	} else {
		info.Kind = "luks1"
	}
	return info, true
}

func probeLVM2(buf []byte) (FSInfo, bool) {
	// The LVM2 label sits in one of the first four sectors.
	for _, sector := range []int{0, 512, 1024, 1536} {
		if magic(buf, sector, "LABELONE") {
			return FSInfo{Kind: "lvm2"}, true
		}
	}
	return FSInfo{}, false
}

// ext superblock constants, all relative to the superblock at offset 0x400.
const (
	extSB              = 0x400
	extMagicOff        = extSB + 0x38
	extFeatureCompat   = extSB + 0x5C
	extFeatureIncompat = extSB + 0x60
	extLabelOff        = extSB + 0x78

	extCompatHasJournal = 0x0004
	// Any of these being set means the volume is ext4 rather than ext3.
	extIncompatExtents = 0x0040
	extIncompat64Bit   = 0x0080
	extIncompatFlexBG  = 0x0200
)

func probeExt(buf []byte) (FSInfo, bool) {
	m := at(buf, extMagicOff, 2)
	if m == nil || binary.LittleEndian.Uint16(m) != 0xEF53 {
		return FSInfo{}, false
	}
	compat := binary.LittleEndian.Uint32(at(buf, extFeatureCompat, 4))
	incompat := binary.LittleEndian.Uint32(at(buf, extFeatureIncompat, 4))

	kind := "ext2"
	switch {
	case incompat&(extIncompatExtents|extIncompat64Bit|extIncompatFlexBG) != 0:
		kind = "ext4"
	case compat&extCompatHasJournal != 0:
		kind = "ext3"
	}
	return FSInfo{Kind: kind, Label: cstr(at(buf, extLabelOff, 16))}, true
}

func probeBtrfs(buf []byte) (FSInfo, bool) {
	// btrfs puts its primary superblock at 0x10000, magic 0x40 bytes in.
	if !magic(buf, 0x10040, "_BHRfS_M") {
		return FSInfo{}, false
	}
	return FSInfo{Kind: "btrfs", Label: cstr(at(buf, 0x1012B, 256))}, true
}

func probeXFS(buf []byte) (FSInfo, bool) {
	if !magic(buf, 0, "XFSB") {
		return FSInfo{}, false
	}
	return FSInfo{Kind: "xfs", Label: cstr(at(buf, 0x6C, 12))}, true
}

func probeF2FS(buf []byte) (FSInfo, bool) {
	m := at(buf, 0x400, 4)
	if m == nil || binary.LittleEndian.Uint32(m) != 0xF2F52010 {
		return FSInfo{}, false
	}
	return FSInfo{Kind: "f2fs"}, true
}

func probeSwap(buf []byte) (FSInfo, bool) {
	// The signature lives in the last ten bytes of the first page, and the
	// page size is not fixed, so check the sizes actually used in practice.
	for _, pageSize := range []int{4096, 8192, 16384, 65536} {
		off := pageSize - 10
		if magic(buf, off, "SWAPSPACE2") || magic(buf, off, "SWAP-SPACE") {
			return FSInfo{Kind: "swap", Label: cstr(at(buf, 0x41C, 16))}, true
		}
	}
	return FSInfo{}, false
}

func probeSquashfs(buf []byte) (FSInfo, bool) {
	if magic(buf, 0, "hsqs") || magic(buf, 0, "sqsh") {
		return FSInfo{Kind: "squashfs"}, true
	}
	return FSInfo{}, false
}

func probeXNU(buf []byte) (FSInfo, bool) {
	switch {
	case magic(buf, 0x20, "NXSB"):
		return FSInfo{Kind: "apfs"}, true
	case magic(buf, 0x400, "H+"), magic(buf, 0x400, "HX"):
		return FSInfo{Kind: "hfsplus"}, true
	}
	return FSInfo{}, false
}

func probeNTFSFamily(buf []byte) (FSInfo, bool) {
	switch {
	case magic(buf, 3, "NTFS    "):
		return FSInfo{Kind: "ntfs"}, true
	case magic(buf, 3, "EXFAT   "):
		return FSInfo{Kind: "exfat"}, true
	}
	return FSInfo{}, false
}

func probeFAT(buf []byte) (FSInfo, bool) {
	// Without the boot signature this is not a FAT boot sector at all, which
	// keeps us from matching stray ASCII in some other format's header.
	if sig := at(buf, 510, 2); sig == nil || sig[0] != 0x55 || sig[1] != 0xAA {
		return FSInfo{}, false
	}
	// FAT32 moves the type string and the label out to the extended BPB.
	if magic(buf, 0x52, "FAT32   ") {
		return FSInfo{Kind: "fat32", Label: cstr(at(buf, 0x47, 11))}, true
	}
	label := cstr(at(buf, 0x2B, 11))
	switch {
	case magic(buf, 0x36, "FAT12   "):
		return FSInfo{Kind: "fat12", Label: label}, true
	case magic(buf, 0x36, "FAT16   "):
		return FSInfo{Kind: "fat16", Label: label}, true
	case magic(buf, 0x36, "FAT     "):
		// Plain "FAT" leaves the width to the cluster count, which we do not
		// parse here; fat16 is the safe guess and go-diskfs re-detects anyway.
		return FSInfo{Kind: "fat16", Label: label}, true
	}
	return FSInfo{}, false
}

func probeISO9660(buf []byte) (FSInfo, bool) {
	if !magic(buf, 0x8001, "CD001") {
		return FSInfo{}, false
	}
	return FSInfo{Kind: "iso9660", Label: cstr(at(buf, 0x8028, 32))}, true
}
