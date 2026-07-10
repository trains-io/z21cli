package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newPingCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Check connectivity to a z21 command station",
		Long:  "Queries LAN_GET_SERIAL_NUMBER and LAN_GET_HWINFO (spec §2.1, §2.20). Without --host, uses the current context. With --host, pings directly without saving a session.",
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				msgs, err := c.Call(ctx, protocol.GetSerialNumber(), protocol.GetHWInfo())
				if err != nil {
					return fmt.Errorf("ping %s: %w", c.Addr(), err)
				}

				serial, ok := protocol.SerialFromMessages(msgs)
				if !ok {
					return fmt.Errorf("ping %s: no LAN_GET_SERIAL_NUMBER reply in response", c.Addr())
				}

				hw, err := protocol.HWInfoFromMessages(msgs)
				if err != nil {
					return fmt.Errorf("ping %s: %w", c.Addr(), err)
				}

				fmt.Printf("pong from %s (serial=%s, hw=0x%08x fw=0x%08x)\n",
					c.Addr(), serial, hw.HwType, hw.FirmwareVersion)
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
