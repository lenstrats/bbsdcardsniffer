package main

import (
	"context"
	"errors"
	"sync"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"bbsdcardsniffer/internal/assistant"
	"bbsdcardsniffer/internal/config"
	"bbsdcardsniffer/internal/i18n"
	"bbsdcardsniffer/internal/volume"
)

// assistantEvent is the channel the agent's progress reaches the UI on.
const assistantEvent = "assistant:event"

// assistantState keeps one conversation per opened volume.
type assistantState struct {
	mu     sync.Mutex
	agent  *assistant.Agent
	volume int
	cancel context.CancelFunc
	busy   bool
}

// AssistantStatus tells the UI what it can offer.
type AssistantStatus struct {
	// HasKey reports whether a Claude API key is available.
	HasKey bool `json:"hasKey"`
	// KeyFromEnv reports that the key came from ANTHROPIC_API_KEY, in which
	// case the interface should not offer to overwrite it.
	KeyFromEnv bool `json:"keyFromEnv"`
	// Writable reports whether the open disk allows changes.
	Writable bool `json:"writable"`
	// Busy reports whether a question is currently being answered.
	Busy bool `json:"busy"`
	// BackupPath is where a backup of the open medium was written this
	// session, or empty if none was made. The editing gate uses it to tell
	// whether the safe step has already been taken.
	BackupPath string `json:"backupPath"`
}

func (a *App) AssistantStatus() AssistantStatus {
	key, fromEnv := config.APIKey()

	a.assistant.mu.Lock()
	busy := a.assistant.busy
	a.assistant.mu.Unlock()

	a.mu.Lock()
	writable := a.session != nil && a.session.Writable
	backup := ""
	if a.session != nil {
		backup = a.backedUp[a.session.Path]
	}
	a.mu.Unlock()

	return AssistantStatus{
		HasKey:     key != "",
		KeyFromEnv: fromEnv,
		Writable:   writable,
		Busy:       busy,
		BackupPath: backup,
	}
}

// SetAPIKey stores a Claude API key for future runs.
func (a *App) SetAPIKey(key string) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	c.APIKey = key
	return config.Save(c)
}

// EnableWrites reopens the current disk in the requested mode. Editing needs a
// writable handle, and the browsing side deliberately cannot provide one.
func (a *App) EnableWrites(enable bool) (*DiskInfo, error) {
	a.mu.Lock()
	current := a.session
	a.mu.Unlock()

	if current == nil {
		return nil, errors.New(i18n.T().NoDeviceOpen())
	}
	if current.Writable == enable {
		return a.describeCurrent()
	}

	path := current.Path

	// Drop the conversation: it was formed against a session that is closing,
	// and its tool set differs between the two modes.
	a.resetAssistant()

	a.mu.Lock()
	defer a.mu.Unlock()
	a.session.Close()
	a.session = nil

	open := volume.Open
	if enable {
		open = volume.OpenWritable
	}
	s, err := open(path)
	if err != nil {
		return nil, describeOpenError(path, err)
	}
	a.session = s

	return &DiskInfo{
		Path:      s.Path,
		Size:      s.Size,
		SizeHuman: s.SizeHuman,
		Table:     s.Table,
		Volumes:   s.Volumes,
		Writable:  s.Writable,
	}, nil
}

func (a *App) describeCurrent() (*DiskInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.session == nil {
		return nil, errors.New(i18n.T().NoDeviceOpen())
	}
	return &DiskInfo{
		Path:      a.session.Path,
		Size:      a.session.Size,
		SizeHuman: a.session.SizeHuman,
		Table:     a.session.Table,
		Volumes:   a.session.Volumes,
		Writable:  a.session.Writable,
	}, nil
}

// resetAssistant forgets the conversation so far.
func (a *App) resetAssistant() {
	a.assistant.mu.Lock()
	defer a.assistant.mu.Unlock()
	if a.assistant.cancel != nil {
		a.assistant.cancel()
	}
	a.assistant.agent = nil
	a.assistant.cancel = nil
}

// ResetAssistant clears the conversation, exposed for the "new chat" button.
func (a *App) ResetAssistant() { a.resetAssistant() }

// CancelAssistant stops the question being answered.
func (a *App) CancelAssistant() error {
	a.assistant.mu.Lock()
	defer a.assistant.mu.Unlock()
	if a.assistant.cancel == nil || !a.assistant.busy {
		return errors.New(i18n.T().AssistantIdle())
	}
	a.assistant.cancel()
	return nil
}

// AskAssistant puts a question to Claude about one volume and streams the
// answer back over the assistant:event channel. It returns when the turn ends.
func (a *App) AskAssistant(volIndex int, prompt string) error {
	key, _ := config.APIKey()
	if key == "" {
		return errors.New(i18n.T().NoAPIKey())
	}

	s, err := a.current()
	if err != nil {
		return err
	}
	if volIndex < 0 || volIndex >= len(s.Volumes) {
		return errors.New(i18n.T().NoSuchVolume(volIndex))
	}
	vol := s.Volumes[volIndex]
	if !vol.Supported {
		return errors.New(i18n.T().VolumeUnreadable(volIndex, vol.FS.Kind))
	}

	a.assistant.mu.Lock()
	if a.assistant.busy {
		a.assistant.mu.Unlock()
		return errors.New(i18n.T().AssistantBusy())
	}
	// A conversation is tied to one volume; switching volumes starts a new one.
	if a.assistant.agent == nil || a.assistant.volume != volIndex {
		a.assistant.agent = assistant.New(key, s, vol)
		a.assistant.volume = volIndex
	}
	agent := a.assistant.agent
	ctx, cancel := context.WithCancel(a.ctx)
	a.assistant.cancel = cancel
	a.assistant.busy = true
	a.assistant.mu.Unlock()

	defer func() {
		a.assistant.mu.Lock()
		a.assistant.busy = false
		a.assistant.cancel = nil
		a.assistant.mu.Unlock()
		cancel()
	}()

	return agent.Ask(ctx, prompt, func(e assistant.Event) {
		wruntime.EventsEmit(a.ctx, assistantEvent, e)
	})
}
