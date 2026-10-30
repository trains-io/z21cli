package cmd

import (
	"fmt"

	"github.com/trains-io/z21.go/protocol"
)

func printTurnoutInfo(info protocol.TurnoutInfo) {
	printLabelValueTable("TURNOUT", "STATE",
		labeledValue{"Address", fmt.Sprintf("%d", info.Address)},
		labeledValue{"Position", protocol.FormatTurnoutPosition(info.Position)},
	)
}

func printExtAccessoryInfo(info protocol.ExtAccessoryInfo) {
	printLabelValueTable("ACCESSORY", "STATE",
		labeledValue{"Address", fmt.Sprintf("%d", info.Address)},
		labeledValue{"Value", formatExtAccessoryValue(info)},
	)
}

func formatTurnoutInfoEvent(info protocol.TurnoutInfo) string {
	return fmt.Sprintf("[TURNOUT] %d: %s (LAN_X_TURNOUT_INFO)", info.Address, protocol.FormatTurnoutPosition(info.Position))
}

func formatExtAccessoryInfoEvent(info protocol.ExtAccessoryInfo) string {
	return fmt.Sprintf("[ACCESSORY] %d: value %s (LAN_X_EXT_ACCESSORY_INFO)", info.Address, formatExtAccessoryValue(info))
}

// formatExtAccessoryValue returns the value in decimal and hex, for example
// "18 (0x12)", or "unknown" while the Z21 does not know it (spec §5.6).
func formatExtAccessoryValue(info protocol.ExtAccessoryInfo) string {
	switch info.Status {
	case protocol.ExtAccessoryStatusValid:
		return fmt.Sprintf("%d (0x%02x)", info.Value, info.Value)
	case protocol.ExtAccessoryStatusUnknown:
		return "unknown"
	default:
		return fmt.Sprintf("%d (0x%02x), status 0x%02x", info.Value, info.Value, info.Status)
	}
}
