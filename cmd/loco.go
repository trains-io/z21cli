package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

const maxLocoAddress = 9999

func newLocoCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "loco",
		Short: "Drive locomotives",
		Long: `Read and drive locomotives (spec §4). Without --host, uses the current context.

Every command asks for the loco with LAN_X_GET_LOCO_INFO, which subscribes this
client to its changes (at most 16 locos per client).`,
	}

	registerZ21PersistentDialFlags(cmd, &host, &port, &timeout)
	cmd.AddCommand(
		newLocoInfoCmd(&host, &port, &timeout),
		newLocoDriveCmd(&host, &port, &timeout),
		newLocoFuncCmd(&host, &port, &timeout),
		newLocoStopCmd(&host, &port, &timeout),
	)
	requireSubcommand(cmd)
	return cmd
}

func newLocoInfoCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "info ADDR",
		Short: "Show a locomotive's speed, direction and functions",
		Long:  "Query LAN_X_GET_LOCO_INFO (spec §4.1) for loco address ADDR (1-9999).",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseLocoAddress(args[0])
			if err != nil {
				return err
			}
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				info, err := callLoco(ctx, c, addr)
				if err != nil {
					return fmt.Errorf("loco info %s: %w", c.Addr(), err)
				}
				printLocoInfo(info)
				return nil
			})
		},
	}
}

func newLocoDriveCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	var (
		forward, reverse bool
		steps            int
	)

	cmd := &cobra.Command{
		Use:   "drive ADDR SPEED",
		Short: "Set a locomotive's speed and direction",
		Long: `Send LAN_X_SET_LOCO_DRIVE (spec §4.2) and show the loco's new state.

SPEED is a speed step: 0 stops, at most 14, 28 or 126 depending on the speed
step mode. The loco keeps its direction and speed step mode unless
--forward, --reverse or --steps are given.`,
		Example: `  z21 loco drive 3 40
  z21 loco drive 3 0
  z21 loco drive 1234 10 --reverse --steps 28`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseLocoAddress(args[0])
			if err != nil {
				return err
			}
			step, err := strconv.ParseUint(args[1], 10, 8)
			if err != nil {
				return fmt.Errorf("invalid SPEED %q: want a speed step from 0 to 126", args[1])
			}
			var speedSteps byte
			stepsGiven := cmd.Flags().Changed("steps")
			if stepsGiven {
				if speedSteps, err = parseLocoSpeedSteps(steps); err != nil {
					return err
				}
			}
			directionGiven := forward || reverse

			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				if !stepsGiven || !directionGiven {
					info, err := callLoco(ctx, c, addr)
					if err != nil {
						return fmt.Errorf("loco drive %s: %w", c.Addr(), err)
					}
					if !stepsGiven {
						speedSteps = protocol.LocoDriveSpeedSteps(info.SpeedSteps)
					}
					if !directionGiven {
						forward = info.Forward
					}
				}
				speed, err := protocol.EncodeLocoSpeed(speedSteps, uint8(step))
				if err != nil {
					return fmt.Errorf("loco drive: %w", err)
				}
				info, err := callLoco(ctx, c, addr, protocol.SetLocoDrive(addr, speedSteps, forward, speed))
				if err != nil {
					return fmt.Errorf("loco drive %s: %w", c.Addr(), err)
				}
				printLocoInfo(info)
				return nil
			})
		},
	}

	cmd.Flags().BoolVar(&forward, "forward", false, "drive forward")
	cmd.Flags().BoolVar(&reverse, "reverse", false, "drive in reverse")
	cmd.Flags().IntVar(&steps, "steps", 0, "speed step mode: 14, 28 or 128 (default: the loco's)")
	cmd.MarkFlagsMutuallyExclusive("forward", "reverse")
	return cmd
}

func newLocoFuncCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "func ADDR N=on|off|toggle...",
		Short: "Switch locomotive functions",
		Long: `Send LAN_X_SET_LOCO_FUNCTION (spec §4.3.1) for each function N (0-31) and
show the loco's new state.`,
		Example: `  z21 loco func 3 0=on
  z21 loco func 3 1=toggle 2=off`,
		Args: wrapArgs(cobra.MinimumNArgs(2)),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseLocoAddress(args[0])
			if err != nil {
				return err
			}
			reqs := make([]protocol.Message, 0, len(args)-1)
			for _, arg := range args[1:] {
				fn, action, err := parseLocoFunction(arg)
				if err != nil {
					return err
				}
				reqs = append(reqs, protocol.SetLocoFunction(addr, fn, action))
			}
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				info, err := callLoco(ctx, c, addr, reqs...)
				if err != nil {
					return fmt.Errorf("loco func %s: %w", c.Addr(), err)
				}
				printLocoInfo(info)
				return nil
			})
		},
	}
}

func newLocoStopCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	return &cobra.Command{
		Use:   "stop ADDR",
		Short: "Emergency stop one locomotive",
		Long: `Send LAN_X_SET_LOCO_E_STOP (spec §4.5) and show the loco's new state. The
loco stops at once and keeps its direction; "z21 loco drive" ends the stop.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := parseLocoAddress(args[0])
			if err != nil {
				return err
			}
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				info, err := callLoco(ctx, c, addr, protocol.SetLocoEStop(addr))
				if err != nil {
					return fmt.Errorf("loco stop %s: %w", c.Addr(), err)
				}
				printLocoInfo(info)
				return nil
			})
		},
	}
}

// callLoco sends reqs, which get no reply, followed by LAN_X_GET_LOCO_INFO in
// one datagram, and returns the loco's state after reqs.
func callLoco(ctx context.Context, c *client.Client, addr uint16, reqs ...protocol.Message) (protocol.LocoInfo, error) {
	msgs, err := c.Call(ctx, append(reqs, protocol.GetLocoInfo(addr))...)
	if err != nil {
		return protocol.LocoInfo{}, err
	}
	return protocol.LocoInfoFromMessages(msgs)
}

func parseLocoAddress(s string) (uint16, error) {
	v, err := strconv.ParseUint(s, 10, 16)
	if err != nil || v == 0 || v > maxLocoAddress {
		return 0, fmt.Errorf("invalid loco address %q: want 1 to %d", s, maxLocoAddress)
	}
	return uint16(v), nil
}

func parseLocoSpeedSteps(steps int) (byte, error) {
	switch steps {
	case 14:
		return protocol.LocoSpeedSteps14, nil
	case 28:
		return protocol.LocoSpeedSteps28, nil
	case 128:
		return protocol.LocoSpeedSteps128, nil
	default:
		return 0, fmt.Errorf("invalid --steps %d: want 14, 28 or 128", steps)
	}
}

func parseLocoFunction(s string) (uint8, protocol.LocoFunctionAction, error) {
	num, state, ok := strings.Cut(s, "=")
	fn, err := strconv.ParseUint(num, 10, 8)
	if !ok || err != nil || fn > 31 {
		return 0, 0, fmt.Errorf("invalid function %q: want N=on, N=off or N=toggle with N from 0 to 31", s)
	}
	switch state {
	case "on":
		return uint8(fn), protocol.LocoFunctionOn, nil
	case "off":
		return uint8(fn), protocol.LocoFunctionOff, nil
	case "toggle":
		return uint8(fn), protocol.LocoFunctionToggle, nil
	default:
		return 0, 0, fmt.Errorf("invalid function %q: want N=on, N=off or N=toggle with N from 0 to 31", s)
	}
}
