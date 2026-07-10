package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newLogoffCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "logoff",
		Short: "End the z21 client session",
		Long:  "Sends LAN_LOGOFF (spec §2.4). Without --host, uses the current context and clears its saved session. With --host, sends logoff on a direct connection without saving a session.",
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				if err := c.Send(ctx, protocol.Logoff()); err != nil {
					return fmt.Errorf("logoff %s: %w", c.Addr(), err)
				}
				fmt.Printf("Logged off from %s\n", c.Addr())
				return nil
			})
			if err != nil {
				return err
			}
			if usesDirectDial(cmd) {
				return nil
			}
			return clearCurrentContextSession(cmd)
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
