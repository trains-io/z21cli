package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

const (
	maxAccessoryAddress = 2047
	defaultTurnoutPulse = 100 * time.Millisecond
)

func newTurnoutCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "turnout",
		Short: "Switch turnouts",
		Long: `Read and switch turnouts (spec §5.1–5.3). Without --host, uses the current context.

ADDR is the turnout address as on the wire (0-2047): 0 is turnout 1 in the Z21 app.`,
	}

	registerZ21PersistentDialFlags(cmd, &host, &port, &timeout)
	cmd.AddCommand(
		newTurnoutGetCmd(&host, &port, &timeout),
		newTurnoutSetCmd(&host, &port, &timeout),
	)
	requireSubcommand(cmd)
	return cmd
}

func newTurnoutGetCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "get ADDR",
		Short: "Show the position of a turnout",
		Long:  "Query LAN_X_GET_TURNOUT_INFO (spec §5.1) for turnout address ADDR (0-2047).",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseAccessoryAddress(args[0])
			if err != nil {
				return err
			}
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				info, err := callTurnout(ctx, c, addr)
				if err != nil {
					return fmt.Errorf("turnout get %s: %w", c.Addr(), err)
				}
				printTurnoutInfo(info)
				return nil
			})
		},
	}
}

func newTurnoutSetCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	var (
		pulse time.Duration
		queue bool
	)

	cmd := &cobra.Command{
		Use:   "set ADDR 1|2",
		Short: "Switch a turnout to output 1 or 2",
		Long: `Send LAN_X_SET_TURNOUT (spec §5.2) and show the turnout's new position.

Like a throttle, it activates the output, waits --pulse and deactivates it
again, so the decoder's coil is not left powered. --pulse 0 leaves the output
activated. --queue sets the Q flag: the Z21 queues the command instead of
sending it to the track at once (firmware 1.24 and later).`,
		Example: `  z21 turnout set 0 2
  z21 turnout set 12 1 --pulse 250ms`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseAccessoryAddress(args[0])
			if err != nil {
				return err
			}
			var output2 bool
			switch args[1] {
			case "1":
			case "2":
				output2 = true
			default:
				return fmt.Errorf("invalid output %q: want 1 or 2", args[1])
			}
			if pulse < 0 || pulse >= *timeout {
				return fmt.Errorf("invalid --pulse %s: want 0 or more and less than --timeout %s", pulse, *timeout)
			}

			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				activate := protocol.SetTurnout(addr, protocol.TurnoutSwitch{Activate: true, Output2: output2, Queue: queue})
				reqs := []protocol.Message{activate}
				if pulse > 0 {
					if err := c.Send(ctx, activate); err != nil {
						return fmt.Errorf("turnout set %s: %w", c.Addr(), err)
					}
					if err := sleep(ctx, pulse); err != nil {
						return fmt.Errorf("turnout set %s: %w", c.Addr(), err)
					}
					reqs = []protocol.Message{protocol.SetTurnout(addr, protocol.TurnoutSwitch{Output2: output2, Queue: queue})}
				}
				info, err := callTurnout(ctx, c, addr, reqs...)
				if err != nil {
					return fmt.Errorf("turnout set %s: %w", c.Addr(), err)
				}
				printTurnoutInfo(info)
				return nil
			})
		},
	}

	cmd.Flags().DurationVar(&pulse, "pulse", defaultTurnoutPulse, "how long the output stays activated; 0 leaves it activated")
	cmd.Flags().BoolVar(&queue, "queue", false, "let the Z21 queue the command (Q flag)")
	return cmd
}

func newAccessoryCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "accessory",
		Short: "Set extended accessory decoders",
		Long: `Read and set extended accessory decoders, for example signals (spec §5.4–5.6).
Without --host, uses the current context.

ADDR is the accessory address as on the wire (0-2047).`,
	}

	registerZ21PersistentDialFlags(cmd, &host, &port, &timeout)
	cmd.AddCommand(
		newAccessoryGetCmd(&host, &port, &timeout),
		newAccessorySetCmd(&host, &port, &timeout),
	)
	requireSubcommand(cmd)
	return cmd
}

func newAccessoryGetCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "get ADDR",
		Short: "Show the value of an extended accessory decoder",
		Long:  "Query LAN_X_GET_EXT_ACCESSORY_INFO (spec §5.5) for accessory address ADDR (0-2047).",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseAccessoryAddress(args[0])
			if err != nil {
				return err
			}
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				info, err := callExtAccessory(ctx, c, addr)
				if err != nil {
					return fmt.Errorf("accessory get %s: %w", c.Addr(), err)
				}
				printExtAccessoryInfo(info)
				return nil
			})
		},
	}
}

func newAccessorySetCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "set ADDR VALUE",
		Short: "Send a value to an extended accessory decoder",
		Long: `Send LAN_X_SET_EXT_ACCESSORY (spec §5.4) and show the accessory's new value.

VALUE (0-255, decimal or 0x hex) is passed to the decoder as is, for example a
signal aspect.`,
		Example: `  z21 accessory set 3 0x12
  z21 accessory set 3 0`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseAccessoryAddress(args[0])
			if err != nil {
				return err
			}
			value, err := strconv.ParseUint(args[1], 0, 8)
			if err != nil {
				return fmt.Errorf("invalid VALUE %q: want 0 to 255, decimal or 0x hex", args[1])
			}
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				info, err := callExtAccessory(ctx, c, addr, protocol.SetExtAccessory(addr, byte(value)))
				if err != nil {
					return fmt.Errorf("accessory set %s: %w", c.Addr(), err)
				}
				printExtAccessoryInfo(info)
				return nil
			})
		},
	}
}

// callTurnout sends reqs, which get no reply, followed by
// LAN_X_GET_TURNOUT_INFO in one datagram, and returns the turnout after reqs.
func callTurnout(ctx context.Context, c *client.Client, addr uint16, reqs ...protocol.Message) (protocol.TurnoutInfo, error) {
	msgs, err := c.Call(ctx, append(reqs, protocol.GetTurnoutInfo(addr))...)
	if err != nil {
		return protocol.TurnoutInfo{}, err
	}
	return protocol.TurnoutInfoFromMessages(msgs)
}

// callExtAccessory sends reqs, which get no reply, followed by
// LAN_X_GET_EXT_ACCESSORY_INFO in one datagram, and returns the accessory
// after reqs.
func callExtAccessory(ctx context.Context, c *client.Client, addr uint16, reqs ...protocol.Message) (protocol.ExtAccessoryInfo, error) {
	msgs, err := c.Call(ctx, append(reqs, protocol.GetExtAccessoryInfo(addr))...)
	if err != nil {
		return protocol.ExtAccessoryInfo{}, err
	}
	return protocol.ExtAccessoryInfoFromMessages(msgs)
}

// sleep waits for d or until ctx is done.
func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func parseAccessoryAddress(s string) (uint16, error) {
	v, err := strconv.ParseUint(s, 0, 16)
	if err != nil || v > maxAccessoryAddress {
		return 0, fmt.Errorf("invalid address %q: want 0 to %d", s, maxAccessoryAddress)
	}
	return uint16(v), nil
}
