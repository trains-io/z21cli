package cmd

import (
	"testing"

	"github.com/trains-io/z21cli/internal/contextstore"
	"github.com/trains-io/z21.go/protocol"
)

func TestParseBroadcastFlags(t *testing.T) {
	tests := []struct {
		in   string
		want uint32
	}{
		{"0x101", 0x101},
		{"257", 257},
		{"0x00000101", protocol.DefaultBroadcastFlags},
	}

	for _, tt := range tests {
		got, err := parseBroadcastFlags(tt.in)
		if err != nil {
			t.Fatalf("parseBroadcastFlags(%q) error = %v", tt.in, err)
		}
		if got != tt.want {
			t.Fatalf("parseBroadcastFlags(%q) = %#x, want %#x", tt.in, got, tt.want)
		}
	}
}

func TestEffectiveBroadcastFlags(t *testing.T) {
	flags := uint32(0x103)
	session := &contextstore.Session{BroadcastFlags: &flags}

	got := effectiveBroadcastFlags(session)
	if got != 0x103 {
		t.Fatalf("effectiveBroadcastFlags() = %#x, want 0x103", got)
	}

	got = effectiveBroadcastFlags(nil)
	if got != protocol.DefaultBroadcastFlags {
		t.Fatalf("effectiveBroadcastFlags(nil) = %#x, want default", got)
	}
}

func TestNeedsAppContextBroadcast(t *testing.T) {
	tests := []struct {
		name  string
		sub   []string
		flags []string
		want  bool
	}{
		{"broadcast get", []string{"get"}, nil, true},
		{"broadcast get direct host flag", []string{"get"}, []string{"--host", "10.0.0.1"}, false},
		{"broadcast set context", []string{"set"}, nil, true},
		{"broadcast set direct host flag", []string{"set"}, []string{"--host", "10.0.0.1"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broadcastCmd := newBroadcastCmd()
			cmd, _, err := broadcastCmd.Find(tt.sub)
			if err != nil {
				t.Fatal(err)
			}
			if tt.flags != nil {
				if err := cmd.ParseFlags(tt.flags); err != nil {
					t.Fatalf("ParseFlags() error = %v", err)
				}
			}
			if got := needsAppContext(cmd, nil); got != tt.want {
				t.Fatalf("needsAppContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldApplyBroadcastFlags(t *testing.T) {
	broadcastCmd := newBroadcastCmd()

	setCmd, _, err := broadcastCmd.Find([]string{"set"})
	if err != nil {
		t.Fatal(err)
	}
	if shouldApplyBroadcastFlags(setCmd) {
		t.Fatal("expected broadcast set to skip applyBroadcastFlags")
	}

	getCmd, _, err := broadcastCmd.Find([]string{"get"})
	if err != nil {
		t.Fatal(err)
	}
	if shouldApplyBroadcastFlags(getCmd) {
		t.Fatal("expected broadcast get to skip applyBroadcastFlags")
	}

	if !shouldApplyBroadcastFlags(newPingCmd()) {
		t.Fatal("expected ping to apply saved broadcast flags on connect")
	}

	powerCmd := newPowerCmd()
	onCmd, _, err := powerCmd.Find([]string{"on"})
	if err != nil {
		t.Fatal(err)
	}
	if shouldApplyBroadcastFlags(onCmd) {
		t.Fatal("expected power on to bundle broadcast flags with the command")
	}

	if shouldApplyBroadcastFlags(newStopCmd()) {
		t.Fatal("expected stop to bundle broadcast flags with the command")
	}
}

func TestBroadcastDirectHost(t *testing.T) {
	broadcastCmd := newBroadcastCmd()
	setCmd, _, err := broadcastCmd.Find([]string{"set"})
	if err != nil {
		t.Fatal(err)
	}

	if host, err := directHost(setCmd); err != nil || host != "" {
		t.Fatalf("empty host = %q, err = %v", host, err)
	}

	if err := broadcastCmd.PersistentFlags().Set("host", "10.0.0.5"); err != nil {
		t.Fatal(err)
	}
	if host, err := directHost(setCmd); err != nil || host != "10.0.0.5" {
		t.Fatalf("flag host = %q, err = %v", host, err)
	}
}
