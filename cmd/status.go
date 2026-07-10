package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newStatusCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show z21 track status",
		Long:  "Queries LAN_X_GET_STATUS (spec §2.4). Without --host, uses the current context. With --host, queries directly without saving a session.",
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				msgs, err := c.Call(ctx, protocol.GetXStatus())
				if err != nil {
					return fmt.Errorf("status %s: %w", c.Addr(), err)
				}

				xStatus, err := protocol.XStatusFromMessages(msgs)
				if err != nil {
					return fmt.Errorf("status %s: %w", c.Addr(), err)
				}

				printZ21TrackStatus(xStatus.CentralState)
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
