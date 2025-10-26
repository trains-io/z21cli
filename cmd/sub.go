package cmd

import (
	"fmt"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

const (
	ENABLED  string = "Yes"
	DISABLED string = "No"
)

var subs = []struct {
	flag        uint32
	name        string
	description string
}{
	{
		flag: z21.TRACK_UPDATES,
		name: "TRACK_UPDATES",
		description: `Receive broadcasts and info messages concerning driving and switching. 
The following events are concerned: 
  - track power (on/off)
  - track programming mode
  - track short circuit
  - emergency stop
  - loco info (loco address must be subscribed too) 
  - turnout info`,
	},
	{
		flag:        z21.FEEDBACK_UPDATES,
		name:        "FEEDBACK_UPDATES",
		description: `Receive R-Bus events from feedback devices.`,
	},
	{
		flag:        z21.RAILCOM_SUB_UPDATES,
		name:        "RAILCOM_SUB_UPDATES",
		description: `Receive RailCom events from subscribed locos.`,
	},
	{
		flag:        z21.FAST_CLOCK_UPDATES,
		name:        "FAST_CLOCK_UPDATES",
		description: `Receive fast clock time messages (from V1.43).`,
	},
	{
		flag:        z21.SYSTEM_UPDATES,
		name:        "SYSTEM_UPDATES",
		description: `Receive Z21 system status updates.`,
	},
	{
		flag: z21.LOCO_UPDATES,
		name: "LOCO_UPDATES",
		description: `Extends TRACK_UPDATES events without having to subscribe 
to the corresponding loco addresses, i.e. for all controlled locos! 
Due to the high network traffic, this flag must be used with caution.
  - V1.20..V1.23: events are sent for all locos
  - V1.24: events are sent only for modified locos`,
	},
	{
		flag:        z21.CAN_BOOSTER_UPDATES,
		name:        "CAN_BOOSTER_UPDATES",
		description: `Receive CAN bus booster events (from V1.41).`,
	},
	{
		flag: z21.RAILCOM_UPDATES,
		name: "RAILCOM_UPDATES",
		description: `Receive RailCom events without having to subscribe 
to the corresponding loco addresses, i.e. for all controlled locos! 
Due to the high network traffic, this flag must be used with caution.
(from V1.29)`,
	},
	{
		flag:        z21.CAN_DETECTOR_UPDATES,
		name:        "CAN_DETECTOR_UPDATES",
		description: `Receive CAN bus events from track occupancy detectors (from V1.30).`,
	},
	{
		flag:        z21.LOCONET_UPDATES,
		name:        "LOCONET_UPDATES",
		description: `Receive LocoNet events excluding loco and switch events (from V1.20).`,
	},
	{
		flag: z21.LOCONET_LOCO_UPDATES,
		name: "LOCONET_LOCO_UPDATES",
		description: `Receive LocoNet loco events: 
  - OPC_LOCO_SPD
  - OPC_LOCO_DIRF
  - OPC_LOCO_SND
  - OPC_LOCO_F912 
  - OPC_EXP_CMD
  (from V1.20)`,
	},
	{
		flag: z21.LOCONET_SWITCH_UPDATES,
		name: "LOCONET_SWITCH_UPDATES",
		description: `Receive LocoNet switch events: 
  - OPC_SW_REQ
  - OPC_SW_REP
  - OPC_SW_ACK
  - OPC_SW_STATE
  (from V1.20)`,
	},
	{
		flag:        z21.LOCONET_DETECTOR_UPDATES,
		name:        "LOCONET_DETECTOR_UPDATES",
		description: `Receive LocoNet events from track occupancy detectors (from V1.22).`,
	},
}

var subCmd = &cobra.Command{
	Use:   "sub",
	Short: "Manage Z21 subscriptions",
}

// ---------- subcommands ----------

// list
var subListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all subscribed Z21 events",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		f, err := Req(app.Conn, &z21.SubscribedBroadcastFlags{})
		if err != nil {
			return err
		}
		printSubscriptions(f.Flags)

		return nil
	},
}

// add NAME
var subAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Subscribe to a specific event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		name := args[0]
		if !hasSub(name) {
			return fmt.Errorf("unsupported subscription %q", name)
		}

		f, err := Req(app.Conn, &z21.SubscribedBroadcastFlags{})
		if err != nil {
			return err
		}

		s, err := getSub(name)
		if err != nil {
			return err
		}
		subscription := f.Flags | z21.Mask32(s)

		if subscription == f.Flags {
			fmt.Printf("Already subscribed to %q\n", name)
			return nil
		}

		_, err = Req(app.Conn, &z21.BroadcastFlags{Flags: subscription})
		if err != nil {
			return err
		}

		fmt.Printf("Subscribed to %q\n", name)
		return nil
	},
}

// rm NAME
var subRmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Unsubscribe from a specific event",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		name := args[0]
		if !hasSub(name) {
			return fmt.Errorf("unsupported subscription %q", name)
		}

		// TODO: set flags

		fmt.Printf("Unsubscribed from %q\n", name)
		return nil
	},
}

func printSubscriptions(m z21.Mask32) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.AppendHeader(table.Row{"Name", "Sub (Y/N)", "Description"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:     "Description",
			WidthMax: 73,
			WidthMin: 20,
		},
	})

	for _, s := range subs {
		subscribed := DISABLED
		if m.Has(s.flag) {
			subscribed = ENABLED
		}
		t.AppendRow(
			table.Row{s.name, subscribed, s.description},
		)
	}
	fmt.Printf("Bitmap: 0x%08x\n", uint32(m))
	t.Render()
}

func hasSub(name string) bool {
	for _, s := range subs {
		if s.name == name {
			return true
		}
	}
	return false
}

func getSub(name string) (uint32, error) {
	for _, s := range subs {
		if s.name == name {
			return s.flag, nil
		}
	}
	return 0, fmt.Errorf("unsupported subscription %q", name)
}

// ---------- init ----------

func init() {
	subCmd.AddCommand(
		subListCmd,
		subAddCmd,
		subRmCmd,
	)
}
