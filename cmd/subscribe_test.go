package cmd

import (
	"testing"

	"github.com/trains-io/z21.go/protocol"
)

func TestLookupSubscription(t *testing.T) {
	tests := []struct {
		name string
		want uint32
	}{
		{"SYSTEM_UPDATES", protocol.BroadcastFlagSystemState},
		{"system_updates", protocol.BroadcastFlagSystemState},
		{"CAN_DETECTOR_UPDATES", protocol.BroadcastFlagCANDetector},
		{"XPRESSNET_UPDATES", protocol.BroadcastFlagXpressNet},
		{"LOCONET_OCC_UPDATES", protocol.BroadcastFlagLocoNetOcc},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag, err := lookupSubscription(tt.name)
			if err != nil {
				t.Fatal(err)
			}
			if flag != tt.want {
				t.Fatalf("flag = %#x, want %#x", flag, tt.want)
			}
		})
	}

	if _, err := lookupSubscription("UNKNOWN"); err == nil {
		t.Fatal("expected error for unknown subscription")
	}
}

func TestKnownSubscriptionsCoverAllBroadcastFlags(t *testing.T) {
	if len(knownSubscriptions) != len(protocol.KnownBroadcastFlags) {
		t.Fatalf("subscriptions = %d, want %d", len(knownSubscriptions), len(protocol.KnownBroadcastFlags))
	}
	for _, flag := range protocol.KnownBroadcastFlags {
		if _, ok := subscriptionNamesByFlag[flag.Value]; !ok {
			t.Fatalf("missing subscription name for %#x (%s)", flag.Value, flag.Name)
		}
	}
}

func TestActiveSubscriptions(t *testing.T) {
	mask := protocol.BroadcastFlagSystemState | protocol.BroadcastFlagCANDetector
	active := activeSubscriptions(mask)
	if len(active) != 2 {
		t.Fatalf("active = %v, want 2 entries", active)
	}
}

func TestPrintSubscriptionList(t *testing.T) {
	got := captureStdout(t, func() {
		printSubscriptionList(protocol.BroadcastFlagSystemState | protocol.BroadcastFlagXpressNet)
	})

	want := "" +
		"Broadcast mask: 0x00000101\n" +
		"\n" +
		"Active subscriptions:\n" +
		"\n" +
		"XPRESSNET_UPDATES: Track power, programming mode, short circuit, stop, locomotive and turnout messages.\n" +
		"SYSTEM_UPDATES:    Z21 system status such as track voltage.\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPrintSubscriptionListEmpty(t *testing.T) {
	got := captureStdout(t, func() {
		printSubscriptionList(0)
	})

	want := "Broadcast mask: 0x00000000\n\nNo active subscriptions.\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestFormatSystemStateEvent(t *testing.T) {
	got := formatSystemStateEvent(protocol.SystemState{
		MainCurrent:   30,
		ProgCurrent:   0,
		Temperature:   27,
		SupplyVoltage: 19900,
		VCCVoltage:    18000,
	})
	want := "[SYS] Main: 30mA  Prog: 0mA   Temp: 27°C  Volt: 19.9V (18.0V)"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNeedsAppContextSubscribeMonitor(t *testing.T) {
	subCmd := newSubscribeCmd()
	addCmd, _, err := subCmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	if !needsAppContext(addCmd, nil) {
		t.Fatal("expected subscribe add to need app context")
	}

	monitorCmd := newMonitorCmd()
	if !needsAppContext(monitorCmd, nil) {
		t.Fatal("expected monitor to need app context")
	}
}

func TestShouldApplyBroadcastFlagsSubscribe(t *testing.T) {
	subCmd := newSubscribeCmd()

	addCmd, _, err := subCmd.Find([]string{"add"})
	if err != nil {
		t.Fatal(err)
	}
	if shouldApplyBroadcastFlags(addCmd) {
		t.Fatal("expected subscribe add to skip applyBroadcastFlags")
	}

	lsCmd, _, err := subCmd.Find([]string{"ls"})
	if err != nil {
		t.Fatal(err)
	}
	if !shouldApplyBroadcastFlags(lsCmd) {
		t.Fatal("expected subscribe ls to apply saved broadcast flags")
	}
}
