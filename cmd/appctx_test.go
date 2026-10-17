package cmd

import (
	"context"
	"net"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21cli/internal/contextstore"
)

func TestNeedsAppContext(t *testing.T) {
	root := &cobra.Command{Use: "z21"}
	contextCmd := &cobra.Command{Use: "context"}
	listCmd := &cobra.Command{Use: "list"}
	resetCmd := &cobra.Command{Use: "reset"}

	root.AddCommand(contextCmd, newPingCmd(), newInfoCmd(), newStatusCmd(), newSystemCmd())
	contextCmd.AddCommand(listCmd, resetCmd)

	tests := []struct {
		name  string
		cmd   func() *cobra.Command
		flags []string
		want  bool
	}{
		{"context list", func() *cobra.Command { return listCmd }, nil, false},
		{"context reset", func() *cobra.Command { return resetCmd }, nil, true},
		{"ping with host", newPingCmd, []string{"--host", "10.0.0.1"}, false},
		{"ping default context", newPingCmd, nil, true},
		{"serial default context", newSerialCmd, nil, true},
		{"serial with host", newSerialCmd, []string{"--host", "10.0.0.1"}, false},
		{"version default context", newVersionCmd, nil, true},
		{"version with host", newVersionCmd, []string{"--host", "10.0.0.1"}, false},
		{"firmware default context", newFirmwareCmd, nil, true},
		{"firmware with host", newFirmwareCmd, []string{"--host", "10.0.0.1"}, false},
		{"code default context", newCodeCmd, nil, true},
		{"code with host", newCodeCmd, []string{"--host", "10.0.0.1"}, false},
		{"hwinfo default context", newHWInfoCmd, nil, true},
		{"hwinfo with host", newHWInfoCmd, []string{"--host", "10.0.0.1"}, false},
		{"status with host", newStatusCmd, []string{"--host", "10.0.0.1"}, false},
		{"status default context", newStatusCmd, nil, true},
		{"system with host", newSystemCmd, []string{"--host", "10.0.0.1"}, false},
		{"system default context", newSystemCmd, nil, true},
		{"info default context", newInfoCmd, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.cmd()
			if tt.flags != nil {
				if err := cmd.ParseFlags(tt.flags); err != nil {
					t.Fatalf("ParseFlags() error = %v", err)
				}
			}
			if got := needsAppContext(cmd, nil); got != tt.want {
				t.Fatalf("needsAppContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsUnderContext(t *testing.T) {
	root := &cobra.Command{Use: "z21"}
	contextCmd := &cobra.Command{Use: "context"}
	listCmd := &cobra.Command{Use: "list"}
	root.AddCommand(contextCmd)
	contextCmd.AddCommand(listCmd)

	if !isUnderContext(listCmd) {
		t.Fatal("expected list command to be under context")
	}
	if isUnderContext(root) {
		t.Fatal("expected root not to be under context")
	}
}

func TestUsesDirectDial(t *testing.T) {
	pingCmd := newPingCmd()
	if usesDirectDial(pingCmd) {
		t.Fatal("expected no direct dial without --host")
	}
	if err := pingCmd.Flags().Set("host", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if !usesDirectDial(pingCmd) {
		t.Fatal("expected direct dial with --host")
	}
}

func TestInitAppContextPinnedPortInUse(t *testing.T) {
	// Held as by a running z21 monitor.
	holder, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	pinned := holder.LocalAddr().(*net.UDPAddr).Port

	tests := []struct {
		path    []string
		wantErr bool
	}{
		{path: []string{"status"}},
		{path: []string{"power", "on"}},
		{path: []string{"subscribe", "add"}, wantErr: true},
		{path: []string{"subscribe", "list"}, wantErr: true},
		{path: []string{"broadcast", "get"}, wantErr: true},
		{path: []string{"logoff"}, wantErr: true},
		{path: []string{"context", "reset"}, wantErr: true},
		{path: []string{"monitor"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.path, " "), func(t *testing.T) {
			saveContextWithSession(t, pinned)
			cmd := commandAt(tt.path...)

			err := InitAppContext(cmd)
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "stop monitor first") {
					t.Fatalf("InitAppContext = %v, want an error telling to stop monitor", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("InitAppContext = %v, want fallback to a free port", err)
			}
			app := GetAppContext(cmd)
			defer app.Client.Close()
			if got := app.Client.LocalAddr().(*net.UDPAddr).Port; got == pinned {
				t.Fatalf("local port = %d, want a free port other than the pinned one", got)
			}
			if app.Context.Session != nil {
				t.Fatal("app context has the context's session, want none on the fallback port")
			}
		})
	}
}

func TestInitAppContextUsesPinnedPort(t *testing.T) {
	free, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero})
	if err != nil {
		t.Fatal(err)
	}
	pinned := free.LocalAddr().(*net.UDPAddr).Port
	free.Close()

	saveContextWithSession(t, pinned)
	cmd := commandAt("subscribe", "add")
	if err := InitAppContext(cmd); err != nil {
		t.Fatalf("InitAppContext = %v", err)
	}
	app := GetAppContext(cmd)
	defer app.Client.Close()
	if got := app.Client.LocalAddr().(*net.UDPAddr).Port; got != pinned {
		t.Fatalf("local port = %d, want the pinned port %d", got, pinned)
	}
}

// saveContextWithSession stores a current context whose session pins
// localPort, in a fresh HOME.
func saveContextWithSession(t *testing.T, localPort int) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	store := &contextstore.File{
		Current: "sim",
		Contexts: []contextstore.Info{{
			Name:    "sim",
			Host:    "127.0.0.1",
			Port:    21105,
			Session: &contextstore.Session{LocalHost: "0.0.0.0", LocalPort: localPort},
		}},
	}
	if err := contextstore.Save("", store); err != nil {
		t.Fatal(err)
	}
}

// commandAt returns the last of a chain of commands below a z21 root, with a
// context as cobra sets one on Execute.
func commandAt(path ...string) *cobra.Command {
	cmd := &cobra.Command{Use: "z21"}
	for _, name := range path {
		sub := &cobra.Command{Use: name}
		cmd.AddCommand(sub)
		cmd = sub
	}
	cmd.SetContext(context.Background())
	return cmd
}
