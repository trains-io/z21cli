package cmd

import (
	"testing"

	"github.com/trains-io/z21.go/protocol"
)

func TestParseNetID(t *testing.T) {
	tests := []struct {
		in   string
		want uint16
	}{
		{"0xdb04", 0xDB04},
		{"DB04", 0xDB04},
		{"db04", 0xDB04},
	}

	for _, tt := range tests {
		got, err := parseNetID(tt.in)
		if err != nil {
			t.Fatalf("parseNetID(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("parseNetID(%q) = %#x, want %#x", tt.in, got, tt.want)
		}
	}
}

func TestSummarizeCANDevices(t *testing.T) {
	msgs := []protocol.Message{
		{
			Header: protocol.HeaderLANCANDetector,
			Data:   []byte{0x04, 0xdb, 0x1f, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00},
		},
		{
			Header: protocol.HeaderLANCANDetector,
			Data:   []byte{0x04, 0xdb, 0x1f, 0x00, 0x07, 0x01, 0x00, 0x01, 0x00, 0x00},
		},
	}

	devices, err := summarizeCANDevices(msgs)
	if err != nil {
		t.Fatal(err)
	}
	if len(devices) != 1 {
		t.Fatalf("len(devices) = %d, want 1", len(devices))
	}
	if devices[0].NetID != 0xDB04 || devices[0].Addr != 31 {
		t.Fatalf("device = %+v", devices[0])
	}
	if got := formatCANPorts(devices[0].Ports); got != "1,8" {
		t.Fatalf("ports = %q, want 1,8", got)
	}
}

func TestCanNeedsAppContext(t *testing.T) {
	canCmd := newCanCmd()
	discoverCmd, _, err := canCmd.Find([]string{"discover"})
	if err != nil {
		t.Fatal(err)
	}
	if !needsAppContext(discoverCmd, nil) {
		t.Fatal("expected can discover to need app context")
	}

	setCmd, _, err := canCmd.Find([]string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	if err := canCmd.PersistentFlags().Set("host", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if needsAppContext(setCmd, nil) {
		t.Fatal("expected can set with --host to skip app context")
	}
}

func TestCanParentRequiresSubcommand(t *testing.T) {
	canCmd := newCanCmd()
	err := canCmd.RunE(canCmd, nil)
	if err == nil {
		t.Fatal("expected error without subcommand")
	}
	if !printSubcommandRequired(err) {
		t.Fatalf("expected subcommand required error, got %v", err)
	}
}

func TestFormatCANPortsRange(t *testing.T) {
	ports := map[uint8]struct{}{}
	for i := uint8(0); i < 8; i++ {
		ports[i] = struct{}{}
	}
	if got := formatCANPorts(ports); got != "1-8" {
		t.Fatalf("formatCANPorts() = %q, want 1-8", got)
	}
}
