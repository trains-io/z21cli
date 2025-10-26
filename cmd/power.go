package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

var powerCmd = &cobra.Command{
	Use:   "power",
	Short: "Manage track and booster power",
}

var powerOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Turn track power on",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		_, err := Req(app.Conn, &z21.BroadcastFlags{Flags: z21.Mask32(z21.TRACK_UPDATES)})
		if err != nil {
			return err
		}

		st, err := Req(app.Conn, &z21.TrackPower{On: true})
		if err != nil {
			return err
		}
		if !st.On {
			return fmt.Errorf("failed to turn power on")
		}
		fmt.Printf("Track power is turned on.\n")
		return nil
	},
}

var powerOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Turn track power off",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		_, err := Req(app.Conn, &z21.BroadcastFlags{Flags: z21.Mask32(z21.TRACK_UPDATES)})
		if err != nil {
			return err
		}

		st, err := Req(app.Conn, &z21.TrackPower{On: false})
		if err != nil {
			return err
		}
		if st.On {
			return fmt.Errorf("failed to turn power off")
		}
		fmt.Printf("Track power is turned off.\n")
		return nil
	},
}

var powerStopCmd = &cobra.Command{
	Use:     "stop",
	Aliases: []string{"halt"},
	Short:   "Emergency stop all locomotives",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		_, err := Req(app.Conn, &z21.BroadcastFlags{Flags: z21.Mask32(z21.TRACK_UPDATES)})
		if err != nil {
			return err
		}

		_, err = Req(app.Conn, &z21.Stop{})
		if err != nil {
			return err
		}
		fmt.Printf("Emergency stop is activated! The locomotives are stopped but the track voltage remains switched on.\n")
		return nil
	},
}

func init() {
	powerCmd.AddCommand(
		powerOnCmd,
		powerOffCmd,
		powerStopCmd,
	)
}
