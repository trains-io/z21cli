package cmd

import (
	"fmt"

	"github.com/trains-io/z21.go/protocol"
)

func formatSystemStateEvent(state protocol.SystemState) string {
	return fmt.Sprintf("[SYS] Main: %dmA  Prog: %dmA   Temp: %d°C  Volt: %.1fV (%.1fV)",
		state.MainCurrent,
		state.ProgCurrent,
		state.Temperature,
		millivoltsToVolts(state.SupplyVoltage),
		millivoltsToVolts(state.VCCVoltage),
	)
}

func formatBCStoppedEvent() string {
	return "[STOP] Emergency stop (LAN_X_BC_STOPPED)"
}

func formatTrackPowerBCEvent(on bool) string {
	if on {
		return "[PWR] Track voltage on (LAN_X_BC_TRACK_POWER_ON)"
	}
	return "[PWR] Track voltage off (LAN_X_BC_TRACK_POWER_OFF)"
}

func formatBCProgrammingModeEvent() string {
	return "[PROG] Programming mode active (LAN_X_BC_PROGRAMMING_MODE)"
}

func formatBCShortCircuitEvent() string {
	return "[SHORT] Track short circuit (LAN_X_BC_TRACK_SHORT_CIRCUIT)"
}

func formatUnknownCommandEvent() string {
	return "[X] Unknown command (LAN_X_UNKNOWN_COMMAND)"
}

func formatXStatusChangedEvent(state byte) string {
	return fmt.Sprintf("[STAT] %s (LAN_X_STATUS_CHANGED)", protocol.FormatXStatusFlags(state))
}

func formatMonitorEvent(msg protocol.Message) (string, bool) {
	if protocol.IsSystemStateDataChanged(msg) {
		state, err := protocol.ParseSystemState(msg.Data)
		if err != nil {
			return "", false
		}
		return formatSystemStateEvent(state), true
	}
	if msg.Header != protocol.HeaderLANX {
		return "", false
	}
	if protocol.IsBCStopped(msg.Data) {
		return formatBCStoppedEvent(), true
	}
	if on, err := protocol.ParseTrackPowerBC(msg.Data); err == nil {
		return formatTrackPowerBCEvent(on), true
	}
	if protocol.IsBCProgrammingMode(msg.Data) {
		return formatBCProgrammingModeEvent(), true
	}
	if protocol.IsBCShortCircuit(msg.Data) {
		return formatBCShortCircuitEvent(), true
	}
	if protocol.IsUnknownCommand(msg.Data) {
		return formatUnknownCommandEvent(), true
	}
	if status, err := protocol.ParseXStatus(msg.Data); err == nil {
		return formatXStatusChangedEvent(status.CentralState), true
	}
	return "", false
}
