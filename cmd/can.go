package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

const (
	DEFAULT_SCAN_TIMEOUT time.Duration = 2 * time.Second
)

var canCmd = &cobra.Command{
	Use:   "can",
	Short: "Manage CAN Bus",
}

// ---------- subcommands ----------

// discover [--timeout | -t SECONDS]
var canDiscoverCmd = &cobra.Command{
	Use:     "discover",
	Aliases: []string{"d"},
	Short:   "Discover and list all CAN devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		timeout, _ := cmd.Flags().GetDuration("timeout")

		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}
		events := app.Conn.Events()

		_, err := Req(app.Conn, &z21.CanDetector{NetworkID: z21.CAN_BROADCAST_NID})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		devices := map[uint16]z21.Detector{}

		fmt.Printf("Discover CAN devices (timeout: %s) ...\n", timeout)
		for {
			select {
			case <-ctx.Done():
				printCanDevices(devices)
				return nil
			case ev := <-events:
				switch v := ev.(type) {
				case *z21.CanDetector:
					if v.Type == z21.CANMessageTypeStatus {
						dev, exists := devices[v.NetworkID]
						if !exists {
							dev = z21.Detector{
								NetworkID: v.NetworkID,
								Address:   v.Address,
								Ports:     []z21.DetectorPort{},
							}
						}
						dev.Ports = append(dev.Ports, z21.DetectorPort{Index: v.Port})
						devices[v.NetworkID] = dev
					}
				}
			}
		}
	},
}

func printCanDevices(devices map[uint16]z21.Detector) {
	if len(devices) == 0 {
		fmt.Printf("No devices found\n")
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.AppendHeader(table.Row{"NetID", "Addr", "Port(s)"})
	for _, d := range devices {
		t.AppendRow(
			table.Row{
				fmt.Sprintf("0x%04X", d.NetworkID),
				fmt.Sprintf("%d", d.Address),
				formatPortsAsRange(d.Ports),
			},
		)
	}
	t.Render()
}

func printCanDeviceInfo(d *z21.Detector) {
	if d == nil {
		fmt.Printf("Device not found\n")
		return
	}

	fmt.Printf("Device: 0x%04X (address: %d)\n", d.NetworkID, d.Address)
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.AppendHeader(table.Row{"Port", "Status"})
	for _, p := range d.Ports {
		t.AppendRow(
			table.Row{
				formatPortIndex(p.Index),
				formatPortStatus(p.Status),
			},
		)
	}
	t.Render()

}

func formatPortIndex(index uint8) string {
	return fmt.Sprintf("%d", index+1)
}

func formatPortStatus(status uint16) string {
	if status == z21.FREE || status == z21.FREE_NOVOLT {
		return "free"
	} else {
		return "busy"
	}
}

func formatPortsAsRange(ports []z21.DetectorPort) string {
	if len(ports) == 0 {
		return ""
	}

	var parts []string
	start := ports[0].Index
	prev := ports[0].Index

	for _, n := range ports[1:] {
		if n.Index == prev+1 {
			prev = n.Index
			continue
		}

		parts = append(parts, formatRange(start, prev))
		start = n.Index
		prev = n.Index
	}
	parts = append(parts, formatRange(start, prev))
	return strings.Join(parts, ",")
}

func formatRange(start, previous uint8) string {
	// shift to 1-based output
	start++
	previous++

	if start == previous {
		return fmt.Sprintf("%d", start)
	}
	return fmt.Sprintf("%d-%d", start, previous)
}

// info NETID
var canInfoCmd = &cobra.Command{
	Use:     "info",
	Aliases: []string{"i"},
	Short:   "Show CAN device information",
	RunE: func(cmd *cobra.Command, args []string) error {
		hexstr := args[0]
		val, err := strconv.ParseUint(hexstr, 0, 16)
		if err != nil {
			return err
		}
		netid := uint16(val)
		timeout, _ := cmd.Flags().GetDuration("timeout")

		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}
		events := app.Conn.Events()

		_, err = Req(app.Conn, &z21.CanDetector{NetworkID: netid})
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		var device *z21.Detector

		for {
			select {
			case <-ctx.Done():
				printCanDeviceInfo(device)
				return nil
			case ev := <-events:
				switch v := ev.(type) {
				case *z21.CanDetector:
					if v.Type == z21.CANMessageTypeStatus {
						if device == nil {
							device = &z21.Detector{
								NetworkID: v.NetworkID,
								Address:   v.Address,
								Ports:     []z21.DetectorPort{},
							}
						}
						device.Ports = append(device.Ports, z21.DetectorPort{Index: v.Port, Status: v.Value1})
					}
				}
			}
		}
	},
}

// set NETID --addr <ADDR>
var canSetCmd = &cobra.Command{
	Use:     "set",
	Aliases: []string{"s"},
	Short:   "Configure CAN device",
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}

// ---------- init ----------

func init() {
	canCmd.AddCommand(
		canDiscoverCmd,
		canInfoCmd,
		canSetCmd,
	)

	canDiscoverCmd.Flags().DurationP("timeout", "t", DEFAULT_SCAN_TIMEOUT, "timeout in seconds")
	canInfoCmd.Flags().DurationP("timeout", "t", DEFAULT_SCAN_TIMEOUT, "timeout in seconds")
}
