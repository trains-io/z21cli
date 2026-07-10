package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newVersionCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show z21 X-Bus protocol version",
		Long: `Queries LAN_X_GET_VERSION (spec §2.3).

Without --host, uses the current context. With --host, queries directly without saving a session.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, port, timeout, err := infoDial(cmd)
			if err != nil {
				return err
			}
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				info, err := fetchVersionOnly(ctx, c)
				if err != nil {
					return fmt.Errorf("version %s: %w", c.Addr(), err)
				}
				fmt.Println(protocol.FormatXBusVersion(info.XVersion.XBusVersion))
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
