// Package imaging copies whole devices: a card to an image file, and later an
// image file back onto a card. It is separate from the browsing code on
// purpose — internal/blockdev is read-only by construction, and anything that
// writes to a device has to be reached through this package explicitly.
package imaging

import (
	"sync"
	"time"
)

// Phase names the stage a transfer is in, so the UI can label the bar.
type Phase string

const (
	PhaseCopying   Phase = "copying"
	PhaseVerifying Phase = "verifying"
)

// Progress is one update during a transfer.
type Progress struct {
	Phase Phase `json:"phase"`
	Bytes int64 `json:"bytes"`
	Total int64 `json:"total"`
	// BytesPerSecond is measured over the recent past, not the whole run, so
	// the estimate reacts when a card slows down.
	BytesPerSecond float64 `json:"bytesPerSecond"`
	// ETASeconds is negative when there is not enough history to estimate.
	ETASeconds float64 `json:"etaSeconds"`
}

// ProgressFunc receives updates. It is called from the transfer goroutine and
// must not block for long.
type ProgressFunc func(Progress)

// reporter throttles updates to a readable rate and computes the running speed
// from a short trailing window.
type reporter struct {
	fn       ProgressFunc
	total    int64
	interval time.Duration

	mu       sync.Mutex
	phase    Phase
	last     time.Time
	lastByte int64
	speed    float64
}

// minInterval is how often the UI is updated. Faster than this just burns
// bridge calls without the user seeing anything more.
const minInterval = 200 * time.Millisecond

func newReporter(fn ProgressFunc, total int64, now time.Time) *reporter {
	return &reporter{fn: fn, total: total, interval: minInterval, last: now, phase: PhaseCopying}
}

func (r *reporter) setPhase(p Phase) {
	r.mu.Lock()
	r.phase = p
	r.mu.Unlock()
}

// report emits an update if enough time has passed, or if force is set, which
// the caller uses for the final update of a phase.
func (r *reporter) report(done int64, now time.Time, force bool) {
	if r.fn == nil {
		return
	}
	r.mu.Lock()
	elapsed := now.Sub(r.last)
	if !force && elapsed < r.interval {
		r.mu.Unlock()
		return
	}
	if elapsed > 0 {
		instant := float64(done-r.lastByte) / elapsed.Seconds()
		if r.speed == 0 {
			r.speed = instant
		} else {
			// Exponential smoothing; enough to stop the estimate jittering
			// without making it slow to react.
			r.speed = 0.7*r.speed + 0.3*instant
		}
	}
	r.last = now
	r.lastByte = done

	eta := -1.0
	if r.speed > 0 && r.total > 0 {
		eta = float64(r.total-done) / r.speed
	}
	p := Progress{
		Phase:          r.phase,
		Bytes:          done,
		Total:          r.total,
		BytesPerSecond: r.speed,
		ETASeconds:     eta,
	}
	r.mu.Unlock()

	r.fn(p)
}

// Result describes a finished transfer.
type Result struct {
	// Bytes is the number of device bytes read or written, before any
	// compression is applied.
	Bytes int64 `json:"bytes"`
	// SHA256 is over those same uncompressed bytes, so it can be compared
	// against the device regardless of how the image is stored.
	SHA256 string `json:"sha256"`
	// FileBytes is the size on disk, which differs from Bytes when compressed.
	FileBytes int64   `json:"fileBytes"`
	Seconds   float64 `json:"seconds"`
	Path      string  `json:"path"`
	// Verified is set when a read-back comparison ran and matched.
	Verified bool `json:"verified"`
}
