package cmd

import (
	"fmt"
	"strings"

	"github.com/trains-io/z21.go/protocol"
)

func printLocoInfo(info protocol.LocoInfo) {
	format := "DCC"
	if info.MMFormat {
		format = "MM"
	}
	busy := "no"
	if info.Busy {
		busy = "yes (another throttle)"
	}
	printLabelValueTable("LOCO", "STATE",
		labeledValue{"Address", fmt.Sprintf("%d", info.Address)},
		labeledValue{"Format", format},
		labeledValue{"Speed steps", formatLocoSpeedSteps(info)},
		labeledValue{"Direction", formatLocoDirection(info)},
		labeledValue{"Speed", formatLocoSpeed(info)},
		labeledValue{"Functions", formatLocoFunctions(info)},
		labeledValue{"Busy", busy},
	)
}

func formatLocoInfoEvent(info protocol.LocoInfo) string {
	speed := formatLocoSpeed(info)
	if _, eStop := info.SpeedStep(); !eStop {
		speed = "speed " + speed
	}
	line := fmt.Sprintf("[LOCO] %d: %s, %s, functions %s",
		info.Address, formatLocoDirection(info), speed, formatLocoFunctions(info))
	if info.Busy {
		line += ", busy"
	}
	return line + " (LAN_X_LOCO_INFO)"
}

func formatLocoSpeedSteps(info protocol.LocoInfo) string {
	switch info.SpeedSteps {
	case protocol.LocoInfoSpeedSteps14:
		return "14"
	case protocol.LocoInfoSpeedSteps28:
		return "28"
	case protocol.LocoInfoSpeedSteps128:
		return "128"
	default:
		return fmt.Sprintf("unknown (%d)", info.SpeedSteps)
	}
}

func formatLocoDirection(info protocol.LocoInfo) string {
	if info.Forward {
		return "forward"
	}
	return "reverse"
}

// formatLocoSpeed returns the speed step out of the highest one, for example
// "40/126", or "emergency stop".
func formatLocoSpeed(info protocol.LocoInfo) string {
	step, eStop := info.SpeedStep()
	if eStop {
		return "emergency stop"
	}
	maxStep := protocol.MaxLocoSpeedStep(protocol.LocoDriveSpeedSteps(info.SpeedSteps))
	if maxStep == 0 {
		return fmt.Sprintf("%d", step)
	}
	return fmt.Sprintf("%d/%d", step, maxStep)
}

// formatLocoFunctions lists the functions that are on, for example "F0 F3",
// or "off".
func formatLocoFunctions(info protocol.LocoInfo) string {
	mask := uint32(info.FunctionsF1F4&0x0F)<<1 |
		uint32(info.FunctionsF5F12)<<5 |
		uint32(info.FunctionsF13F20)<<13 |
		uint32(info.FunctionsF21F28)<<21 |
		uint32(info.FunctionsF29F31&0x07)<<29
	if info.Headlight {
		mask |= 1
	}
	var on []string
	for fn := range 32 {
		if mask&(1<<fn) != 0 {
			on = append(on, fmt.Sprintf("F%d", fn))
		}
	}
	if len(on) == 0 {
		return "off"
	}
	return strings.Join(on, " ")
}
