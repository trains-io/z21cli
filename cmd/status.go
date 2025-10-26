package cmd

import (
	"fmt"
	"math"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

const (
	STATUS_ON       string = "ON"
	STATUS_OFF      string = "OFF"
	STATUS_ACTIVE   string = "ACTIVE"
	STATUS_INACTIVE string = "INACTIVE"
)

var status = []struct {
	flag        uint8
	description string
	states      []string
}{
	{
		flag:        z21.EMERGENCY_STOP,
		description: `Emergency Stop`,
		states:      []string{STATUS_OFF, STATUS_ON},
	},
	{
		flag:        z21.TRACK_VOLTAGE_OFF,
		description: "Track Voltage",
		states:      []string{STATUS_ON, STATUS_OFF},
	},
	{
		flag:        z21.SHORT_CIRCUIT,
		description: "Short Circuit",
		states:      []string{STATUS_OFF, STATUS_ON},
	},
	{
		flag:        z21.PROGRAMMING_MODE_ACTIVE,
		description: "Programming Mode",
		states:      []string{STATUS_INACTIVE, STATUS_ACTIVE},
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current Z21 status",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		status, err := getTrackStatus(app.Conn)
		if err != nil {
			return err
		}

		data, err := getSystemStatus(app.Conn)
		if err != nil {
			return err
		}

		printTrackStatus(status.Mask)
		fmt.Println()
		printSystemStatus(data)

		return nil
	},
}

// ---------- subcommands ----------

// track
var statusTrackCmd = &cobra.Command{
	Use:   "track",
	Short: "Show track status",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		status, err := getTrackStatus(app.Conn)
		if err != nil {
			return err
		}

		printTrackStatus(status.Mask)
		return nil
	},
}

// system
var statusSystemCmd = &cobra.Command{
	Use:   "system",
	Short: "Show system status",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		data, err := getSystemStatus(app.Conn)
		if err != nil {
			return err
		}

		printSystemStatus(data)
		return nil
	},
}

func getTrackStatus(conn *z21.Conn) (*z21.Status, error) {
	status, err := Req(conn, &z21.Status{})
	if err != nil {
		return nil, err
	}
	return status, nil
}

func printTrackStatus(m z21.Mask8) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.AppendHeader(
		table.Row{
			"Track", fmt.Sprintf("Status (0x%02x)", uint8(m)),
		},
	)
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:     "Status",
			WidthMax: 73,
			WidthMin: 20,
		},
	})

	for _, s := range status {
		onoff := s.states[0]
		if m.Has(s.flag) {
			onoff = s.states[1]
		}
		t.AppendRow(
			table.Row{s.description, onoff},
		)
	}
	t.Render()
}

func getSystemStatus(conn *z21.Conn) (*z21.SysData, error) {
	data, err := Req(conn, &z21.SysData{})
	if err != nil {
		return nil, err
	}
	return data, nil
}

func printSystemStatus(d *z21.SysData) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.Style().Options.DrawBorder = false
	t.Style().Options.SeparateColumns = false
	t.AppendHeader(table.Row{"Main", "Prog", "Temp", "Supply", "Internal"})
	t.AppendRow(
		table.Row{
			fmt.Sprintf("%dmA", d.MainCurrent),
			fmt.Sprintf("%dmA", d.ProgCurrent),
			fmt.Sprintf("%d°C", d.Temperature),
			fmt.Sprintf("%sV", mVToVoltString(d.SupplyVoltage)),
			fmt.Sprintf("%sV", mVToVoltString(d.VccVoltage)),
		},
	)
	t.Render()
}

func mVToVoltString(mV uint16) string {
	volts := float64(mV) / 1000.0
	voltsRounded := math.Round(volts*10) / 10
	return fmt.Sprintf("%.1f", voltsRounded)
}

// ---------- init ----------

func init() {
	statusCmd.AddCommand(
		statusTrackCmd,
		statusSystemCmd,
	)
}
