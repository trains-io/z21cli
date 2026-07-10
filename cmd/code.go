package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func formatLockState(code byte) string {
	switch code {
	case protocol.CodeNoLock:
		return "No Lock (all features permitted)"
	case protocol.CodeZ21StartLocked:
		return "Locked (driving and switching blocked)"
	case protocol.CodeZ21StartUnlocked:
		return "Unlocked (driving and switching permitted)"
	default:
		return fmt.Sprintf("Unknown (0x%02x)", code)
	}
}

func newCodeCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "code",
		Short: "Show z21 feature lock state",
		Long: `Queries LAN_GET_CODE (spec §2.21).

Without --host, uses the current context. With --host, queries directly without saving a session.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, port, timeout, err := infoDial(cmd)
			if err != nil {
				return err
			}
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				info, err := fetchCodeOnly(ctx, c)
				if err != nil {
					return fmt.Errorf("code %s: %w", c.Addr(), err)
				}
				fmt.Println(formatLockState(info.LockCode))
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}
