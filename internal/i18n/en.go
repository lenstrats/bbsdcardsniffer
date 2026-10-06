package i18n

import "fmt"

var english = Messages{
	NoDeviceOpen: func() string { return "no device is open" },
	NoPermission: func(path string) string {
		return fmt.Sprintf("no permission to read %s: run this application as root or Administrator", path)
	},
	DeviceBusy: func(path string) string {
		return fmt.Sprintf("%s is in use: unmount its volumes first", path)
	},
	NoSuchVolume: func(index int) string {
		return fmt.Sprintf("there is no volume %d on this disk", index+1)
	},
	VolumeUnreadable: func(index int, kind string) string {
		return fmt.Sprintf("volume %d holds %s, which cannot be browsed", index+1, kind)
	},
	FSUnsupported: func(kind string) string {
		return fmt.Sprintf("%s cannot be read by this application", kind)
	},
	DiskReadOnly: func() string {
		return "this disk is open read-only; enable editing first"
	},
	CannotShorten: func(path string, size int) string {
		return fmt.Sprintf("%s: this filesystem cannot shorten a file, and the old contents past byte %d would remain", path, size)
	},

	UnmountUnsupported: func() string {
		return "unmounting from this application is only supported on macOS"
	},
	UnmountFailed: func(id, detail string) string {
		return fmt.Sprintf("could not unmount %s: %s", id, detail)
	},

	TransferRunning: func(kind string) string {
		return fmt.Sprintf("a %s is already running", kind)
	},
	NoTransferRunning: func() string { return "no transfer is running" },
	CloneNeedsBoth: func() string {
		return "copying needs both a source and a destination file"
	},
	WriteNeedsBoth: func() string { return "writing needs both an image and a card" },
	NoTrimTable: func(detail string) string {
		return fmt.Sprintf("cannot read the partition table to trim the copy: %s", detail)
	},
	BackupOverwrites: func() string {
		return "the backup cannot overwrite the image it is copying"
	},
	NotRemovableFixed: func(node, name string) string {
		return fmt.Sprintf("%s (%s) is a fixed disk: refusing to write to a non-removable disk", node, name)
	},
	NotRemovableSystem: func(node, name string) string {
		return fmt.Sprintf("%s (%s) is an internal disk: refusing to write to a non-removable disk", node, name)
	},
	NotRemovableUnknown: func(path string) string {
		return fmt.Sprintf("%s is not in the list of attached disks: refusing to write to it", path)
	},
	ImageTooLarge: func(needed, capacity int64) string {
		return fmt.Sprintf("the image is larger than the card: it needs more than %d bytes but the card holds %d", needed, capacity)
	},
	VerifyFailedClone: func(want, got string) string {
		return fmt.Sprintf("verification failed: the card hashed to %s but the image reads back as %s", want, got)
	},
	VerifyFailedWrite: func(want, got string) string {
		return fmt.Sprintf("verification failed: %s was written but the card reads back as %s", want, got)
	},

	NoAPIKey:      func() string { return "no Claude API key configured" },
	AssistantBusy: func() string { return "the assistant is still working on the previous question" },
	AssistantIdle: func() string { return "the assistant is not working on anything" },
	NoSuchTool:    func() string { return "no such tool" },
	OutOfToolTurns: func(rounds int) string {
		return fmt.Sprintf("gave up after %d rounds of tool calls without reaching an answer", rounds)
	},
}
