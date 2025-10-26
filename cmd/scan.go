package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan CAN Bus",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		_, err := Req(app.Conn, &z21.CanDetector{Address: z21.CAN_BROADCAST_ADDR})
		if err != nil {
			return err
		}
		return nil
	},
}
