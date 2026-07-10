package cmd

import "fmt"

const (
	xStatusEmergencyStop   byte = 0x01
	xStatusTrackVoltageOff byte = 0x02
	xStatusShortCircuit    byte = 0x04
	xStatusProgrammingMode byte = 0x20
)

func printZ21TrackStatus(centralState byte) {
	fmt.Printf(" TRACK             STATUS (0X%02X) \n", centralState)
	fmt.Println("---------------------------------")
	fmt.Printf(" Emergency Stop    %s\n", formatEmergencyStop(centralState))
	fmt.Printf(" Track Voltage     %s\n", formatTrackVoltage(centralState))
	fmt.Printf(" Short Circuit     %s\n", formatShortCircuit(centralState))
	fmt.Printf(" Programming Mode  %s\n", formatProgrammingMode(centralState))
}

func formatEmergencyStop(state byte) string {
	if state&xStatusEmergencyStop != 0 {
		return "ON"
	}
	return "OFF"
}

func formatTrackVoltage(state byte) string {
	if state&xStatusTrackVoltageOff != 0 {
		return "OFF"
	}
	return "ON"
}

func formatShortCircuit(state byte) string {
	if state&xStatusShortCircuit != 0 {
		return "ON"
	}
	return "OFF"
}

func formatProgrammingMode(state byte) string {
	if state&xStatusProgrammingMode != 0 {
		return "ACTIVE"
	}
	return "INACTIVE"
}

func millivoltsToVolts(mv uint16) float64 {
	return float64(mv) / 1000.0
}
