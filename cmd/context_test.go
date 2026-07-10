package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/trains-io/z21cli/internal/contextstore"
)

func TestContextAddDefaultsHost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cmd := newContextAddCmd()
	cmd.SetArgs([]string{"local"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	store, err := contextstore.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if len(store.Contexts) != 1 {
		t.Fatalf("contexts = %d, want 1", len(store.Contexts))
	}
	if store.Contexts[0].Host != "127.0.0.1" {
		t.Fatalf("host = %q, want 127.0.0.1", store.Contexts[0].Host)
	}
	if store.Contexts[0].Port != 21105 {
		t.Fatalf("port = %d, want 21105", store.Contexts[0].Port)
	}
}

func TestContextAddPositionalHost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cmd := newContextAddCmd()
	cmd.SetArgs([]string{"remote", "192.168.2.6"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	store, err := contextstore.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if store.Contexts[0].Host != "192.168.2.6" {
		t.Fatalf("host = %q, want 192.168.2.6", store.Contexts[0].Host)
	}
}

func TestContextAddHostFlagOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cmd := newContextAddCmd()
	cmd.SetArgs([]string{"lab", "--host", "10.0.0.5"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	store, err := contextstore.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if store.Contexts[0].Host != "10.0.0.5" {
		t.Fatalf("host = %q, want 10.0.0.5", store.Contexts[0].Host)
	}
}

func TestContextAddRejectsHostArgAndFlag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cmd := newContextAddCmd()
	cmd.SetArgs([]string{"dup", "192.168.2.6", "--host", "10.0.0.5"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when host is given as both argument and flag")
	}
}

func TestContextAddDescription(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cmd := newContextAddCmd()
	cmd.SetArgs([]string{"sim", "--description", "Local Z21 simulator"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	store, err := contextstore.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if store.Contexts[0].Description != "Local Z21 simulator" {
		t.Fatalf("description = %q, want Local Z21 simulator", store.Contexts[0].Description)
	}

	got := captureStdout(t, func() {
		showCmd := newContextShowCmd()
		showCmd.SetArgs(nil)
		if err := showCmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})
	if !bytes.Contains([]byte(got), []byte("Description")) || !bytes.Contains([]byte(got), []byte("Local Z21 simulator")) {
		t.Fatalf("show output missing description:\n%s", got)
	}
}

func TestContextAddCustomPort(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	cmd := newContextAddCmd()
	cmd.SetArgs([]string{"sim", "--host", "127.0.0.1", "--port", "21107"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	store, err := contextstore.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if store.Contexts[0].Port != 21107 {
		t.Fatalf("port = %d, want 21107", store.Contexts[0].Port)
	}
}

func TestContextAddPrintsDefaultHost(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	got := captureStdout(t, func() {
		cmd := newContextAddCmd()
		cmd.SetArgs([]string{"sim"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	})

	if !bytes.Contains([]byte(got), []byte("127.0.0.1:21105")) {
		t.Fatalf("output missing default host:\n%s", got)
	}

	path := filepath.Join(dir, ".z21_contexts.json")
	if !bytes.Contains([]byte(got), []byte(path)) {
		t.Fatalf("output missing config file path %q:\n%s", path, got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected contexts file at %s: %v", path, err)
	}
}

func TestContextCommandsPrintConfigFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	path := filepath.Join(dir, ".z21_contexts.json")

	add := newContextAddCmd()
	add.SetArgs([]string{"sim"})
	if err := add.Execute(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		run  func() error
	}{
		{
			name: "list",
			run: func() error {
				cmd := newContextListCmd()
				cmd.SetArgs(nil)
				return cmd.Execute()
			},
		},
		{
			name: "show",
			run: func() error {
				cmd := newContextShowCmd()
				cmd.SetArgs(nil)
				return cmd.Execute()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := captureStdout(t, func() {
				if err := tt.run(); err != nil {
					t.Fatal(err)
				}
			})
			if !bytes.Contains([]byte(got), []byte(path)) {
				t.Fatalf("output missing config file path %q:\n%s", path, got)
			}
		})
	}
}
