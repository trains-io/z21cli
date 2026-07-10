package cmd

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21cli/internal/contextstore"
)

func TestClearCurrentContextSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	store := &contextstore.File{
		Current: "sim",
		Contexts: []contextstore.Info{{
			Name: "sim",
			Host: "127.0.0.1",
			Port: 21105,
			Session: &contextstore.Session{
				LocalHost: "127.0.0.1",
				LocalPort: 54321,
			},
		}},
	}
	if err := contextstore.Save("", store); err != nil {
		t.Fatal(err)
	}

	root := &cobra.Command{Use: "z21"}
	cmd := &cobra.Command{Use: "logoff"}
	root.AddCommand(cmd)
	app := &AppContext{
		Context: contextstore.Info{Name: "sim"},
	}
	cmd.SetContext(withAppContext(context.Background(), app))

	if err := clearCurrentContextSession(cmd); err != nil {
		t.Fatal(err)
	}
	if app.Context.Session != nil {
		t.Fatal("expected in-memory session to be cleared")
	}

	loaded, err := contextstore.Load("")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Contexts[0].Session != nil {
		t.Fatal("expected saved session to be cleared")
	}
}

func withAppContext(parent context.Context, app *AppContext) context.Context {
	return context.WithValue(parent, ctxKey{}, app)
}
