package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

type deviceInfo struct {
	Serial    string
	HW        protocol.HWInfo
	XVersion  protocol.XVersion
	XFirmware protocol.XFirmware
	LockCode  byte
}

type infoAvailability struct {
	serial    bool
	hw        bool
	version   bool
	firmware  bool
	code      bool
}

func (a infoAvailability) any() bool {
	return a.serial || a.hw || a.version || a.firmware || a.code
}

func collectDeviceInfo(ctx context.Context, c *client.Client, timeout time.Duration) (deviceInfo, infoAvailability, []error) {
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	msgs, err := c.Call(queryCtx,
		protocol.GetSerialNumber(),
		protocol.GetHWInfo(),
		protocol.GetXVersion(),
		protocol.GetXFirmware(),
		protocol.GetCode(),
	)
	if err != nil {
		return deviceInfo{}, infoAvailability{}, []error{err}
	}

	return parseDeviceInfoFromMessages(msgs)
}

func parseDeviceInfoFromMessages(msgs []protocol.Message) (deviceInfo, infoAvailability, []error) {
	var (
		info     deviceInfo
		avail    infoAvailability
		failures []error
	)

	if serial, ok := protocol.SerialFromMessages(msgs); ok {
		info.Serial = serial
		avail.serial = true
	} else {
		failures = append(failures, fmt.Errorf("serial: no LAN_GET_SERIAL_NUMBER reply in response"))
	}

	if hw, err := protocol.HWInfoFromMessages(msgs); err != nil {
		failures = append(failures, fmt.Errorf("hw: %w", err))
	} else {
		info.HW = hw
		avail.hw = true
	}

	if version, err := protocol.XVersionFromMessages(msgs); err != nil {
		failures = append(failures, fmt.Errorf("version: %w", err))
	} else {
		info.XVersion = version
		avail.version = true
	}

	if firmware, err := protocol.XFirmwareFromMessages(msgs); err != nil {
		failures = append(failures, fmt.Errorf("firmware: %w", err))
	} else {
		info.XFirmware = firmware
		avail.firmware = true
	}

	if code, err := protocol.CodeFromMessages(msgs); err != nil {
		failures = append(failures, fmt.Errorf("code: %w", err))
	} else {
		info.LockCode = code
		avail.code = true
	}

	return info, avail, failures
}

func printAllDeviceInfo(info deviceInfo, avail infoAvailability) {
	var rows []labeledValue
	if avail.serial {
		rows = append(rows, labeledValue{"Serial number", info.Serial})
	}
	if avail.hw {
		rows = append(rows,
			labeledValue{"Hardware type", fmt.Sprintf("%s (0x%08x)", protocol.FormatHwType(info.HW.HwType), info.HW.HwType)},
			labeledValue{"Firmware (HW)", protocol.FormatFirmwareVersion(info.HW.FirmwareVersion)},
		)
	}
	if avail.firmware {
		rows = append(rows, labeledValue{"Firmware (X-Bus)", protocol.FormatXFirmwareVersion(info.XFirmware)})
	}
	if avail.version {
		rows = append(rows,
			labeledValue{"X-Bus version", protocol.FormatXBusVersion(info.XVersion.XBusVersion)},
			labeledValue{"Command station ID", protocol.FormatCommandStationID(info.XVersion.CommandStationID)},
		)
	}
	if avail.code {
		rows = append(rows,
			labeledValue{"Lock code", fmt.Sprintf("0x%02x", info.LockCode)},
			labeledValue{"Status", protocol.FormatLockCode(info.LockCode)},
		)
	}
	printLabeledValues(rows...)
}

func runInfoAll(
	cmd *cobra.Command,
	host string,
	port int,
	timeout time.Duration,
) error {
	return runZ21Session(cmd, host, port, func(ctx context.Context, c *client.Client) error {
		info, avail, failures := collectDeviceInfo(ctx, c, timeout)
		if avail.any() {
			printAllDeviceInfo(info, avail)
		}
		for _, err := range failures {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", err)
		}
		if !avail.any() {
			return fmt.Errorf("info %s: all queries failed", c.Addr())
		}
		return nil
	})
}

func fetchSerialOnly(ctx context.Context, c *client.Client) (deviceInfo, error) {
	var info deviceInfo
	msgs, err := c.Call(ctx, protocol.GetSerialNumber())
	if err != nil {
		return info, err
	}
	serial, ok := protocol.SerialFromMessages(msgs)
	if !ok {
		return info, fmt.Errorf("no LAN_GET_SERIAL_NUMBER reply in response")
	}
	info.Serial = serial
	return info, nil
}

func fetchHWOnly(ctx context.Context, c *client.Client) (deviceInfo, error) {
	var info deviceInfo
	msgs, err := c.Call(ctx, protocol.GetHWInfo())
	if err != nil {
		return info, err
	}
	info.HW, err = protocol.HWInfoFromMessages(msgs)
	return info, err
}

func fetchVersionOnly(ctx context.Context, c *client.Client) (deviceInfo, error) {
	var info deviceInfo
	msgs, err := c.Call(ctx, protocol.GetXVersion())
	if err != nil {
		return info, err
	}
	info.XVersion, err = protocol.XVersionFromMessages(msgs)
	return info, err
}

func fetchFirmwareOnly(ctx context.Context, c *client.Client) (deviceInfo, error) {
	var info deviceInfo
	msgs, err := c.Call(ctx, protocol.GetXFirmware())
	if err != nil {
		return info, err
	}
	info.XFirmware, err = protocol.XFirmwareFromMessages(msgs)
	return info, err
}

func fetchCodeOnly(ctx context.Context, c *client.Client) (deviceInfo, error) {
	var info deviceInfo
	msgs, err := c.Call(ctx, protocol.GetCode())
	if err != nil {
		return info, err
	}
	info.LockCode, err = protocol.CodeFromMessages(msgs)
	return info, err
}

func infoDial(cmd *cobra.Command) (host string, port int, timeout time.Duration, err error) {
	host, err = directHost(cmd)
	if err != nil {
		return "", 0, 0, err
	}
	port, err = dialPort(cmd)
	if err != nil {
		return "", 0, 0, err
	}
	timeout, err = dialTimeout(cmd)
	if err != nil {
		return "", 0, 0, err
	}
	return host, port, timeout, nil
}
