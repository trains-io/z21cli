package cmd

import (
	"time"

	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show aggregated z21 device information",
		Long: `Query z21 command station identity and capability information.

Sends one combined UDP datagram with LAN_GET_SERIAL_NUMBER, LAN_GET_HWINFO,
LAN_X_GET_VERSION, LAN_X_GET_FIRMWARE_VERSION, and LAN_GET_CODE (spec §1.3).
Replies may arrive in separate UDP datagrams and are collected until all
requested fields are answered or the request times out.`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			host, port, timeout, err := infoDial(cmd)
			if err != nil {
				return err
			}
			return runInfoAll(cmd, host, port, timeout)
		},
	}

	registerZ21PersistentDialFlags(cmd, &host, &port, &timeout)
	return cmd
}
