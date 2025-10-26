package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

var monitorCmd = &cobra.Command{
	Use:     "monitor",
	Aliases: []string{"mon"},
	Short:   "Watch Z21 broadcast events",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		fmt.Println("Waiting for Z21 events ...")
		for ev := range app.Conn.Events() {
			switch v := ev.(type) {
			case *z21.SysData:
				fmt.Printf(
					"[SYS] Main: %-5s Prog: %-5s Temp: %-5s Volt: %-5s (%-5s)\n",
					fmt.Sprintf("%dmA", v.MainCurrent),
					fmt.Sprintf("%dmA", v.ProgCurrent),
					fmt.Sprintf("%d°C", v.Temperature),
					fmt.Sprintf("%sV", mVToVoltString(v.SupplyVoltage)),
					fmt.Sprintf("%sV", mVToVoltString(v.VccVoltage)),
				)
			case *z21.TrackPower:
				fmt.Printf("[TRK] Power: %s\n",
					map[bool]string{true: "ON", false: "OFF"}[v.On],
				)
			}
		}

		return nil
	},
}
