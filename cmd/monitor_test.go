package cmd

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/trains-io/z21.go/protocol"
	"github.com/trains-io/z21.go/server"
	"github.com/trains-io/z21.go/server/servertest"
)

const monitorFlags = protocol.BroadcastFlagXpressNet

// monitorFixture runs runMonitor against an in-process station. The client
// subscribes only through its prepended datasets, as in z21.
type monitorFixture struct {
	srv        *servertest.Server
	keepAlives chan struct{} // one per LAN_GET_SERIAL_NUMBER received
	lines      chan string   // monitor output
	timeout    <-chan time.Time

	mu   sync.Mutex
	loco protocol.LocoInfo // state of every loco, reported with its address
}

func startMonitor(t *testing.T, sessionTimeout, keepAlive time.Duration) *monitorFixture {
	t.Helper()
	return startMonitorWith(t, sessionTimeout, keepAlive, protocol.SetBroadcastFlags(monitorFlags))
}

func startMonitorWith(t *testing.T, sessionTimeout, keepAlive time.Duration, prepend ...protocol.Message) *monitorFixture {
	t.Helper()
	f := &monitorFixture{
		keepAlives: make(chan struct{}, 100),
		lines:      make(chan string),
		timeout:    time.After(5 * time.Second),
		loco:       protocol.LocoInfo{SpeedSteps: protocol.LocoInfoSpeedSteps128, Forward: true},
	}
	station := server.HandlerFunc(func(w server.ResponseWriter, r *server.Request) {
		if r.Message.Header == protocol.HeaderLANGetSerialNumber {
			w.Reply(protocol.EncodeSerialNumber(1))
			f.keepAlives <- struct{}{}
			return
		}
		if addr, err := protocol.ParseGetLocoInfo(r.Message.Data); err == nil && r.Message.Header == protocol.HeaderLANX {
			w.Reply(protocol.EncodeLocoInfo(f.locoInfo(addr)))
		}
	})
	f.srv = servertest.NewServer(station, server.WithSessionTimeout(sessionTimeout))
	t.Cleanup(f.srv.Close)

	c, err := f.srv.Dial()
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	c.SetPrependMessages(prepend...)

	out, outW := io.Pipe()
	go func() {
		defer close(f.lines)
		s := bufio.NewScanner(out)
		for s.Scan() {
			f.lines <- s.Text()
		}
	}()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- runMonitor(ctx, c, outW, keepAlive)
		outW.Close()
	}()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Errorf("runMonitor after cancel = %v, want nil", err)
		}
	})

	if got := f.nextLine(t); got != "Waiting for Z21 events ..." {
		t.Fatalf("first line = %q, want the waiting message", got)
	}
	return f
}

func (f *monitorFixture) locoInfo(addr uint16) protocol.LocoInfo {
	f.mu.Lock()
	defer f.mu.Unlock()
	info := f.loco
	info.Address = addr
	return info
}

// driveLoco changes the station's loco state and broadcasts addr's new state
// to its subscribers, returning how many there were.
func (f *monitorFixture) driveLoco(addr uint16, speed byte) int {
	f.mu.Lock()
	f.loco.Speed = speed
	f.mu.Unlock()
	return f.srv.Server.BroadcastLocoInfo(f.locoInfo(addr))
}

func (f *monitorFixture) nextLine(t *testing.T) string {
	t.Helper()
	select {
	case line := <-f.lines:
		return line
	case <-f.timeout:
		t.Fatal("monitor printed nothing")
		return ""
	}
}

func (f *monitorFixture) awaitKeepAlives(t *testing.T, n int) {
	t.Helper()
	for range n {
		select {
		case <-f.keepAlives:
		case <-f.timeout:
			t.Fatal("keep-alives did not arrive")
		}
	}
}

// broadcastTrackPowerOff checks that a broadcast reaches the monitor.
func (f *monitorFixture) broadcastTrackPowerOff(t *testing.T) {
	t.Helper()
	if n := f.srv.Server.Broadcast(monitorFlags, protocol.EncodeTrackPowerBC(false)); n != 1 {
		t.Fatalf("Broadcast reached %d clients, want 1 (subscribed)", n)
	}
	if got, want := f.nextLine(t), formatTrackPowerBCEvent(false); got != want {
		t.Fatalf("event line = %q, want %q", got, want)
	}
}

func TestRunMonitorSubscribesAtStart(t *testing.T) {
	f := startMonitor(t, server.DefaultSessionTimeout, time.Hour)

	f.awaitKeepAlives(t, 1)
	f.broadcastTrackPowerOff(t)
}

func TestRunMonitorKeepsSession(t *testing.T) {
	f := startMonitor(t, 200*time.Millisecond, 50*time.Millisecond)

	// Ten keep-alives span more than twice the session timeout.
	f.awaitKeepAlives(t, 10)
	f.broadcastTrackPowerOff(t)
}

func TestRunMonitorResubscribesWhenForgotten(t *testing.T) {
	// The session expires between keep-alives, as after a Z21 restart.
	f := startMonitor(t, 100*time.Millisecond, 300*time.Millisecond)

	f.awaitKeepAlives(t, 3)
	f.broadcastTrackPowerOff(t)
}

func TestRunMonitorLoco(t *testing.T) {
	f := startMonitorWith(t, server.DefaultSessionTimeout, 50*time.Millisecond, monitorLocoPrepend(0, []uint16{3})...)

	if got, want := f.nextLine(t), "[LOCO] 3: forward, speed 0/126, functions off (LAN_X_LOCO_INFO)"; got != want {
		t.Fatalf("first event = %q, want %q (the loco at start)", got, want)
	}
	// The keep-alives ask for the unchanged loco again; it is not printed again.
	f.awaitKeepAlives(t, 4)
	if n := f.driveLoco(3, 0x29); n != 1 {
		t.Fatalf("BroadcastLocoInfo reached %d clients, want 1 (subscribed to loco 3 with XPRESSNET_UPDATES)", n)
	}
	if got, want := f.nextLine(t), "[LOCO] 3: forward, speed 40/126, functions off (LAN_X_LOCO_INFO)"; got != want {
		t.Fatalf("event after drive = %q, want %q", got, want)
	}
	f.awaitKeepAlives(t, 2)
	f.broadcastTrackPowerOff(t)
}

func TestRunMonitorLocoResubscribesWhenForgotten(t *testing.T) {
	f := startMonitorWith(t, 100*time.Millisecond, 300*time.Millisecond, monitorLocoPrepend(0, []uint16{3})...)

	if got := f.nextLine(t); !strings.HasPrefix(got, "[LOCO] 3:") {
		t.Fatalf("first event = %q, want loco 3", got)
	}
	f.awaitKeepAlives(t, 3)
	if n := f.driveLoco(3, 0x29); n != 1 {
		t.Fatalf("BroadcastLocoInfo reached %d clients, want 1 (subscription renewed by the keep-alive)", n)
	}
	if got := f.nextLine(t); !strings.Contains(got, "speed 40/126") {
		t.Fatalf("event after drive = %q, want speed 40/126", got)
	}
}

func TestMonitorLocoPrepend(t *testing.T) {
	tests := []struct {
		name      string
		flags     uint32
		wantFlags uint32
	}{
		{name: "none", flags: 0, wantFlags: protocol.BroadcastFlagXpressNet},
		{name: "system state", flags: protocol.BroadcastFlagSystemState, wantFlags: protocol.BroadcastFlagSystemState | protocol.BroadcastFlagXpressNet},
		{name: "XpressNet", flags: protocol.BroadcastFlagXpressNet, wantFlags: protocol.BroadcastFlagXpressNet},
		{name: "all loco info", flags: protocol.BroadcastFlagAllLocoInfo, wantFlags: protocol.BroadcastFlagAllLocoInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := monitorLocoPrepend(tt.flags, []uint16{3, 1234})
			want := []protocol.Message{protocol.SetBroadcastFlags(tt.wantFlags), protocol.GetLocoInfo(3), protocol.GetLocoInfo(1234)}
			if len(got) != len(want) {
				t.Fatalf("got %d datasets, want %d", len(got), len(want))
			}
			for i := range want {
				if got[i].Header != want[i].Header || string(got[i].Data) != string(want[i].Data) {
					t.Fatalf("dataset %d = %+v, want %+v", i, got[i], want[i])
				}
			}
		})
	}
}

func TestParseMonitorLocos(t *testing.T) {
	got, err := parseMonitorLocos([]string{"3", "1234", "3"})
	if err != nil || len(got) != 2 || got[0] != 3 || got[1] != 1234 {
		t.Fatalf("parseMonitorLocos = %v, %v; want [3 1234] (duplicates dropped)", got, err)
	}

	seventeen := make([]string, 17)
	for i := range seventeen {
		seventeen[i] = strconv.Itoa(i + 1)
	}
	if _, err := parseMonitorLocos(seventeen); err == nil || !strings.Contains(err.Error(), "at most 16") {
		t.Fatalf("17 locos: error = %v, want at most 16", err)
	}
	if _, err := parseMonitorLocos([]string{"0"}); err == nil || !strings.Contains(err.Error(), "invalid loco address") {
		t.Fatalf("address 0: error = %v, want invalid loco address", err)
	}
}
