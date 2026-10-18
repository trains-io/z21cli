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
	if protocol.IsCANDetectorReport(msg) {
		return formatCANDetectorEvent(msg)
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
	if info, err := protocol.ParseLocoInfo(msg.Data); err == nil {
		return formatLocoInfoEvent(info), true
	}
	if status, err := protocol.ParseXStatus(msg.Data); err == nil {
		return formatXStatusChangedEvent(status.CentralState), true
	}
	return "", false
}

func formatCANDetectorEvent(msg protocol.Message) (string, bool) {
	report, err := protocol.ParseCANDetector(msg.Data)
	if err != nil {
		return "", false
	}

	switch report.Type {
	case protocol.CANDetectorTypeOccupancy:
		return fmt.Sprintf("[CAN] Detector 0x%04X module %d port %d: %s (LAN_CAN_DETECTOR)",
			report.NetID,
			report.Addr,
			report.Port+1,
			protocol.OccupancyStatusLabel(report.Value1),
		), true
	default:
		if protocol.IsCANDetectorLocoAddressType(report.Type) {
			locos := report.LocoAddresses()
			return fmt.Sprintf("[CAN] Detector 0x%04X module %d port %d loco %s / %s (LAN_CAN_DETECTOR)",
				report.NetID,
				report.Addr,
				report.Port+1,
				formatCANDetectorLoco(locos[0]),
				formatCANDetectorLoco(locos[1]),
			), true
		}
		return fmt.Sprintf("[CAN] Detector 0x%04X module %d port %d type 0x%02x (LAN_CAN_DETECTOR)",
			report.NetID,
			report.Addr,
			report.Port+1,
			report.Type,
		), true
	}
}

func formatCANDetectorLoco(loco protocol.CANDetectorLocoAddress) string {
	if loco.Address == 0 {
		return "-"
	}
	switch loco.Direction {
	case protocol.CANDetectorLocoDirectionForward:
		return fmt.Sprintf("%d fwd", loco.Address)
	case protocol.CANDetectorLocoDirectionBackward:
		return fmt.Sprintf("%d rev", loco.Address)
	default:
		return fmt.Sprintf("%d", loco.Address)
	}
}
