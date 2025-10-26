package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

var (
	flagAll      bool
	flagVersion  bool
	flagSerial   bool
	flagFirmware bool
	flagDevice   bool
	flagHardware bool
	flagScope    bool
)

var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Query system information from the Z21 control station",
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		sn, err := Req(app.Conn, &z21.SerialNumber{})
		if err != nil {
			return err
		}

		version, err := Req(app.Conn, &z21.Version{})
		if err != nil {
			return err
		}

		hwinfo, err := Req(app.Conn, &z21.HwInfo{})
		if err != nil {
			return err
		}

		scope, err := Req(app.Conn, &z21.Code{})
		if err != nil {
			return err
		}

		out := []string{}

		if !flagSerial && !flagVersion && !flagDevice && !flagFirmware && !flagHardware && !flagAll && !flagScope {
			flagAll = true
		}

		if flagAll || flagDevice {
			out = append(out, fmt.Sprintf("%s", version.CommandStationID))
		}

		if flagAll || flagHardware {
			out = append(out, fmt.Sprintf("%s", hwinfo.Hardware))
		}

		if flagAll || flagSerial {
			out = append(out, fmt.Sprintf("%d", sn.SerialNumber))
		}

		if flagAll || flagVersion {
			out = append(out, version.XBusProtoVersion)
		}

		if flagAll || flagFirmware {
			out = append(out, hwinfo.FirmwareVersion)
		}

		if flagAll || flagScope {
			var s string = ""
			switch scope.Code {
			case z21.Z21_NO_LOCK:
				s = "no lock"
			case z21.Z21_START_LOCKED:
				s = "locked"
			case z21.Z21_START_UNLOCKED:
				s = "unlocked"
			default:
				s = "unknown"
			}
			out = append(out, fmt.Sprintf("[%s]", s))
		}

		fmt.Printf("%s\n", strings.Join(out, " "))

		return nil
	},
}

func init() {
	infoCmd.Flags().BoolVarP(
		&flagAll,
		"all", "a",
		false,
		"print all information, in the following order:\n"+
			"device family, hardware platform, serial number, "+
			"X-Bus protocol version, firmware version",
	)
	infoCmd.Flags().BoolVarP(
		&flagDevice,
		"device-family", "d",
		false,
		"print the z21 device family",
	)
	infoCmd.Flags().BoolVarP(
		&flagHardware,
		"hardware-platform", "i",
		false,
		"print the z21 hardware platform",
	)
	infoCmd.Flags().BoolVarP(
		&flagVersion,
		"x-bus-version", "x",
		false,
		"print the X-Bus protocol version",
	)
	infoCmd.Flags().BoolVarP(
		&flagSerial,
		"serial", "S",
		false,
		"print the serial number",
	)
	infoCmd.Flags().BoolVarP(
		&flagFirmware,
		"firmware-version", "f",
		false,
		"print the firmware version",
	)
	infoCmd.Flags().BoolVarP(
		&flagScope,
		"scope", "s",
		false,
		"print the software feature scope",
	)
}
