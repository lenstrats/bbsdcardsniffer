package imaging

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"time"

	"bbsdcardsniffer/internal/blockdev"
	"bbsdcardsniffer/internal/i18n"
)

// blockSize is the unit we move data in. It is a multiple of every logical
// sector size in use, which keeps every read aligned, and large enough that
// the per-read overhead disappears against the transfer itself.
const blockSize = 4 << 20

// CloneOptions describes a card-to-image copy.
type CloneOptions struct {
	// Source is the device (or image) to read.
	Source string
	// Dest is the image file to write.
	Dest string
	// Limit stops the copy after this many bytes. Zero copies the whole
	// device. Callers use it to skip the unallocated tail of a card.
	Limit int64
	// Gzip compresses the image as it is written.
	Gzip bool
	// Verify re-reads the written image afterwards and checks it hashes to
	// the same value as the bytes taken off the card.
	Verify bool
}

// countingWriter tracks how many bytes actually reached the file, which
// differs from the device byte count once compression is in play.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// Clone copies a device to an image file. It reports progress as it goes and
// stops promptly when ctx is cancelled, leaving the partial file in place for
// the caller to delete.
func Clone(ctx context.Context, opts CloneOptions, report ProgressFunc) (*Result, error) {
	if opts.Source == "" || opts.Dest == "" {
		return nil, errors.New(i18n.T().CloneNeedsBoth())
	}

	dev, err := blockdev.Open(opts.Source)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", opts.Source, err)
	}
	defer dev.Close()

	deviceSize, err := dev.Length()
	if err != nil {
		return nil, fmt.Errorf("size of %s: %w", opts.Source, err)
	}
	total := deviceSize
	if opts.Limit > 0 && opts.Limit < total {
		total = opts.Limit
	}

	src, err := dev.Sys()
	if err != nil {
		return nil, err
	}

	out, err := os.Create(opts.Dest)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", opts.Dest, err)
	}
	defer out.Close()

	counter := &countingWriter{w: out}
	var sink io.Writer = counter
	var gz *gzip.Writer
	if opts.Gzip {
		gz = gzip.NewWriter(counter)
		sink = gz
	}

	start := time.Now()
	rep := newReporter(report, total, start)
	digest := sha256.New()

	copied, err := copyDevice(ctx, sink, digest, src, deviceSize, total, rep)
	if err != nil {
		return nil, err
	}

	if gz != nil {
		if err := gz.Close(); err != nil {
			return nil, fmt.Errorf("finish compression: %w", err)
		}
	}
	// Without this the image can still be in the page cache when we verify,
	// which would check our own buffers rather than what reached the disk.
	if err := out.Sync(); err != nil {
		return nil, fmt.Errorf("flush %s: %w", opts.Dest, err)
	}

	sum := hex.EncodeToString(digest.Sum(nil))
	result := &Result{
		Bytes:     copied,
		SHA256:    sum,
		FileBytes: counter.n,
		Seconds:   time.Since(start).Seconds(),
		Path:      opts.Dest,
	}

	if opts.Verify {
		rep.setPhase(PhaseVerifying)
		readBack, err := hashImage(ctx, opts.Dest, opts.Gzip, total, rep)
		if err != nil {
			return nil, fmt.Errorf("verify %s: %w", opts.Dest, err)
		}
		if readBack != sum {
			return nil, errors.New(i18n.T().VerifyFailedClone(sum, readBack))
		}
		result.Verified = true
		result.Seconds = time.Since(start).Seconds()
	}

	return result, nil
}

// copyDevice streams total bytes from src into dst, hashing as it goes. Reads
// stay aligned to blockSize even when the last chunk is trimmed, because raw
// devices reject anything else.
func copyDevice(ctx context.Context, dst io.Writer, digest hash.Hash, src *os.File, deviceSize, total int64, rep *reporter) (int64, error) {
	buf := make([]byte, blockSize)
	var done int64

	for done < total {
		if err := ctx.Err(); err != nil {
			return done, err
		}

		// Read a full aligned block whenever the device still has one, and
		// only trim what we hand onwards.
		readLen := int64(blockSize)
		if rem := deviceSize - done; rem < readLen {
			readLen = rem
		}
		if readLen <= 0 {
			break
		}

		n, err := src.ReadAt(buf[:readLen], done)
		if n == 0 && err != nil {
			return done, fmt.Errorf("read at offset %d: %w", done, err)
		}

		chunk := buf[:n]
		if rem := total - done; int64(len(chunk)) > rem {
			chunk = chunk[:rem]
		}
		if _, err := dst.Write(chunk); err != nil {
			return done, fmt.Errorf("write at offset %d: %w", done, err)
		}
		digest.Write(chunk)

		done += int64(len(chunk))
		rep.report(done, time.Now(), false)

		if err != nil && !errors.Is(err, io.EOF) {
			return done, fmt.Errorf("read at offset %d: %w", done, err)
		}
	}

	rep.report(done, time.Now(), true)
	return done, nil
}

// hashImage reads an image back and returns the SHA-256 of its uncompressed
// contents, so it can be compared with what came off the card.
func hashImage(ctx context.Context, path string, compressed bool, total int64, rep *reporter) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var src io.Reader = f
	if compressed {
		gz, err := gzip.NewReader(f)
		if err != nil {
			return "", err
		}
		defer gz.Close()
		src = gz
	}

	digest := sha256.New()
	buf := make([]byte, blockSize)
	var done int64
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := src.Read(buf)
		if n > 0 {
			digest.Write(buf[:n])
			done += int64(n)
			rep.report(done, time.Now(), false)
		}
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", err
		}
	}
	rep.report(done, time.Now(), true)

	if done != total {
		return "", fmt.Errorf("image holds %d bytes but %d were written", done, total)
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
