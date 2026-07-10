package cmd

import (
	"testing"

	"github.com/trains-io/z21.go/protocol"
)

func TestFormatMonitorEvent(t *testing.T) {
	tests := []struct {
		name string
		msg  protocol.Message
		want string
	}{
		{
			name: "system state",
			msg: protocol.Message{
				Header: protocol.HeaderLANSystemStateDataChanged,
				Data:   []byte{0x64, 0x00, 0, 0, 0, 0, 0x2A, 0x00, 0x10, 0x27, 0xE8, 0x03, 0, 0, 0, 0x31},
			},
			want: "[SYS] Main: 100mA  Prog: 0mA   Temp: 42°C  Volt: 10.0V (1.0V)",
		},
		{
			name: "stopped",
			msg:  protocol.Message{Header: protocol.HeaderLANX, Data: []byte{0x81, 0x00, 0x81}},
			want: "[STOP] Emergency stop (LAN_X_BC_STOPPED)",
		},
		{
			name: "track power on",
			msg:  protocol.Message{Header: protocol.HeaderLANX, Data: []byte{0x61, 0x01, 0x60}},
			want: "[PWR] Track voltage on (LAN_X_BC_TRACK_POWER_ON)",
		},
		{
			name: "track power off",
			msg:  protocol.Message{Header: protocol.HeaderLANX, Data: []byte{0x61, 0x00, 0x61}},
			want: "[PWR] Track voltage off (LAN_X_BC_TRACK_POWER_OFF)",
		},
		{
			name: "programming mode",
			msg:  protocol.Message{Header: protocol.HeaderLANX, Data: []byte{0x61, 0x02, 0x63}},
			want: "[PROG] Programming mode active (LAN_X_BC_PROGRAMMING_MODE)",
		},
		{
			name: "short circuit",
			msg:  protocol.Message{Header: protocol.HeaderLANX, Data: []byte{0x61, 0x08, 0x69}},
			want: "[SHORT] Track short circuit (LAN_X_BC_TRACK_SHORT_CIRCUIT)",
		},
		{
			name: "unknown command",
			msg:  protocol.Message{Header: protocol.HeaderLANX, Data: []byte{0x61, 0x82, 0xE3}},
			want: "[X] Unknown command (LAN_X_UNKNOWN_COMMAND)",
		},
		{
			name: "status changed",
			msg:  protocol.Message{Header: protocol.HeaderLANX, Data: []byte{0x62, 0x22, 0x04, 0x40}},
			want: "[STAT] short circuit (LAN_X_STATUS_CHANGED)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := formatMonitorEvent(tt.msg)
			if !ok {
				t.Fatal("formatMonitorEvent() ok = false, want true")
			}
			if got != tt.want {
				t.Fatalf("formatMonitorEvent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMonitorEventIgnoresUnknown(t *testing.T) {
	if _, ok := formatMonitorEvent(protocol.Message{Header: protocol.HeaderLANGetHWInfo}); ok {
		t.Fatal("expected unknown message to be ignored")
	}
}
