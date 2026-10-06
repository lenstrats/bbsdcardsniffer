// Package i18n translates the messages the backend produces.
//
// The frontend has its own catalogues for the interface itself; these are the
// texts only the Go side knows how to phrase — why a disk would not open, why
// a write is refused, what a transfer ran into. The two sets do not overlap,
// so nothing is translated twice.
//
// Deliberately not translated: errors that come out of the filesystem drivers
// ("invalid checksum type 0", "resource busy"). They are diagnostic rather
// than actionable, they are searchable as they stand, and rewording a
// third-party library's message would only make it harder to look up. Those
// travel as detail alongside a translated explanation.
package i18n

import "sync/atomic"

// Locale is a language tag. The set matches the frontend's.
type Locale string

const (
	English Locale = "en"
	Dutch   Locale = "nl"
	German  Locale = "de"
)

// Messages is every text the backend can produce. Fields are functions so a
// translation decides its own word order rather than filling in blanks in an
// English sentence.
//
// Completeness is enforced by TestEveryLocaleIsComplete, which reflects over
// this struct: a field left unset in any locale fails the build's test run.
type Messages struct {
	// Opening a medium.
	NoDeviceOpen     func() string
	NoPermission     func(path string) string
	DeviceBusy       func(path string) string
	NoSuchVolume     func(index int) string
	VolumeUnreadable func(index int, kind string) string
	FSUnsupported    func(kind string) string
	DiskReadOnly     func() string
	CannotShorten    func(path string, size int) string

	// Unmounting.
	UnmountUnsupported func() string
	UnmountFailed      func(id, detail string) string

	// Transfers.
	TransferRunning     func(kind string) string
	NoTransferRunning   func() string
	CloneNeedsBoth      func() string
	WriteNeedsBoth      func() string
	NoTrimTable         func(detail string) string
	BackupOverwrites    func() string
	NotRemovableFixed   func(node, name string) string
	NotRemovableSystem  func(node, name string) string
	NotRemovableUnknown func(path string) string
	ImageTooLarge       func(needed, capacity int64) string
	VerifyFailedClone   func(want, got string) string
	VerifyFailedWrite   func(want, got string) string

	// Assistant.
	NoAPIKey       func() string
	AssistantBusy  func() string
	AssistantIdle  func() string
	NoSuchTool     func() string
	OutOfToolTurns func(rounds int) string
}

// current holds the active locale. Requests can arrive from several frontend
// calls at once, so it is swapped atomically rather than guarded by a mutex.
var current atomic.Pointer[Messages]

func init() {
	m := english
	current.Store(&m)
}

// catalogues is every locale that ships, by tag. English is the default and
// the fallback for anything not listed.
var catalogues = map[Locale]Messages{
	English: english,
	Dutch:   dutch,
	German:  german,
}

// Set switches the active locale. An unknown tag falls back to English rather
// than failing: a missing translation should never break the application.
func Set(locale Locale) {
	m, ok := catalogues[locale]
	if !ok {
		m = english
	}
	current.Store(&m)
}

// T returns the active catalogue.
func T() *Messages { return current.Load() }
