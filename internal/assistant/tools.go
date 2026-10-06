// Package assistant runs a Claude agent whose tools are the disk operations of
// this application: it can list, read and search the files on a card or image,
// and — only when the session was opened for writing — change them.
//
// The tool surface is deliberately small and path-based. Every call is reported
// to the caller so the user sees exactly what the model touched.
package assistant

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"

	"bbsdcardsniffer/internal/volume"
)

// maxReadBytes caps a single read_file call. Larger files are read in pages so
// one oversized log cannot fill the context window.
const maxReadBytes = 64 * 1024

// maxWalk bounds a recursive search, so a pathological tree cannot hang a turn.
const maxWalk = 20000

// tool couples a Claude tool definition to its implementation.
type tool struct {
	def anthropic.ToolParam
	// mutates marks tools that change the medium. They are only offered when
	// the session was opened for writing.
	mutates bool
	run     func(*volume.Session, int, json.RawMessage) (string, error)
}

func schema(props map[string]any, required ...string) anthropic.ToolInputSchemaParam {
	return anthropic.ToolInputSchemaParam{
		Properties: props,
		Required:   required,
	}
}

func str(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func integer(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

// args decodes a tool's input into v.
func args(raw json.RawMessage, v any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, v)
}

// tools returns the full tool set. Callers filter out the mutating ones when
// the session is read-only.
func tools() []tool {
	return []tool{
		{
			def: anthropic.ToolParam{
				Name:        "list_dir",
				Description: anthropic.String("List the entries of a directory on the volume. Paths are rooted at \"/\"."),
				InputSchema: schema(map[string]any{
					"path": str("Directory to list, for example \"/etc\". Use \"/\" for the root."),
				}, "path"),
			},
			run: runListDir,
		},
		{
			def: anthropic.ToolParam{
				Name: "read_file",
				Description: anthropic.String(
					"Read the contents of a file as text. Returns at most 64 KiB per call; use offset to page " +
						"through a larger file. Gzipped files such as rotated logs are decompressed automatically. " +
						"For a log, set tail to read the end of the file, which is where the most recent entries are."),
				InputSchema: schema(map[string]any{
					"path":   str("File to read, for example \"/etc/fstab\"."),
					"offset": integer("Byte offset to start at. Defaults to 0. Ignored when tail is set."),
					"limit":  integer("Maximum bytes to return, up to 65536."),
					"tail":   integer("Read this many bytes from the end of the file instead of from the start. Use this for logs."),
				}, "path"),
			},
			run: runReadFile,
		},
		{
			def: anthropic.ToolParam{
				Name: "find",
				Description: anthropic.String(
					"Search the directory tree for entries whose name contains a substring. " +
						"Use this to locate a file when its exact path is unknown."),
				InputSchema: schema(map[string]any{
					"name": str("Substring to match against entry names, case-insensitive."),
					"path": str("Directory to search under. Defaults to \"/\"."),
				}, "name"),
			},
			run: runFind,
		},
		{
			def: anthropic.ToolParam{
				Name: "grep",
				Description: anthropic.String(
					"Search the contents of text files under a directory for a substring, and return the " +
						"matching lines with their file and line number. This is the tool to reach for when " +
						"reading logs: it follows gzipped rotated logs, and for a file too large to scan whole " +
						"it searches the end, where the recent entries are."),
				InputSchema: schema(map[string]any{
					"pattern": str("Substring to look for, case-insensitive."),
					"path":    str("Directory to search under. Defaults to \"/\"."),
					"max":     integer("Maximum matching lines to return. Defaults to 100."),
				}, "pattern"),
			},
			run: runGrep,
		},
		{
			def: anthropic.ToolParam{
				Name:        "stat",
				Description: anthropic.String("Report the size, mode and modification time of one entry."),
				InputSchema: schema(map[string]any{
					"path": str("Entry to describe."),
				}, "path"),
			},
			run: runStat,
		},
		{
			def: anthropic.ToolParam{
				Name: "write_file",
				Description: anthropic.String(
					"Replace the entire contents of a file, creating it if it does not exist. " +
						"Read the file first if you are editing rather than replacing it: this tool " +
						"takes the complete new contents, not a patch."),
				InputSchema: schema(map[string]any{
					"path":    str("File to write."),
					"content": str("The complete new contents of the file."),
				}, "path", "content"),
			},
			mutates: true,
			run:     runWriteFile,
		},
		{
			def: anthropic.ToolParam{
				Name:        "make_dir",
				Description: anthropic.String("Create a directory, including any missing parents."),
				InputSchema: schema(map[string]any{
					"path": str("Directory to create."),
				}, "path"),
			},
			mutates: true,
			run:     runMakeDir,
		},
		{
			def: anthropic.ToolParam{
				Name:        "delete",
				Description: anthropic.String("Delete a file or an empty directory. This cannot be undone."),
				InputSchema: schema(map[string]any{
					"path": str("Entry to delete."),
				}, "path"),
			},
			mutates: true,
			run:     runDelete,
		},
	}
}

// --- implementations ---

func runListDir(s *volume.Session, vol int, raw json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := args(raw, &in); err != nil {
		return "", err
	}
	if in.Path == "" {
		in.Path = "/"
	}

	entries, err := s.ReadDir(vol, in.Path)
	if err != nil {
		return "", err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return entries[i].Name < entries[j].Name
	})

	var b strings.Builder
	fmt.Fprintf(&b, "%s (%d entries)\n", path.Clean("/"+in.Path), len(entries))
	for _, e := range entries {
		kind := "file"
		switch {
		case e.IsDir:
			kind = "dir "
		case e.IsSymlink:
			kind = "link"
		}
		fmt.Fprintf(&b, "%s %10s  %s  %s\n", kind, e.SizeHuman, e.Mode, e.Name)
	}
	return b.String(), nil
}

func runReadFile(s *volume.Session, vol int, raw json.RawMessage) (string, error) {
	var in struct {
		Path   string `json:"path"`
		Offset int64  `json:"offset"`
		Limit  int    `json:"limit"`
		Tail   int64  `json:"tail"`
	}
	if err := args(raw, &in); err != nil {
		return "", err
	}
	if in.Limit <= 0 || in.Limit > maxReadBytes {
		in.Limit = maxReadBytes
	}
	if in.Tail > int64(in.Limit) {
		in.Tail = int64(in.Limit)
	}

	t, err := readText(s, vol, in.Path, in.Offset, in.Limit, in.Tail)
	if err != nil {
		return "", err
	}
	if !isText(t.data) {
		return describeUnreadable(in.Path, t.total, t.data), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s, bytes %d-%d of %d", in.Path, t.offset, t.offset+int64(len(t.data)), t.full)
	if t.compressed {
		fmt.Fprintf(&b, " (gunzipped from %d bytes on disk)", t.total)
	}
	b.WriteString("\n\n")
	b.Write(t.data)
	if t.truncated {
		fmt.Fprintf(&b, "\n\n[truncated; call again with offset=%d for more]", t.offset+int64(len(t.data)))
	}
	return b.String(), nil
}

func runStat(s *volume.Session, vol int, raw json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := args(raw, &in); err != nil {
		return "", err
	}

	parent := path.Dir(path.Clean("/" + in.Path))
	name := path.Base(path.Clean("/" + in.Path))
	entries, err := s.ReadDir(vol, parent)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.Name == name {
			return fmt.Sprintf("%s\n  size %s (%d bytes)\n  mode %s\n  modified %s\n  dir %v symlink %v",
				e.Path, e.SizeHuman, e.Size, e.Mode, e.ModTime, e.IsDir, e.IsSymlink), nil
		}
	}
	return "", fmt.Errorf("%s does not exist", in.Path)
}

func runFind(s *volume.Session, vol int, raw json.RawMessage) (string, error) {
	var in struct {
		Name string `json:"name"`
		Path string `json:"path"`
	}
	if err := args(raw, &in); err != nil {
		return "", err
	}
	if in.Path == "" {
		in.Path = "/"
	}
	needle := strings.ToLower(in.Name)

	var hits []string
	visited := 0
	err := walk(s, vol, in.Path, &visited, func(e volume.Entry) error {
		if strings.Contains(strings.ToLower(e.Name), needle) {
			hits = append(hits, e.Path)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(hits) == 0 {
		return fmt.Sprintf("no entry under %s has %q in its name (%d entries searched)", in.Path, in.Name, visited), nil
	}
	return fmt.Sprintf("%d matches (%d entries searched):\n%s", len(hits), visited, strings.Join(hits, "\n")), nil
}

func runGrep(s *volume.Session, vol int, raw json.RawMessage) (string, error) {
	var in struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
		Max     int    `json:"max"`
	}
	if err := args(raw, &in); err != nil {
		return "", err
	}
	if in.Path == "" {
		in.Path = "/"
	}
	if in.Max <= 0 {
		in.Max = 100
	}
	needle := strings.ToLower(in.Pattern)

	var out []string
	visited := 0
	err := walk(s, vol, in.Path, &visited, func(e volume.Entry) error {
		if e.IsDir || e.IsSymlink || e.Size == 0 {
			return nil
		}
		if len(out) >= in.Max {
			return errStopWalk
		}
		t, err := scanText(s, vol, e.Path, e.Size)
		if err != nil || !isText(t.data) {
			// An unreadable or binary file is not worth aborting the search for.
			return nil
		}
		// Line numbers are relative to what was scanned, so a partial scan says
		// so rather than quoting a number that does not match the real file.
		partial := t.offset > 0
		for i, line := range strings.Split(string(t.data), "\n") {
			if !strings.Contains(strings.ToLower(line), needle) {
				continue
			}
			where := fmt.Sprintf("%s:%d", e.Path, i+1)
			if partial {
				where = fmt.Sprintf("%s:~%d (tail only)", e.Path, i+1)
			}
			out = append(out, fmt.Sprintf("%s: %s", where, strings.TrimSpace(line)))
			if len(out) >= in.Max {
				return errStopWalk
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(out) == 0 {
		return fmt.Sprintf("no line under %s contains %q (%d entries searched)", in.Path, in.Pattern, visited), nil
	}
	return strings.Join(out, "\n"), nil
}

func runWriteFile(s *volume.Session, vol int, raw json.RawMessage) (string, error) {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := args(raw, &in); err != nil {
		return "", err
	}
	if err := s.WriteFile(vol, in.Path, []byte(in.Content)); err != nil {
		return "", err
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(in.Content), in.Path), nil
}

func runMakeDir(s *volume.Session, vol int, raw json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := args(raw, &in); err != nil {
		return "", err
	}
	if err := s.Mkdir(vol, in.Path); err != nil {
		return "", err
	}
	return "created " + in.Path, nil
}

func runDelete(s *volume.Session, vol int, raw json.RawMessage) (string, error) {
	var in struct {
		Path string `json:"path"`
	}
	if err := args(raw, &in); err != nil {
		return "", err
	}
	if err := s.Remove(vol, in.Path); err != nil {
		return "", err
	}
	return "deleted " + in.Path, nil
}

// errStopWalk ends a walk early once enough results have been collected.
var errStopWalk = fmt.Errorf("walk complete")

// walk visits every entry under root, depth first, up to maxWalk entries.
func walk(s *volume.Session, vol int, root string, visited *int, fn func(volume.Entry) error) error {
	queue := []string{root}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]

		entries, err := s.ReadDir(vol, dir)
		if err != nil {
			// A directory we cannot read should not abort the whole search.
			continue
		}
		for _, e := range entries {
			*visited++
			if *visited > maxWalk {
				return nil
			}
			if err := fn(e); err != nil {
				if err == errStopWalk {
					return nil
				}
				return err
			}
			if e.IsDir && !e.IsSymlink {
				queue = append(queue, e.Path)
			}
		}
	}
	return nil
}

// isText mirrors the preview pane's heuristic: valid text with no NUL bytes.
func isText(b []byte) bool {
	if len(b) == 0 {
		return true
	}
	for _, c := range b {
		if c == 0 {
			return false
		}
	}
	return true
}
