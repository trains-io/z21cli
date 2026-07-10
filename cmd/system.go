package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newSystemCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "system",
		Short: "Show z21 system state",
		Long:  "Queries LAN_SYSTEMSTATE_GETDATA (spec §2.19). Without --host, uses the current context. With --host, queries directly without saving a session.",
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runZ21Op(cmd, host, port, timeout, func(ctx context.Context, c *client.Client) error {
				msgs, err := c.Call(ctx, protocol.SystemStateGetData())
				if err != nil {
					return fmt.Errorf("system %s: %w", c.Addr(), err)
				}

				state, err := protocol.SystemStateFromMessages(msgs)
				if err != nil {
					return fmt.Errorf("system %s: %w", c.Addr(), err)
				}

				printSystemState(state)
				return nil
			})
		},
	}

	registerZ21DialFlags(cmd, &host, &port, &timeout)
	return cmd
}

func printSystemState(state protocol.SystemState) {
	printLabelValueTable("METRIC", "VALUE",
		labeledValue{"Main current", fmt.Sprintf("%d mA", state.MainCurrent)},
		labeledValue{"Prog current", fmt.Sprintf("%d mA", state.ProgCurrent)},
		labeledValue{"Filtered main", fmt.Sprintf("%d mA", state.FilteredMainCurrent)},
		labeledValue{"Temperature", fmt.Sprintf("%d °C", state.Temperature)},
		labeledValue{"Supply voltage", fmt.Sprintf("%d mV", state.SupplyVoltage)},
		labeledValue{"VCC voltage", fmt.Sprintf("%d mV", state.VCCVoltage)},
		labeledValue{"Central state", fmt.Sprintf("0x%02x (%s)", state.CentralState, protocol.FormatXStatusFlags(state.CentralState))},
		labeledValue{"Central state ex", fmt.Sprintf("0x%02x (%s)", state.CentralStateEx, protocol.FormatCentralStateExFlags(state.CentralStateEx))},
		labeledValue{"Capabilities", fmt.Sprintf("0x%02x (%s)", state.Capabilities, protocol.FormatCapabilities(state.Capabilities))},
	)
}
