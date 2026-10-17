package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
)

func newMonitorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "monitor",
		Aliases: []string{"mon"},
		Short:   "Watch z21 broadcast events",
		Long: `Print subscribed z21 broadcast events as they arrive.

Subscribe first with "z21 sub add <NAME>", then run monitor. Uses the current
context session so the z21 command station sends events to this client. A
keep-alive every 30 seconds stops the z21 from dropping the session (and the
subscriptions) after a minute without requests.

Press Ctrl+C to stop.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)
			if app == nil || app.Client == nil {
				return fmt.Errorf("Z21 connection not initialized")
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return runMonitor(ctx, app.Client, cmd.OutOrStdout(), client.DefaultKeepAliveInterval)
		},
	}
	return cmd
}

// runMonitor prints the events read from c to w until ctx is done, sending a
// keep-alive every keepAlive meanwhile.
func runMonitor(ctx context.Context, c *client.Client, w io.Writer, keepAlive time.Duration) error {
	ctx, cancel := context.WithCancel(ctx)
	keepAliveDone := make(chan struct{})
	go func() {
		defer close(keepAliveDone)
		_ = c.KeepAlive(ctx, keepAlive)
	}()
	defer func() {
		cancel()
		<-keepAliveDone
	}()

	fmt.Fprintln(w, "Waiting for Z21 events ...")
	for {
		msgs, err := c.ReadPacket(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("monitor %s: %w", c.Addr(), err)
		}

		for _, msg := range msgs {
			if line, ok := formatMonitorEvent(msg); ok {
				fmt.Fprintln(w, line)
			}
		}
	}
}
