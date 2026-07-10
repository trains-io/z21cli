package cmd

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newCanCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "can",
		Short: "Discover and configure CAN occupancy detectors",
		Long: `Manage z21 CAN occupancy detectors via LAN_CAN_DETECTOR (spec §10.1).

Module address changes use the undocumented maintenance tunnel (LAN header 0xC2) on UDP port 21106.`,
	}

	registerZ21PersistentDialFlags(cmd, &host, &port, &timeout)
	cmd.AddCommand(
		newCanDiscoverCmd(&host, &port, &timeout),
		newCanInfoCmd(&host, &port, &timeout),
		newCanSetCmd(&host, &timeout),
	)
	requireSubcommand(cmd)
	return cmd
}

func newCanDiscoverCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	var collect time.Duration

	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover CAN occupancy detectors",
		Long:  `Poll all CAN detectors with LAN_CAN_DETECTOR and NetID 0xD000 (spec §10.1).`,
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			window := collect
			if window <= 0 {
				window = *timeout
			}
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				fmt.Printf("Discover CAN devices (timeout: %s) ...\n", window)
				msgs, err := c.RequestCollect(ctx, protocol.GetAllCANDetectors(), window)
				if err != nil {
					return fmt.Errorf("can discover %s: %w", c.Addr(), err)
				}

				devices, err := summarizeCANDevices(msgs)
				if err != nil {
					return fmt.Errorf("can discover %s: %w", c.Addr(), err)
				}
				printCANDiscoverTable(devices)
				return nil
			})
		},
	}

	cmd.Flags().DurationVar(&collect, "collect", 0, "Reply collection window (default: --timeout)")
	return cmd
}

func newCanInfoCmd(host *string, port *int, timeout *time.Duration) *cobra.Command {
	var collect time.Duration

	cmd := &cobra.Command{
		Use:   "info NETID",
		Short: "Show CAN detector port status",
		Long:  `Poll one CAN detector by NetID (for example 0xDB04) via LAN_CAN_DETECTOR.`,
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			netID, err := parseNetID(args[0])
			if err != nil {
				return err
			}

			window := collect
			if window <= 0 {
				window = *timeout
			}
			return runZ21Op(cmd, *host, *port, *timeout, func(ctx context.Context, c *client.Client) error {
				msgs, err := c.RequestCollect(ctx, protocol.GetCANDetector(netID), window)
				if err != nil {
					return fmt.Errorf("can info %s: %w", c.Addr(), err)
				}

				reports, err := protocol.CANDetectorReportsFromMessages(msgs)
				if err != nil {
					return fmt.Errorf("can info %s: %w", c.Addr(), err)
				}

				printCANInfo(netID, reports)
				return nil
			})
		},
	}

	cmd.Flags().DurationVar(&collect, "collect", 0, "Reply collection window (default: --timeout)")
	return cmd
}

func newCanSetCmd(host *string, timeout *time.Duration) *cobra.Command {
	var (
		maintPort   int
		railcomCh2  bool
		noRailcomCh2 bool
	)

	cmd := &cobra.Command{
		Use:   "set NETID ADDRESS",
		Short: "Set CAN detector module address",
		Long: `Write a detector module address via the maintenance tunnel (LAN header 0xC2).

Sends to UDP port 21106 by default. The RailCom channel 2 forward flag is encoded only on
the commit packet; use --railcom-ch2 or --no-railcom-ch2 (default: enabled).`,
		Args: ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			netID, err := parseNetID(args[0])
			if err != nil {
				return err
			}

			addr, err := strconv.ParseUint(args[1], 10, 16)
			if err != nil || addr == 0 {
				return fmt.Errorf("invalid module address %q", args[1])
			}

			forward := true
			if noRailcomCh2 {
				forward = false
			}
			if railcomCh2 {
				forward = true
			}

			msg, err := protocol.SetCANDetectorModuleAddress(netID, uint16(addr), forward)
			if err != nil {
				return err
			}

			return runZ21Op(cmd, *host, maintPort, *timeout, func(ctx context.Context, c *client.Client) error {
				if err := c.Send(ctx, msg); err != nil {
					return fmt.Errorf("can set %s: %w", c.Addr(), err)
				}
				fmt.Printf("Module address set to %d on detector 0x%04X (maintenance port %d).\n", addr, netID, maintPort)
				return nil
			})
		},
	}

	cmd.Flags().IntVar(&maintPort, "maint-port", client.MaintenancePort, "z21 maintenance UDP port")
	cmd.Flags().BoolVar(&railcomCh2, "railcom-ch2", false, "Enable RailCom channel 2 forward to z21 on commit")
	cmd.Flags().BoolVar(&noRailcomCh2, "no-railcom-ch2", false, "Disable RailCom channel 2 forward to z21 on commit")
	return cmd
}

func parseNetID(s string) (uint16, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	s = strings.TrimPrefix(s, "0x")
	v, err := strconv.ParseUint(s, 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid CAN NetID %q", s)
	}
	if v == 0 {
		return 0, fmt.Errorf("CAN NetID must be non-zero")
	}
	return uint16(v), nil
}

type canDeviceSummary struct {
	NetID uint16
	Addr  uint16
	Ports map[uint8]struct{}
}

func summarizeCANDevices(msgs []protocol.Message) ([]canDeviceSummary, error) {
	byNetID := map[uint16]*canDeviceSummary{}

	for _, msg := range msgs {
		if msg.Header != protocol.HeaderLANCANDetector {
			continue
		}
		report, err := protocol.ParseCANDetector(msg.Data)
		if err != nil {
			return nil, err
		}
		if report.NetID == protocol.CANDetectorPollAll {
			continue
		}

		dev := byNetID[report.NetID]
		if dev == nil {
			dev = &canDeviceSummary{NetID: report.NetID, Ports: map[uint8]struct{}{}}
			byNetID[report.NetID] = dev
		}
		if dev.Addr == 0 {
			dev.Addr = report.Addr
		}
		dev.Ports[report.Port] = struct{}{}
	}

	if len(byNetID) == 0 {
		return nil, fmt.Errorf("no CAN detectors found")
	}

	out := make([]canDeviceSummary, 0, len(byNetID))
	for _, dev := range byNetID {
		out = append(out, *dev)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].NetID < out[j].NetID })
	return out, nil
}

func printCANDiscoverTable(devices []canDeviceSummary) {
	rows := make([][]string, 0, len(devices))
	for _, dev := range devices {
		rows = append(rows, []string{
			fmt.Sprintf("0x%04X", dev.NetID),
			fmt.Sprintf("%d", dev.Addr),
			formatCANPorts(dev.Ports),
		})
	}
	printTable("", []string{"NETID", "ADDR", "PORT(S)"}, rows)
}

func printCANInfo(netID uint16, reports []protocol.CANDetectorReport) {
	var moduleAddr uint16
	ports := make(map[uint8]string, 8)

	for _, report := range reports {
		if report.NetID != netID {
			continue
		}
		if moduleAddr == 0 {
			moduleAddr = report.Addr
		}
		if report.Type != protocol.CANDetectorTypeOccupancy {
			continue
		}
		ports[report.Port] = protocol.OccupancyStatusLabel(report.Value1)
	}

	fmt.Printf("Device: 0x%04X (address: %d)\n", netID, moduleAddr)

	rows := make([][]string, 0, 8)
	for port := uint8(0); port < 8; port++ {
		status, ok := ports[port]
		if !ok {
			status = "unknown"
		}
		rows = append(rows, []string{fmt.Sprintf("%d", port+1), status})
	}
	printTable("", []string{"PORT", "STATUS"}, rows)
}

func formatCANPorts(ports map[uint8]struct{}) string {
	if len(ports) == 0 {
		return ""
	}

	ordered := make([]int, 0, len(ports))
	for port := range ports {
		ordered = append(ordered, int(port))
	}
	sort.Ints(ordered)

	// Display 1-based port numbers to match the maintenance tool.
	display := make([]int, len(ordered))
	for i, port := range ordered {
		display[i] = port + 1
	}

	if len(display) == 8 && display[0] == 1 && display[7] == 8 {
		return "1-8"
	}

	parts := make([]string, len(display))
	for i, port := range display {
		parts[i] = strconv.Itoa(port)
	}
	return strings.Join(parts, ",")
}
