package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
)

func newSerialCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "serial",
		Short: "Show z21 serial number",
		Long: `Queries LAN_GET_SERIAL_NUMBER (spec §2.1).

Without --host, uses the current context. With --host, queries directly without saving a session.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, port, timeout, err := infoDial(cmd)
			if err != nil {
				return err
			}
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				info, err := fetchSerialOnly(ctx, c)
				if err != nil {
					return fmt.Errorf("serial %s: %w", c.Addr(), err)
				}
				fmt.Println(info.Serial)
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
