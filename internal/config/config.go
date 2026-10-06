// Package config stores the small amount of state that has to outlive a run:
// currently only the Claude API key.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Config is what we persist between runs.
type Config struct {
	// APIKey is the Claude API key. It is stored in plain text in a file only
	// the current user can read; this is not a secret store, and the
	// ANTHROPIC_API_KEY environment variable is the better option where the
	// user already has one set.
	APIKey string `json:"apiKey,omitempty"`
}

func path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "bbsdcardsniffer", "config.json"), nil
}

// Load reads the stored configuration, returning an empty one if none exists.
func Load() (Config, error) {
	var c Config
	p, err := path()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(p)
	if errors.Is(err, fs.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, fmt.Errorf("parse %s: %w", p, err)
	}
	return c, nil
}

// Save writes the configuration, readable only by the current user.
func Save(c Config) error {
	p, err := path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

// APIKey resolves the key to use. The environment wins, so a key exported for
// the shell session does not have to be entered again in the interface.
func APIKey() (key string, fromEnv bool) {
	if env := strings.TrimSpace(os.Getenv("ANTHROPIC_API_KEY")); env != "" {
		return env, true
	}
	c, err := Load()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(c.APIKey), false
}
