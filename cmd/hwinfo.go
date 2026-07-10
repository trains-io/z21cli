package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
)

func newHWInfoCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "hwinfo",
		Short: "Show z21 hardware type and firmware version",
		Long: `Queries LAN_GET_HWINFO (spec §2.20).

Without --host, uses the current context. With --host, queries directly without saving a session.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, port, timeout, err := infoDial(cmd)
			if err != nil {
				return err
			}
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				info, err := fetchHWOnly(ctx, c)
				if err != nil {
					return fmt.Errorf("hwinfo %s: %w", c.Addr(), err)
				}
				fmt.Printf("hw=0x%08x fw=0x%08x\n", info.HW.HwType, info.HW.FirmwareVersion)
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
