package cmd

import "testing"

func TestFormatTrackStatusFields(t *testing.T) {
	state := byte(0x00)
	if got := formatEmergencyStop(state); got != "OFF" {
		t.Fatalf("emergency stop = %q", got)
	}
	if got := formatTrackVoltage(state); got != "ON" {
		t.Fatalf("track voltage = %q", got)
	}
	if got := formatShortCircuit(state); got != "OFF" {
		t.Fatalf("short circuit = %q", got)
	}
	if got := formatProgrammingMode(state); got != "INACTIVE" {
		t.Fatalf("programming mode = %q", got)
	}

	state = xStatusEmergencyStop | xStatusTrackVoltageOff | xStatusShortCircuit | xStatusProgrammingMode
	if got := formatEmergencyStop(state); got != "ON" {
		t.Fatalf("emergency stop = %q", got)
	}
	if got := formatTrackVoltage(state); got != "OFF" {
		t.Fatalf("track voltage = %q", got)
	}
	if got := formatShortCircuit(state); got != "ON" {
		t.Fatalf("short circuit = %q", got)
	}
	if got := formatProgrammingMode(state); got != "ACTIVE" {
		t.Fatalf("programming mode = %q", got)
	}
}

func TestPrintZ21TrackStatus(t *testing.T) {
	got := captureStdout(t, func() {
		printZ21TrackStatus(0x00)
	})

	want := "" +
		" TRACK             STATUS (0X00) \n" +
		"---------------------------------\n" +
		" Emergency Stop    OFF\n" +
		" Track Voltage     ON\n" +
		" Short Circuit     OFF\n" +
		" Programming Mode  INACTIVE\n"
	if got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}
