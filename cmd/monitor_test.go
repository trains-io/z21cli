package cmd

import (
	"bufio"
	"context"
	"io"
	"testing"
	"time"

	"github.com/trains-io/z21.go/protocol"
	"github.com/trains-io/z21.go/server"
	"github.com/trains-io/z21.go/server/servertest"
)

const monitorFlags = protocol.BroadcastFlagXpressNet

// monitorFixture runs runMonitor against an in-process station. The client
// subscribes only through its prepended broadcast flags, as in z21.
type monitorFixture struct {
	srv        *servertest.Server
	keepAlives chan struct{} // one per LAN_GET_SERIAL_NUMBER received
	lines      chan string   // monitor output
	timeout    <-chan time.Time
}

func startMonitor(t *testing.T, sessionTimeout, keepAlive time.Duration) *monitorFixture {
	t.Helper()
	f := &monitorFixture{
		keepAlives: make(chan struct{}, 100),
		lines:      make(chan string),
		timeout:    time.After(5 * time.Second),
	}
	station := server.HandlerFunc(func(w server.ResponseWriter, r *server.Request) {
		if r.Message.Header == protocol.HeaderLANGetSerialNumber {
			w.Reply(protocol.EncodeSerialNumber(1))
			f.keepAlives <- struct{}{}
		}
	})
	f.srv = servertest.NewServer(station, server.WithSessionTimeout(sessionTimeout))
	t.Cleanup(f.srv.Close)

	c, err := f.srv.Dial()
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	c.SetPrependMessages(protocol.SetBroadcastFlags(monitorFlags))

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
