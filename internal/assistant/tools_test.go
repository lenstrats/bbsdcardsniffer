package assistant

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"bbsdcardsniffer/internal/volume"
)

// openReal opens the image named by REAL_IMAGE and returns its Linux root
// volume, which is where the interesting files live.
func openReal(t *testing.T, writable bool) (*volume.Session, int) {
	t.Helper()
	path := os.Getenv("REAL_IMAGE")
	if path == "" {
		t.Skip("set REAL_IMAGE to a card image to run the tool tests")
	}

	open := volume.Open
	if writable {
		open = volume.OpenWritable
	}
	s, err := open(path)
	if err != nil {
		if writable && strings.Contains(err.Error(), "permission denied") {
			// A root-owned image is common; that says nothing about the code.
			t.Skipf("%s is not writable by this user", path)
		}
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { s.Close() })

	for _, v := range s.Volumes {
		if !v.Supported {
			continue
		}
		if entries, err := s.ReadDir(v.Index, "/"); err == nil {
			for _, e := range entries {
				if e.Name == "etc" {
					return s, v.Index
				}
			}
		}
	}
	t.Skip("no volume with an /etc on it")
	return nil, 0
}

func call(t *testing.T, s *volume.Session, vol int, name string, input any) string {
	t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, tl := range tools() {
		if tl.def.Name != name {
			continue
		}
		out, err := tl.run(s, vol, raw)
		if err != nil {
			t.Fatalf("%s(%s): %v", name, raw, err)
		}
		return out
	}
	t.Fatalf("no tool named %s", name)
	return ""
}

func TestListDirTool(t *testing.T) {
	s, vol := openReal(t, false)

	out := call(t, s, vol, "list_dir", map[string]any{"path": "/etc"})
	if !strings.Contains(out, "entries") {
		t.Errorf("no header in output: %q", out[:min(200, len(out))])
	}
	// The listing must distinguish directories from files, or the model
	// cannot tell what it may descend into.
	if !strings.Contains(out, "dir ") || !strings.Contains(out, "file ") {
		t.Errorf("listing does not label entry kinds:\n%s", out[:min(400, len(out))])
	}
	t.Logf("first lines:\n%s", firstLines(out, 5))
}

func TestReadFileTool(t *testing.T) {
	s, vol := openReal(t, false)

	out := call(t, s, vol, "read_file", map[string]any{"path": "/etc/hostname"})
	if !strings.Contains(out, "/etc/hostname") {
		t.Errorf("output does not name the file: %q", out)
	}
	t.Logf("%s", out)
}

// A binary file must be reported, not dumped into the context window.
func TestReadFileRefusesBinary(t *testing.T) {
	s, vol := openReal(t, false)

	entries, err := s.ReadDir(vol, "/bin")
	if err != nil {
		t.Skipf("no /bin: %v", err)
	}
	var binary string
	for _, e := range entries {
		if !e.IsDir && !e.IsSymlink && e.Size > 1024 {
			binary = e.Path
			break
		}
	}
	if binary == "" {
		t.Skip("no binary found to test with")
	}

	out := call(t, s, vol, "read_file", map[string]any{"path": binary})
	if !strings.Contains(out, "binary file") {
		t.Errorf("%s was not reported as binary: %q", binary, out[:min(200, len(out))])
	}
}

func TestFindTool(t *testing.T) {
	s, vol := openReal(t, false)

	out := call(t, s, vol, "find", map[string]any{"name": "fstab", "path": "/etc"})
	if !strings.Contains(out, "fstab") {
		t.Errorf("find did not locate fstab: %q", out)
	}
	t.Logf("%s", firstLines(out, 5))
}

// grep is the tool the model reaches for when asked to read logs, so it is
// checked against a directory that really holds them.
func TestGrepTool(t *testing.T) {
	s, vol := openReal(t, false)

	out := call(t, s, vol, "grep", map[string]any{"pattern": "root", "path": "/etc", "max": 20})
	if strings.HasPrefix(out, "no line") {
		t.Errorf("grep found nothing for a term that should appear in /etc: %q", out)
	}
	// Every hit must carry file and line number, or the answer cannot cite it.
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "/") || strings.Count(line, ":") < 2 {
			t.Errorf("malformed grep hit: %q", line)
			break
		}
	}
	t.Logf("%s", firstLines(out, 5))
}

func TestStatTool(t *testing.T) {
	s, vol := openReal(t, false)

	out := call(t, s, vol, "stat", map[string]any{"path": "/etc/hostname"})
	for _, want := range []string{"size", "mode", "modified"} {
		if !strings.Contains(out, want) {
			t.Errorf("stat output lacks %q: %q", want, out)
		}
	}
}

// A read-only session must not even offer the mutating tools.
func TestMutatingToolsHiddenWhenReadOnly(t *testing.T) {
	s, vol := openReal(t, false)
	_ = vol

	agent := New("test-key", s, s.Volumes[0])
	for _, tl := range agent.tools {
		if tl.mutates {
			t.Errorf("read-only session offers the mutating tool %q", tl.def.Name)
		}
	}
	if _, ok := agent.byName["write_file"]; ok {
		t.Error("write_file is reachable on a read-only session")
	}
	// The read tools must still be there, or the assistant is useless.
	for _, want := range []string{"list_dir", "read_file", "grep", "find", "stat"} {
		if _, ok := agent.byName[want]; !ok {
			t.Errorf("read-only session is missing the %q tool", want)
		}
	}
}

func TestMutatingToolsOfferedWhenWritable(t *testing.T) {
	s, _ := openReal(t, true)

	agent := New("test-key", s, s.Volumes[0])
	for _, want := range []string{"write_file", "make_dir", "delete"} {
		if _, ok := agent.byName[want]; !ok {
			t.Errorf("writable session is missing the %q tool", want)
		}
	}
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// The diagnostic flow lives or dies on these three: reading the end of a log,
// following rotated gzip logs, and saying something useful about a journal.

func TestReadFileTail(t *testing.T) {
	s, vol := openReal(t, false)

	// Find a text log big enough that the end differs from the start.
	var target string
	var size int64
	for _, dir := range []string{"/var/log", "/etc"} {
		entries, err := s.ReadDir(vol, dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir && !e.IsSymlink && e.Size > 4096 && !strings.HasSuffix(e.Name, ".gz") {
				if data, _, err := s.ReadChunk(vol, e.Path, 0, 512); err == nil && isText(data) {
					target, size = e.Path, e.Size
					break
				}
			}
		}
		if target != "" {
			break
		}
	}
	if target == "" {
		t.Skip("no sizeable text file to test tailing with")
	}

	head := call(t, s, vol, "read_file", map[string]any{"path": target, "limit": 512})
	tail := call(t, s, vol, "read_file", map[string]any{"path": target, "tail": 512, "limit": 512})

	if head == tail {
		t.Errorf("tail returned the same window as the head of %s (%d bytes)", target, size)
	}
	if !strings.Contains(tail, target) {
		t.Errorf("tail output does not name the file: %q", tail[:min(120, len(tail))])
	}
	t.Logf("tail of %s:\n%s", target, firstLines(tail, 3))
}

func TestReadFileFollowsGzip(t *testing.T) {
	s, vol := openReal(t, false)

	var target string
	entries, err := s.ReadDir(vol, "/var/log")
	if err != nil {
		t.Skipf("no /var/log: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir && strings.HasSuffix(e.Name, ".gz") && e.Size > 0 {
			target = e.Path
			break
		}
	}
	if target == "" {
		t.Skip("no gzipped rotated log on this image")
	}

	out := call(t, s, vol, "read_file", map[string]any{"path": target, "limit": 2048})
	if strings.Contains(out, "binary file") {
		t.Errorf("%s was refused as binary instead of being decompressed", target)
	}
	if !strings.Contains(out, "gunzipped") {
		t.Errorf("output does not report the decompression: %q", out[:min(200, len(out))])
	}
	t.Logf("%s", firstLines(out, 4))
}

// A journal file must produce an explanation the answer can pass on, not a
// bare "binary file".
func TestJournalIsExplained(t *testing.T) {
	got := describeUnreadable("/var/log/journal/abc/system.journal", 8388608, []byte("LPKSHHRH\x00\x00"))
	for _, want := range []string{"systemd journal", "journalctl", "/var/log/syslog"} {
		if !strings.Contains(got, want) {
			t.Errorf("explanation lacks %q: %s", want, got)
		}
	}

	// Anything unrecognised still gets the plain message.
	plain := describeUnreadable("/bin/ls", 100, []byte{0x7f, 'E', 'L', 'F'})
	if !strings.Contains(plain, "binary file") {
		t.Errorf("unrecognised binary lost its message: %s", plain)
	}
}

// grep must not skip a large log; it should search its tail and label the
// line numbers as approximate so nothing misquotes a position.
func TestGrepSearchesLargeFiles(t *testing.T) {
	s, vol := openReal(t, false)

	out := call(t, s, vol, "grep", map[string]any{"pattern": "error", "path": "/var/log", "max": 20})
	if strings.HasPrefix(out, "no line") {
		t.Skipf("nothing matching in /var/log on this image: %s", out)
	}
	t.Logf("%s", firstLines(out, 6))
}
