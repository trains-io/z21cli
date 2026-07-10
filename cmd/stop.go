package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newStopCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Trigger emergency stop",
		Long: `Send LAN_X_SET_STOP (spec §2.13).

Locomotives are halted immediately; track voltage remains on. The z21 replies
with LAN_X_BC_STOPPED when XpressNet broadcasts are enabled (flag 0x00000001).

Without --host, uses the current context.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				msgs, err := c.Call(ctx,
					protocol.SetBroadcastFlags(trackUpdateFlags(cmd)),
					protocol.SetStop(),
				)
				if err != nil {
					return fmt.Errorf("stop %s: %w", c.Addr(), err)
				}

				if err := protocol.BCStoppedFromMessages(msgs); err != nil {
					return fmt.Errorf("stop %s: %w", c.Addr(), err)
				}

				fmt.Println("Emergency stop activated.")
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
