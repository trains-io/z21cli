package cmd

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestTraceEnabledOnRootFlag(t *testing.T) {
	root := &cobra.Command{Use: "z21"}
	registerTraceFlag(root)
	ping := &cobra.Command{Use: "ping"}
	root.AddCommand(ping)

	if traceEnabled(ping) {
		t.Fatal("expected trace disabled by default")
	}

	if err := root.PersistentFlags().Set("trace", "true"); err != nil {
		t.Fatal(err)
	}
	if !traceEnabled(ping) {
		t.Fatal("expected trace enabled on child command")
	}
}
