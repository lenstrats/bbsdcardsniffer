package assistant

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"bbsdcardsniffer/internal/volume"
)

// Reading logs pulls in three things a plain byte-range read does not give you.
//
//   - Logs are appended, so the interesting part is the end. Paging forward
//     from offset 0 through a 200 MB syslog to reach yesterday's failure is
//     not a workable way to investigate.
//   - Rotated logs are gzipped. Served raw they look like binary and get
//     refused, which silently hides everything older than the current file.
//   - systemd's journal is a binary format we cannot decode, and saying only
//     "binary file" leaves the model guessing why the logs seem to be missing.
const (
	// gzipMaxDecompressed caps what a compressed log may expand to in memory.
	gzipMaxDecompressed = 64 << 20
	// scanCap is how much of one file grep will look at. Bigger files are
	// searched from the end, where the recent entries are.
	scanCap = 8 << 20
)

// text is the readable content of a file, with the context needed to describe
// where it came from.
type text struct {
	data []byte
	// total is the file's size on disk, before any decompression.
	total int64
	// offset is where data starts within the readable content.
	offset int64
	// full is the length of the readable content, which differs from total
	// for a compressed file.
	full int64
	// compressed marks a file that was gunzipped on the way through.
	compressed bool
	// truncated marks that only part of the readable content was returned.
	truncated bool
}

// readText reads part of a file, transparently decompressing gzip.
//
// When tail is positive it returns the last tail bytes instead of starting at
// offset, which is what makes "what went wrong most recently" a single call.
func readText(s *volume.Session, vol int, name string, offset int64, limit int, tail int64) (*text, error) {
	head, total, err := s.ReadChunk(vol, name, 0, 2)
	if err != nil {
		return nil, err
	}

	if bytes.HasPrefix(head, []byte{0x1f, 0x8b}) {
		return readGzip(s, vol, name, total, offset, limit, tail)
	}

	if tail > 0 {
		offset = total - tail
		if offset < 0 {
			offset = 0
		}
	}
	data, _, err := s.ReadChunk(vol, name, offset, limit)
	if err != nil {
		return nil, err
	}
	return &text{
		data:      data,
		total:     total,
		offset:    offset,
		full:      total,
		truncated: offset+int64(len(data)) < total,
	}, nil
}

// readGzip expands a compressed log and returns the requested window of it.
func readGzip(s *volume.Session, vol int, name string, total, offset int64, limit int, tail int64) (*text, error) {
	var raw bytes.Buffer
	if _, err := s.CopyFile(vol, name, &raw); err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}

	zr, err := gzip.NewReader(&raw)
	if err != nil {
		return nil, fmt.Errorf("%s looks gzipped but will not open: %w", name, err)
	}
	defer zr.Close()

	// Bound the expansion: a small compressed file can hold a great deal.
	plain, err := io.ReadAll(io.LimitReader(zr, gzipMaxDecompressed))
	if err != nil {
		return nil, fmt.Errorf("decompress %s: %w", name, err)
	}

	full := int64(len(plain))
	if tail > 0 {
		offset = full - tail
		if offset < 0 {
			offset = 0
		}
	}
	if offset > full {
		offset = full
	}
	end := offset + int64(limit)
	if end > full {
		end = full
	}

	return &text{
		data:       plain[offset:end],
		total:      total,
		offset:     offset,
		full:       full,
		compressed: true,
		truncated:  end < full,
	}, nil
}

// scanText returns as much of a file as grep is willing to search, taking the
// end of an oversized file rather than skipping it.
func scanText(s *volume.Session, vol int, name string, size int64) (*text, error) {
	if size > scanCap {
		return readText(s, vol, name, 0, scanCap, scanCap)
	}
	return readText(s, vol, name, 0, int(size)+1, 0)
}

// describeUnreadable explains why a file cannot be shown as text, naming the
// format where we recognise it so the answer can say something useful instead
// of "binary file".
func describeUnreadable(name string, total int64, data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte("LPKSHHRH")):
		// systemd journal files start with this signature.
		return fmt.Sprintf(
			"%s is a systemd journal file (%d bytes). It is a binary format this tool cannot decode. "+
				"Look for plain-text logs instead — /var/log/syslog, /var/log/messages, "+
				"/var/log/daemon.log, /var/log/kern.log or the files under /var/log/apt — or tell the "+
				"user the journal has to be read with journalctl on a Linux machine.", name, total)
	case strings.HasSuffix(name, ".xz"), bytes.HasPrefix(data, []byte{0xfd, '7', 'z'}):
		return fmt.Sprintf("%s is xz-compressed (%d bytes); this tool decompresses gzip only.", name, total)
	case strings.HasSuffix(name, ".zst"), bytes.HasPrefix(data, []byte{0x28, 0xb5, 0x2f, 0xfd}):
		return fmt.Sprintf("%s is zstd-compressed (%d bytes); this tool decompresses gzip only.", name, total)
	}
	return fmt.Sprintf("%s is a binary file of %d bytes; not shown as text.", name, total)
}
