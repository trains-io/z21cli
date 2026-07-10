package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
context session so the z21 command station sends events to this client.

Press Ctrl+C to stop.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)
			if app == nil || app.Client == nil {
				return fmt.Errorf("Z21 connection not initialized")
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return runMonitor(ctx, app.Client)
		},
	}
	return cmd
}

func runMonitor(ctx context.Context, c *client.Client) error {
	fmt.Println("Waiting for Z21 events ...")

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
				fmt.Println(line)
			}
		}
	}
}
