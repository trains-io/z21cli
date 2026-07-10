package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newFirmwareCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "firmware",
		Short: "Show z21 firmware version",
		Long: `Queries LAN_X_GET_FIRMWARE_VERSION (spec §2.15).

Without --host, uses the current context. With --host, queries directly without saving a session.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, port, timeout, err := infoDial(cmd)
			if err != nil {
				return err
			}
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				info, err := fetchFirmwareOnly(ctx, c)
				if err != nil {
					return fmt.Errorf("firmware %s: %w", c.Addr(), err)
				}
				fmt.Println(protocol.FormatXFirmwareVersion(info.XFirmware))
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
