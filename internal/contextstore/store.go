package contextstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Info is a named Z21 command station endpoint.
type Info struct {
	Name        string   `json:"name"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Description string   `json:"description,omitempty"`
	Session     *Session `json:"session,omitempty"`
}

// Session tracks the UDP source port used with a command station.
type Session struct {
	LocalHost      string  `json:"local_host"`
	LocalPort      int     `json:"local_port"`
	BroadcastFlags *uint32 `json:"broadcast_flags,omitempty"`
}

// File is the on-disk context configuration.
type File struct {
	Current  string `json:"current"`
	Contexts []Info `json:"contexts"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".z21_contexts.json"), nil
}

// Load reads contexts from path, or ~/.z21_contexts.json when path is empty.
func Load(path string) (*File, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}

	store := &File{}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return store, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, store); err != nil {
		return nil, err
	}
	return store, nil
}

// Save writes contexts to path, or ~/.z21_contexts.json when path is empty.
func Save(path string, store *File) error {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// Current returns the active context, if any.
func Current(store *File) (*Info, error) {
	if store.Current == "" {
		return nil, fmt.Errorf("no current context set, run `z21 ctx use <NAME>` first")
	}
	for _, c := range store.Contexts {
		if c.Name == store.Current {
			ctx := c
			return &ctx, nil
		}
	}
	return nil, fmt.Errorf("current context %q not found in saved contexts", store.Current)
}

// SaveSession stores session data on the named context.
func SaveSession(store *File, name string, session *Session) error {
	for i := range store.Contexts {
		if store.Contexts[i].Name == name {
			store.Contexts[i].Session = session
			return nil
		}
	}
	return fmt.Errorf("context %q not found in saved contexts", name)
}

// ClearSession removes session data from the named context.
func ClearSession(store *File, name string) bool {
	for i := range store.Contexts {
		if store.Contexts[i].Name == name {
			store.Contexts[i].Session = nil
			return true
		}
	}
	return false
}
