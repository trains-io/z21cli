package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newPowerCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "power",
		Short: "Manage track and booster power",
		Long:  "Control track voltage via LAN_X_SET_TRACK_POWER_ON/OFF (spec §2.21). Without --host, uses the current context.",
	}

	registerZ21PersistentDialFlags(cmd, &host, &port, &timeout)
	cmd.AddCommand(
		newPowerOnCmd(&host, &port, &timeout),
		newPowerOffCmd(&host, &port, &timeout),
		newPowerStatusCmd(&host, &port, &timeout),
	)
	requireSubcommand(cmd)
	return cmd
}

func newPowerOnCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "on",
		Short: "Turn track power on",
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPowerSet(cmd, *host, *port, *timeout, true)
		},
	}
}

func newPowerOffCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "off",
		Short: "Turn track power off",
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPowerSet(cmd, *host, *port, *timeout, false)
		},
	}
}

func newPowerStatusCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show track power state",
		Long:  "Queries LAN_X_GET_STATUS (spec §2.4) for track voltage and emergency stop.",
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				msgs, err := c.Call(ctx, protocol.GetXStatus())
				if err != nil {
					return fmt.Errorf("power status %s: %w", c.Addr(), err)
				}

				status, err := protocol.XStatusFromMessages(msgs)
				if err != nil {
					return fmt.Errorf("power status %s: %w", c.Addr(), err)
				}

				printPowerStatus(status.CentralState)
				return nil
			})
		},
	}
}

func runPowerSet(cmd *cobra.Command, host string, port int, timeout time.Duration, on bool) error {
	return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
		msgs, err := c.Call(ctx,
			protocol.SetBroadcastFlags(trackUpdateFlags(cmd)),
			protocol.SetTrackPower(on),
		)
		if err != nil {
			action := "off"
			if on {
				action = "on"
			}
			return fmt.Errorf("power %s %s: %w", action, c.Addr(), err)
		}

		got, err := protocol.TrackPowerFromMessages(msgs)
		if err != nil {
			action := "off"
			if on {
				action = "on"
			}
			return fmt.Errorf("power %s %s: %w", action, c.Addr(), err)
		}
		if got != on {
			if on {
				return fmt.Errorf("failed to turn track power on")
			}
			return fmt.Errorf("failed to turn track power off")
		}

		if on {
			fmt.Println("Track power is turned on.")
		} else {
			fmt.Println("Track power is turned off.")
		}
		return nil
	})
}

func trackUpdateFlags(cmd *cobra.Command) uint32 {
	flags := protocol.BroadcastFlagXpressNet
	if app := GetAppContext(cmd); app != nil && app.Context.Session != nil {
		flags = effectiveBroadcastFlags(app.Context.Session) | protocol.BroadcastFlagXpressNet
	}
	return flags
}

func printPowerStatus(centralState byte) {
	printLabelValueTable("POWER", "STATUS",
		labeledValue{"Track power", formatTrackVoltage(centralState)},
		labeledValue{"Emergency stop", formatEmergencyStop(centralState)},
	)
}
