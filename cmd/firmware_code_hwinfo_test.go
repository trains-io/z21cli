package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestFirmwareCmdNeedsAppContext(t *testing.T) {
	testQueryCmdNeedsAppContext(t, newFirmwareCmd)
}

func TestCodeCmdNeedsAppContext(t *testing.T) {
	testQueryCmdNeedsAppContext(t, newCodeCmd)
}

func TestHWInfoCmdNeedsAppContext(t *testing.T) {
	testQueryCmdNeedsAppContext(t, newHWInfoCmd)
}

func testQueryCmdNeedsAppContext(t *testing.T, newCmd func() *cobra.Command) {
	t.Helper()

	cmd := newCmd()
	if !needsAppContext(cmd, nil) {
		t.Fatal("expected command to need app context without --host")
	}
	if err := cmd.Flags().Set("host", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if needsAppContext(cmd, nil) {
		t.Fatal("expected command to skip app context with --host")
	}
}
