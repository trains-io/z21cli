package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
)

func registerTraceFlag(cmd *cobra.Command) {
	cmd.PersistentFlags().Bool("trace", false, "hexdump z21 UDP requests and responses to stderr")
}

func traceEnabled(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if flag := c.PersistentFlags().Lookup("trace"); flag != nil {
			enabled, err := c.PersistentFlags().GetBool("trace")
			return err == nil && enabled
		}
	}
	return false
}

func configureClientTrace(cmd *cobra.Command, c *client.Client) {
	if !traceEnabled(cmd) {
		return
	}
	c.SetObserver(client.NewTracer(os.Stderr))
}

func discoveryObserver(cmd *cobra.Command) client.Observer {
	if !traceEnabled(cmd) {
		return nil
	}
	return client.NewTracer(os.Stderr)
}
