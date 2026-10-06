package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"bbsdcardsniffer/internal/i18n"
	"bbsdcardsniffer/internal/volume"
)

// model is fixed rather than configurable: the tool loop and the prompt are
// tuned for it, and letting it drift would change behaviour silently.
const model = "claude-opus-5"

// maxTokens is generous because the answer may quote a long log. Streaming
// keeps a large ceiling from running into request timeouts.
const maxTokens = 32000

// maxTurns bounds one question's tool loop, so a model that keeps searching
// cannot run indefinitely against a 30 GB card.
const maxTurns = 40

// EventKind labels what the UI should render.
type EventKind string

const (
	EventText       EventKind = "text"
	EventToolCall   EventKind = "tool_call"
	EventToolResult EventKind = "tool_result"
	EventDone       EventKind = "done"
)

// Event is one update from a running turn.
type Event struct {
	Kind EventKind `json:"kind"`
	// Text carries a streamed fragment of the answer.
	Text string `json:"text,omitempty"`
	// Tool and Input describe a call the model is making.
	Tool  string `json:"tool,omitempty"`
	Input string `json:"input,omitempty"`
	// Result is the tool's output, trimmed for display.
	Result string `json:"result,omitempty"`
	// Failed marks a tool call that returned an error.
	Failed bool `json:"failed,omitempty"`
	// Mutating marks a call that changed the medium, so the UI can highlight it.
	Mutating bool `json:"mutating,omitempty"`
}

// Agent holds one conversation about one volume.
type Agent struct {
	client  anthropic.Client
	session *volume.Session
	volume  volume.Volume

	tools    []tool
	byName   map[string]tool
	messages []anthropic.MessageParam
}

// New builds an agent over an open volume. Mutating tools are only offered when
// the session was opened for writing — the model is never shown a tool it
// would not be allowed to use.
func New(apiKey string, session *volume.Session, vol volume.Volume) *Agent {
	all := tools()
	available := make([]tool, 0, len(all))
	byName := make(map[string]tool, len(all))
	for _, t := range all {
		if t.mutates && !session.Writable {
			continue
		}
		available = append(available, t)
		byName[t.def.Name] = t
	}

	return &Agent{
		client:  anthropic.NewClient(option.WithAPIKey(apiKey)),
		session: session,
		volume:  vol,
		tools:   available,
		byName:  byName,
	}
}

// systemPrompt tells the model what it is looking at. Stating the medium and
// the access mode up front avoids a round of questions about both.
func (a *Agent) systemPrompt() string {
	var b strings.Builder
	b.WriteString("You are helping someone inspect a storage medium through a disk tool. ")
	b.WriteString("Your tools read and search the files on one volume of it.\n\n")

	fmt.Fprintf(&b, "Medium: %s (%s)\n", a.session.Path, a.session.SizeHuman)
	fmt.Fprintf(&b, "Partition table: %s\n", a.session.Table)
	fmt.Fprintf(&b, "Volume: %s, filesystem %s", a.volume.Label, a.volume.FS.Kind)
	if a.volume.Name != "" {
		fmt.Fprintf(&b, ", partition name %q", a.volume.Name)
	}
	fmt.Fprintf(&b, ", %s\n\n", a.volume.SizeHuman)

	if a.session.Writable {
		b.WriteString("This volume is open for WRITING. You can change files with write_file, " +
			"make_dir and delete. Those changes are permanent and there is no undo. " +
			"Before you change anything, read the file first and say what you intend to change and why. " +
			"Prefer the smallest change that solves the problem.\n\n")
	} else {
		b.WriteString("This volume is open READ-ONLY. You cannot change anything on it. " +
			"If the user asks for a change, explain what you would change and tell them to " +
			"reopen the disk in writable mode.\n\n")
	}

	b.WriteString("Paths are absolute and rooted at \"/\", which is the root of this volume, " +
		"not of the machine you are running on. A Linux root filesystem here has its logs " +
		"under /var/log and its configuration under /etc.\n\n")
	b.WriteString("Work from what the files actually say. When you report a cause, quote the " +
		"line you based it on. If the evidence does not support a conclusion, say what you " +
		"found and what is still unknown rather than guessing.\n\n")

	b.WriteString("When you are asked to diagnose a fault, these are the places worth looking " +
		"on a Linux root filesystem:\n" +
		"- /var/log/syslog, /var/log/messages, /var/log/daemon.log, /var/log/kern.log for " +
		"general and kernel messages, and /var/log/dmesg for the boot log.\n" +
		"- Rotated logs sit beside them as .1, .2.gz and so on; read_file decompresses the " +
		"gzipped ones, so history further back is available.\n" +
		"- Logs are appended, so the failure is usually at the end. Use read_file with tail " +
		"rather than paging forward from the start.\n" +
		"- grep over /var/log is the fastest way in when you do not yet know which file matters.\n" +
		"- /var/log/journal holds systemd's binary journal, which cannot be decoded here; " +
		"fall back to the plain-text logs and say so.\n" +
		"- Configuration that explains a failure usually lives in /etc: fstab for mounts, " +
		"the network directories for connectivity, and systemd unit files under /etc/systemd.\n\n")

	b.WriteString("Investigate before concluding. A single matching line is a lead, not a " +
		"diagnosis: check the surrounding entries and the timestamps, and note whether the " +
		"problem recurs or happened once.")
	return b.String()
}

// toolParams renders the tool definitions for the request.
func (a *Agent) toolParams() []anthropic.ToolUnionParam {
	out := make([]anthropic.ToolUnionParam, 0, len(a.tools))
	for i := range a.tools {
		def := a.tools[i].def
		out = append(out, anthropic.ToolUnionParam{OfTool: &def})
	}
	return out
}

// Ask runs one turn of the conversation, looping over tool calls until the
// model produces its answer. Events are emitted as they happen so the UI can
// stream. The conversation is kept, so follow-up questions have the context.
func (a *Agent) Ask(ctx context.Context, prompt string, emit func(Event)) error {
	a.messages = append(a.messages, anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)))

	for turn := 0; turn < maxTurns; turn++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		reply, err := a.stream(ctx, emit)
		if err != nil {
			return err
		}
		a.messages = append(a.messages, reply.ToParam())

		if reply.StopReason != anthropic.StopReasonToolUse {
			emit(Event{Kind: EventDone})
			return nil
		}

		results, err := a.runTools(ctx, reply, emit)
		if err != nil {
			return err
		}
		a.messages = append(a.messages, anthropic.NewUserMessage(results...))
	}

	return errors.New(i18n.T().OutOfToolTurns(maxTurns))
}

// stream issues one request and forwards text as it arrives.
func (a *Agent) stream(ctx context.Context, emit func(Event)) (*anthropic.Message, error) {
	adaptive := anthropic.ThinkingConfigAdaptiveParam{}
	params := anthropic.MessageNewParams{
		Model:     model,
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{{
			Text: a.systemPrompt(),
			// The prompt and tool list are identical on every turn, so caching
			// them keeps a long tool loop cheap.
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		}},
		Thinking: anthropic.ThinkingConfigParamUnion{OfAdaptive: &adaptive},
		Tools:    a.toolParams(),
		Messages: a.messages,
	}

	stream := a.client.Messages.NewStreaming(ctx, params)
	message := anthropic.Message{}

	for stream.Next() {
		event := stream.Current()
		if err := message.Accumulate(event); err != nil {
			return nil, fmt.Errorf("assemble response: %w", err)
		}
		if delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			if text, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok && text.Text != "" {
				emit(Event{Kind: EventText, Text: text.Text})
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return &message, nil
}

// runTools executes every tool call in a reply and returns the result blocks.
// All results go back in a single user message, which is what keeps the model
// making parallel calls on later turns.
func (a *Agent) runTools(ctx context.Context, reply *anthropic.Message, emit func(Event)) ([]anthropic.ContentBlockParamUnion, error) {
	var results []anthropic.ContentBlockParamUnion

	for _, block := range reply.Content {
		call, ok := block.AsAny().(anthropic.ToolUseBlock)
		if !ok {
			continue
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		input := json.RawMessage(call.Input)
		t, known := a.byName[call.Name]
		emit(Event{
			Kind:     EventToolCall,
			Tool:     call.Name,
			Input:    string(input),
			Mutating: t.mutates,
		})

		if !known {
			missing := i18n.T().NoSuchTool()
			results = append(results, anthropic.NewToolResultBlock(call.ID, missing, true))
			emit(Event{Kind: EventToolResult, Tool: call.Name, Result: missing, Failed: true})
			continue
		}

		out, err := t.run(a.session, a.volume.Index, input)
		if err != nil {
			// Tool failures are reported to the model, not raised: it can
			// often recover by trying a different path.
			msg := err.Error()
			results = append(results, anthropic.NewToolResultBlock(call.ID, msg, true))
			emit(Event{Kind: EventToolResult, Tool: call.Name, Result: msg, Failed: true, Mutating: t.mutates})
			continue
		}

		results = append(results, anthropic.NewToolResultBlock(call.ID, out, false))
		emit(Event{
			Kind:     EventToolResult,
			Tool:     call.Name,
			Result:   summarise(out),
			Mutating: t.mutates,
		})
	}
	return results, nil
}

// summarise trims a tool result for the transcript. The model still receives
// the whole thing; only the display is shortened.
func summarise(s string) string {
	const limit = 400
	if len(s) <= limit {
		return s
	}
	return s[:limit] + fmt.Sprintf("\n… (%d more bytes)", len(s)-limit)
}
