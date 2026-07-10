package cmd

import (
	"testing"
)

func TestSerialCmdNeedsAppContext(t *testing.T) {
	cmd := newSerialCmd()
	if !needsAppContext(cmd, nil) {
		t.Fatal("expected serial to need app context without --host")
	}
	if err := cmd.Flags().Set("host", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if needsAppContext(cmd, nil) {
		t.Fatal("expected serial to skip app context with --host")
	}
}
