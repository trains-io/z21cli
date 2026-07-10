package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/discovery"
)

func newDiscoverCmd() *cobra.Command {
	var (
		port        int
		timeout     time.Duration
		output      string
		concurrency int
	)

	cmd := &cobra.Command{
		Use:   "discover IFACE|NETWORK",
		Short: "Scan the local network for z21 devices",
		Long: `Scan a local network for reachable z21 command stations.

Provide either a network interface name (for example "eth0") or a CIDR network
(for example "192.168.2.0/24"). Each host is probed with LAN_GET_HWINFO.`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			network, err := discovery.ParseTarget(args[0])
			if err != nil {
				return err
			}

			if output == "normal" {
				fmt.Printf("Scanning network %q (port: %d) ...\n", network, port)
			}

			devices, err := discovery.Scan(cmd.Context(), network, discovery.Options{
				Port:        port,
				Timeout:     timeout,
				Concurrency: concurrency,
				Observer:    discoveryObserver(cmd),
			})
			if err != nil {
				return err
			}

			return printDiscoverResults(devices, output)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 21105, "UDP port to probe")
	cmd.Flags().DurationVar(&timeout, "timeout", 2*time.Second, "Probe timeout per host")
	cmd.Flags().StringVarP(&output, "output", "o", "normal", "Output format: short, normal, json")
	cmd.Flags().IntVar(&concurrency, "concurrency", 200, "Maximum concurrent probes")

	return cmd
}

func printDiscoverResults(devices []discovery.Device, output string) error {
	switch strings.ToLower(output) {
	case "short":
		for _, device := range devices {
			fmt.Println(device.IP)
		}
	case "normal":
		fmt.Printf("Found %d Z21 device(s):\n", len(devices))
		rows := make([][]string, 0, len(devices))
		for _, device := range devices {
			rows = append(rows, []string{
				device.IP.String(),
				fmt.Sprintf("%d", device.Port),
				fmt.Sprintf("0x%08x", device.HWInfo.HwType),
				fmt.Sprintf("0x%08x", device.HWInfo.FirmwareVersion),
			})
		}
		printTable("", []string{"IP", "PORT", "HW", "FW"}, rows)
	case "json":
		type jsonDevice struct {
			IP       string `json:"ip"`
			Port     int    `json:"port"`
			Reachable bool   `json:"reachable"`
			HwType   uint32 `json:"hw_type"`
			Firmware uint32 `json:"firmware"`
		}

		out := make([]jsonDevice, 0, len(devices))
		for _, device := range devices {
			out = append(out, jsonDevice{
				IP:        device.IP.String(),
				Port:      device.Port,
				Reachable: true,
				HwType:    device.HWInfo.HwType,
				Firmware:  device.HWInfo.FirmwareVersion,
			})
		}

		data, err := json.Marshal(out)
		if err != nil {
			return fmt.Errorf("discover: marshal JSON: %w", err)
		}
		fmt.Println(string(data))
	default:
		return fmt.Errorf("discover: invalid output format %q (valid: short, normal, json)", output)
	}
	return nil
}
