package imaging

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"bbsdcardsniffer/internal/blockdev"
	"bbsdcardsniffer/internal/device"
	"bbsdcardsniffer/internal/i18n"
)

// writeAlignment is what the tail of a write is padded to. Raw devices reject
// partial-sector writes, and 4096 covers both sector sizes in use.
const writeAlignment = 4096

// WriteOptions describes an image-to-card write.
type WriteOptions struct {
	// Source is the image file, compressed or not.
	Source string
	// Dest is the device to write to. It must be a removable disk.
	Dest string
	// Verify reads the card back afterwards and compares hashes.
	Verify bool
}

// ErrNotRemovable is returned when the target is a fixed or system disk.
// Writing to one destroys it, so this package refuses outright rather than
// leaving the decision to a confirmation dialog that can be clicked through.
var ErrNotRemovable = errors.New("refusing to write to a non-removable disk")

// Write copies an image onto a card. It refuses any target that is not a
// removable disk, unmounts the card first, and optionally reads it back to
// confirm what landed there.
func Write(ctx context.Context, opts WriteOptions, report ProgressFunc) (*Result, error) {
	if opts.Source == "" || opts.Dest == "" {
		return nil, errors.New(i18n.T().WriteNeedsBoth())
	}

	target, err := resolveRemovable(opts.Dest)
	if err != nil {
		return nil, err
	}

	img, err := os.Open(opts.Source)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", opts.Source, err)
	}
	defer img.Close()

	info, err := img.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := info.Size()

	format, err := DetectFormat(img)
	if err != nil {
		return nil, err
	}

	// Unmount or lock the card before anything is opened for writing.
	release, err := prepareForWrite(target)
	if err != nil {
		return nil, err
	}
	defer release()

	src, err := openDecompressed(img, format, fileSize)
	if err != nil {
		return nil, err
	}
	defer src.closer.Close()

	start := time.Now()
	rep := newReporter(report, src.total, start)
	digest := sha256.New()

	written, err := writeToDevice(ctx, target.Path, src, digest, target.Size, rep)
	if err != nil {
		return nil, err
	}

	sum := hex.EncodeToString(digest.Sum(nil))
	result := &Result{
		Bytes:     written,
		SHA256:    sum,
		FileBytes: fileSize,
		Seconds:   time.Since(start).Seconds(),
		Path:      target.Node,
	}

	if opts.Verify {
		rep.setPhase(PhaseVerifying)
		rep.total = written
		readBack, err := hashDevice(ctx, target.Path, written, rep)
		if err != nil {
			return nil, fmt.Errorf("verify %s: %w", target.Node, err)
		}
		if readBack != sum {
			return nil, errors.New(i18n.T().VerifyFailedWrite(sum, readBack))
		}
		result.Verified = true
		result.Seconds = time.Since(start).Seconds()
	}

	return result, nil
}

// CheckTarget reports whether path may be written to, without starting
// anything. Callers use it to refuse a bad target before they disturb any
// state; Write performs the same check again, so this is a convenience, never
// the thing that makes writing safe.
func CheckTarget(path string) error {
	_, err := resolveRemovable(path)
	return err
}

// resolveRemovable looks the target up in the device list and refuses anything
// that is not removable. Failing to find it is also a refusal: an unknown
// device is not one we are willing to overwrite.
func resolveRemovable(path string) (*device.Device, error) {
	devices, err := device.List()
	if err != nil {
		return nil, fmt.Errorf("cannot check whether %s is removable: %w", path, err)
	}
	for i := range devices {
		d := &devices[i]
		if d.Path != path && d.Node != path {
			continue
		}
		if !d.Removable {
			return nil, fmt.Errorf("%s: %w", i18n.T().NotRemovableFixed(d.Node, d.Name), ErrNotRemovable)
		}
		if d.Internal {
			return nil, fmt.Errorf("%s: %w", i18n.T().NotRemovableSystem(d.Node, d.Name), ErrNotRemovable)
		}
		return d, nil
	}
	return nil, fmt.Errorf("%s: %w", i18n.T().NotRemovableUnknown(path), ErrNotRemovable)
}

// writeToDevice streams src onto the device, padding the final block out to a
// sector boundary because raw devices will not take a partial one.
func writeToDevice(ctx context.Context, path string, src *source, digest io.Writer, capacity int64, rep *reporter) (int64, error) {
	out, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return 0, fmt.Errorf("open %s for writing: %w", path, err)
	}
	defer out.Close()

	buf := make([]byte, blockSize)
	var written int64

	for {
		if err := ctx.Err(); err != nil {
			return written, err
		}

		n, readErr := io.ReadFull(src.reader, buf)
		if n > 0 {
			payload := buf[:n]
			digest.Write(payload)

			// Pad the tail so the write lands on a sector boundary. The extra
			// bytes are zeros written past the end of the image, which is
			// what every other writer does too.
			out2 := payload
			if pad := n % writeAlignment; pad != 0 {
				padded := ((n / writeAlignment) + 1) * writeAlignment
				for i := n; i < padded; i++ {
					buf[i] = 0
				}
				out2 = buf[:padded]
			}
			if capacity > 0 && written+int64(len(out2)) > capacity {
				return written, errors.New(i18n.T().ImageTooLarge(written+int64(len(out2)), capacity))
			}
			if _, err := out.Write(out2); err != nil {
				return written, fmt.Errorf("write at offset %d: %w", written, err)
			}
			written += int64(n)
			rep.report(src.consumed(written), time.Now(), false)
		}

		if errors.Is(readErr, io.EOF) || errors.Is(readErr, io.ErrUnexpectedEOF) {
			break
		}
		if readErr != nil {
			return written, fmt.Errorf("read image: %w", readErr)
		}
	}

	// Without this the transfer looks finished while the card is still being
	// written, which is exactly when people pull it out.
	if err := out.Sync(); err != nil {
		return written, fmt.Errorf("flush to %s: %w", path, err)
	}
	rep.report(src.consumed(written), time.Now(), true)
	return written, nil
}

// hashDevice reads the first n bytes back off the card and hashes them.
func hashDevice(ctx context.Context, path string, n int64, rep *reporter) (string, error) {
	dev, err := blockdev.Open(path)
	if err != nil {
		return "", err
	}
	defer dev.Close()

	f, err := dev.Sys()
	if err != nil {
		return "", err
	}

	digest := sha256.New()
	buf := make([]byte, blockSize)
	var done int64
	for done < n {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		// Read whole aligned blocks and trim, since the card only serves
		// sector-aligned reads.
		want := int64(blockSize)
		read, err := f.ReadAt(buf[:want], done)
		if read == 0 && err != nil {
			return "", fmt.Errorf("read back at offset %d: %w", done, err)
		}
		chunk := buf[:read]
		if rem := n - done; int64(len(chunk)) > rem {
			chunk = chunk[:rem]
		}
		digest.Write(chunk)
		done += int64(len(chunk))
		rep.report(done, time.Now(), false)
	}
	rep.report(done, time.Now(), true)
	return hex.EncodeToString(digest.Sum(nil)), nil
}
