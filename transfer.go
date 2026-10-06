package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"bbsdcardsniffer/internal/device"
	"bbsdcardsniffer/internal/i18n"
	"bbsdcardsniffer/internal/imaging"
	"bbsdcardsniffer/internal/volume"
)

// progressEvent is the name the frontend listens on for transfer updates.
const progressEvent = "transfer:progress"

// transfers holds the cancel function of the one transfer allowed at a time.
// Cloning a card while writing another would fight over the same device list
// and confuse the progress bar for no good reason.
type transfers struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	kind   string
}

// begin registers a new transfer, refusing if one is already running.
func (t *transfers) begin(ctx context.Context, kind string) (context.Context, func(), error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cancel != nil {
		return nil, nil, errors.New(i18n.T().TransferRunning(t.kind))
	}
	if ctx == nil {
		// Only reachable before the Wails runtime has started, which in
		// practice means a test; a background parent keeps cancellation working.
		ctx = context.Background()
	}
	child, cancel := context.WithCancel(ctx)
	t.cancel, t.kind = cancel, kind
	return child, func() {
		t.mu.Lock()
		defer t.mu.Unlock()
		cancel()
		t.cancel, t.kind = nil, ""
	}, nil
}

// CancelTransfer stops the running clone or write, if there is one.
func (a *App) CancelTransfer() error {
	a.transfers.mu.Lock()
	defer a.transfers.mu.Unlock()
	if a.transfers.cancel == nil {
		return errors.New(i18n.T().NoTransferRunning())
	}
	a.transfers.cancel()
	return nil
}

// CloneOptions is what the UI sends when starting a clone.
type CloneOptions struct {
	// Device is the path to clone, as listed by ListDevices.
	Device string `json:"device"`
	// Trim stops after the last partition instead of copying the whole card.
	Trim bool `json:"trim"`
	// Compress writes a gzip image.
	Compress bool `json:"compress"`
	// Verify re-reads the image afterwards and compares hashes.
	Verify bool `json:"verify"`
}

// CloneDevice copies a card to an image file the user picks. It returns nil if
// the user cancels the save dialog.
func (a *App) CloneDevice(opts CloneOptions) (*imaging.Result, error) {
	if opts.Device == "" {
		return nil, errors.New(i18n.T().CloneNeedsBoth())
	}

	// Work out where to stop before asking for a filename, so a card we
	// cannot read the partition table of fails before the user picks a path.
	limit, err := cloneLimit(opts.Device, opts.Trim)
	if err != nil {
		return nil, err
	}

	dest, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Save card image",
		DefaultFilename: defaultImageName(opts.Device, opts.Compress),
	})
	if err != nil {
		return nil, err
	}
	if dest == "" {
		return nil, nil
	}

	ctx, done, err := a.transfers.begin(a.ctx, "clone")
	if err != nil {
		return nil, err
	}
	defer done()

	return imaging.Clone(ctx, imaging.CloneOptions{
		Source: opts.Device,
		Dest:   dest,
		Limit:  limit,
		Gzip:   opts.Compress,
		Verify: opts.Verify,
	}, func(p imaging.Progress) {
		wruntime.EventsEmit(a.ctx, progressEvent, p)
	})
}

// cloneLimit returns the byte count to stop the clone at, or 0 for the whole
// device. Reading the partition table needs its own handle, which is fine
// since everything here opens read-only.
func cloneLimit(devPath string, trim bool) (int64, error) {
	if !trim {
		return 0, nil
	}
	s, err := volume.Open(devPath)
	if err != nil {
		return 0, errors.New(i18n.T().NoTrimTable(err.Error()))
	}
	defer s.Close()
	return s.DataEnd(), nil
}

// mediumName reduces a device path or image path to a bare, readable stem.
//
// filepath is not used here: it follows the separator of the platform the
// binary was built for, so a Windows drive path would not be split correctly
// when this logic is reasoned about or tested elsewhere. Both separators are
// therefore handled explicitly.
func mediumName(p string) string {
	stem := p
	if i := strings.LastIndexAny(stem, `/\`); i >= 0 {
		stem = stem[i+1:]
	}
	stem = strings.Trim(stem, `.\/ `)

	// macOS exposes the raw character device as rdiskN; the N is what the user
	// recognises. Only that exact shape is rewritten, so an image called
	// rootfs.img keeps its name.
	if rest, ok := strings.CutPrefix(stem, "rdisk"); ok && rest != "" && isDigits(rest) {
		stem = "disk" + rest
	}

	// Strip a trailing image or compression extension, repeatedly, so
	// card.img.gz becomes card. Anything unrecognised is left alone.
	for {
		lower := strings.ToLower(stem)
		trimmed := false
		for _, ext := range []string{".gz", ".xz", ".bz2", ".zip", ".img", ".iso", ".bin", ".raw", ".dmg"} {
			if strings.HasSuffix(lower, ext) && len(stem) > len(ext) {
				stem = stem[:len(stem)-len(ext)]
				trimmed = true
				break
			}
		}
		if !trimmed {
			break
		}
	}

	if stem == "" {
		return "card"
	}
	return stem
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}

// defaultImageName suggests a filename based on the device it came from.
func defaultImageName(devPath string, compress bool) string {
	name := mediumName(devPath)
	if compress {
		return name + ".img.gz"
	}
	return name + ".img"
}

// TrimInfo tells the UI how much a trimmed clone would save, so the choice is
// an informed one rather than a blind checkbox.
type TrimInfo struct {
	// DataEnd is where the last partition ends.
	DataEnd      int64  `json:"dataEnd"`
	DataEndHuman string `json:"dataEndHuman"`
	// Total is the full device size.
	Total      int64  `json:"total"`
	TotalHuman string `json:"totalHuman"`
	// Table is the partition scheme, which decides whether trimming is lossless.
	Table string `json:"table"`
	// LosesBackupGPT is true when trimming would cut off the backup partition
	// table that GPT keeps in the last sectors of the device.
	LosesBackupGPT bool `json:"losesBackupGpt"`
}

// InspectForClone reports what a clone of this device would look like.
func (a *App) InspectForClone(devPath string) (*TrimInfo, error) {
	s, err := volume.Open(devPath)
	if err != nil {
		return nil, describeOpenError(devPath, err)
	}
	defer s.Close()

	end := s.DataEnd()
	return &TrimInfo{
		DataEnd:        end,
		DataEndHuman:   device.HumanSize(end),
		Total:          s.Size,
		TotalHuman:     s.SizeHuman,
		Table:          s.Table,
		LosesBackupGPT: s.Table == "gpt" && end < s.Size,
	}, nil
}

// ImageInfo describes an image the user picked, so the UI can show what is
// about to be written before the destructive step is offered.
type ImageInfo struct {
	Path      string `json:"path"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	SizeHuman string `json:"sizeHuman"`
	// Format is how the image is stored: raw, gzip, xz, bzip2 or zip.
	Format string `json:"format"`
	// Written is how many bytes will land on the card, or 0 when that cannot
	// be known without decompressing the image first.
	Written      int64  `json:"written"`
	WrittenHuman string `json:"writtenHuman"`
}

// ChooseImage opens a file picker and reports what was selected. It returns
// nil when the user cancels.
func (a *App) ChooseImage() (*ImageInfo, error) {
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title: "Choose an image to write",
		Filters: []wruntime.FileFilter{
			{DisplayName: "Disk images (*.img, *.iso, *.gz, *.xz, *.bz2, *.zip)", Pattern: "*.img;*.iso;*.bin;*.raw;*.gz;*.xz;*.bz2;*.zip"},
			{DisplayName: "All files", Pattern: "*"},
		},
	})
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	format, err := imaging.DetectFormat(f)
	if err != nil {
		return nil, err
	}

	written := imaging.UncompressedSize(f, format, info.Size())
	out := &ImageInfo{
		Path:      path,
		Name:      filepath.Base(path),
		Size:      info.Size(),
		SizeHuman: device.HumanSize(info.Size()),
		Format:    string(format),
		Written:   written,
	}
	if written > 0 {
		out.WrittenHuman = device.HumanSize(written)
	}
	return out, nil
}

// WriteRequest is what the UI sends to start a write.
type WriteRequest struct {
	// Image is a path returned by ChooseImage.
	Image string `json:"image"`
	// Device is the card to write to. It must be removable; the imaging
	// package refuses anything else regardless of what is sent here.
	Device string `json:"device"`
	Verify bool   `json:"verify"`
}

// WriteImage writes an image onto a card, replacing everything on it.
func (a *App) WriteImage(req WriteRequest) (*imaging.Result, error) {
	if req.Image == "" || req.Device == "" {
		return nil, errors.New(i18n.T().WriteNeedsBoth())
	}

	// Refuse an unacceptable target before anything else happens. The check
	// runs again inside imaging.Write, which is what actually guarantees
	// safety; doing it here as well means a bad target does not cost the user
	// their open session.
	if err := imaging.CheckTarget(req.Device); err != nil {
		return nil, err
	}

	// Release our own read handle: the card has to be unmounted for the write,
	// and holding a session open on it only gets in the way.
	if err := a.CloseDevice(); err != nil {
		return nil, err
	}

	ctx, done, err := a.transfers.begin(a.ctx, "write")
	if err != nil {
		return nil, err
	}
	defer done()

	return imaging.Write(ctx, imaging.WriteOptions{
		Source: req.Image,
		Dest:   req.Device,
		Verify: req.Verify,
	}, func(p imaging.Progress) {
		wruntime.EventsEmit(a.ctx, progressEvent, p)
	})
}

// BackupCurrent clones whatever is currently open — a card or an image file —
// to a location the user picks. It exists so the safety step before enabling
// edits is one button rather than a trip through another part of the
// interface, and it deliberately copies the whole medium rather than trimming:
// a backup taken to guard against a mistake should not itself be lossy.
//
// It returns nil if the user cancels the save dialog.
func (a *App) BackupCurrent() (*imaging.Result, error) {
	a.mu.Lock()
	session := a.session
	a.mu.Unlock()

	if session == nil {
		return nil, errors.New(i18n.T().NoDeviceOpen())
	}
	source := session.Path

	dest, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "Save a backup before editing",
		DefaultFilename: backupName(source),
	})
	if err != nil {
		return nil, err
	}
	if dest == "" {
		return nil, nil
	}
	if dest == source {
		return nil, errors.New(i18n.T().BackupOverwrites())
	}

	ctx, done, err := a.transfers.begin(a.ctx, "backup")
	if err != nil {
		return nil, err
	}
	defer done()

	res, err := imaging.Clone(ctx, imaging.CloneOptions{
		Source: source,
		Dest:   dest,
		Verify: true,
	}, func(p imaging.Progress) {
		wruntime.EventsEmit(a.ctx, progressEvent, p)
	})
	if err != nil {
		return nil, err
	}

	// Remember it so the interface can stop nagging about this medium.
	a.mu.Lock()
	if a.backedUp == nil {
		a.backedUp = map[string]string{}
	}
	a.backedUp[source] = dest
	a.mu.Unlock()

	return res, nil
}

// backupName suggests a filename for a backup of the given source.
func backupName(source string) string {
	return mediumName(source) + "-backup.img"
}

// BackupPath reports where the backup of the open medium was written this
// session, or "" if none was made.
func (a *App) BackupPath() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return ""
	}
	return a.backedUp[a.session.Path]
}

// TargetInfo describes what is currently on a card that is about to be
// overwritten, so the confirmation can name what will be lost instead of
// asking about an anonymous device.
type TargetInfo struct {
	Path      string          `json:"path"`
	Node      string          `json:"node"`
	Size      int64           `json:"size"`
	SizeHuman string          `json:"sizeHuman"`
	Table     string          `json:"table"`
	Volumes   []volume.Volume `json:"volumes"`
	// Empty reports that nothing recognisable is on the card: no partition
	// table and no filesystem signature.
	Empty bool `json:"empty"`
	// Unreadable is set when the card's contents could not be inspected at
	// all — no permission, or a card that will not read. The interface then
	// asks for confirmation anyway rather than implying the card is blank.
	Unreadable bool `json:"unreadable"`
	// Detail carries the reason when Unreadable is set.
	Detail string `json:"detail,omitempty"`
}

// InspectTarget reports what is on the card selected as a write target.
//
// Failure here is not an error: a card that cannot be read is precisely the
// case where the user should still be warned, so the result says so instead.
func (a *App) InspectTarget(devPath string) *TargetInfo {
	info := &TargetInfo{Path: devPath, Node: devPath}

	// The device list has the friendly name and the size, and works even when
	// the card's contents do not.
	if devices, err := device.List(); err == nil {
		for _, d := range devices {
			if d.Path == devPath || d.Node == devPath {
				info.Node = d.Node
				info.Size = d.Size
				info.SizeHuman = d.SizeHuman
				break
			}
		}
	}

	s, err := volume.Open(devPath)
	if err != nil {
		info.Unreadable = true
		info.Detail = err.Error()
		return info
	}
	defer s.Close()

	info.Table = s.Table
	info.Volumes = s.Volumes
	if info.Size == 0 {
		info.Size = s.Size
		info.SizeHuman = s.SizeHuman
	}

	// A card counts as empty only when there is no partition table and no
	// volume carries a filesystem signature. Anything else is data someone
	// may still want.
	info.Empty = s.Table == "none"
	for _, v := range s.Volumes {
		if v.FS.Kind != "" {
			info.Empty = false
			break
		}
	}
	return info
}
