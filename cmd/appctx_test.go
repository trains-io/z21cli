package cmd

import (
	"testing"

	"github.com/spf13/cobra"
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
