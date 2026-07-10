package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestStopNeedsAppContext(t *testing.T) {
	stopCmd := newStopCmd()
	if !needsAppContext(stopCmd, nil) {
		t.Fatal("expected stop to need app context")
	}

	if err := stopCmd.Flags().Set("host", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if needsAppContext(stopCmd, nil) {
		t.Fatal("expected stop with --host to skip app context")
	}
}

func TestStopCmdRegistered(t *testing.T) {
	root := &cobra.Command{Use: "z21"}
	root.AddCommand(newStopCmd())

	if _, _, err := root.Find([]string{"stop"}); err != nil {
		t.Fatalf("missing stop command: %v", err)
	}
}
