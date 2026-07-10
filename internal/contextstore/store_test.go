package contextstore

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contexts.json")

	original := &File{
		Current: "sim",
		Contexts: []Info{
			{
				Name:        "sim",
				Host:        "127.0.0.1",
				Port:        21105,
				Description: "Local simulator",
				Session: &Session{
					LocalHost: "127.0.0.1",
					LocalPort: 54321,
				},
			},
		},
	}

	if err := Save(path, original); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if loaded.Current != original.Current {
		t.Fatalf("Current = %q, want %q", loaded.Current, original.Current)
	}
	if len(loaded.Contexts) != 1 {
		t.Fatalf("len(Contexts) = %d, want 1", len(loaded.Contexts))
	}
	if loaded.Contexts[0].Session.LocalPort != 54321 {
		t.Fatalf("Session.LocalPort = %d, want 54321", loaded.Contexts[0].Session.LocalPort)
	}
	if loaded.Contexts[0].Description != "Local simulator" {
		t.Fatalf("Description = %q, want Local simulator", loaded.Contexts[0].Description)
	}
}

func TestCurrent(t *testing.T) {
	store := &File{
		Current: "demo",
		Contexts: []Info{
			{Name: "demo", Host: "192.168.0.1", Port: 21105},
		},
	}

	ctx, err := Current(store)
	if err != nil {
		t.Fatalf("Current() error = %v", err)
	}
	if ctx.Name != "demo" {
		t.Fatalf("Name = %q, want demo", ctx.Name)
	}
}
