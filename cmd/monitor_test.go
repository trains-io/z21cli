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

func TestRunMonitorKeepsSession(t *testing.T) {
	const (
		sessionTimeout = 200 * time.Millisecond
		keepAlive      = 50 * time.Millisecond
	)

	keepAlives := make(chan struct{}, 100)
	station := server.HandlerFunc(func(w server.ResponseWriter, r *server.Request) {
		if r.Message.Header == protocol.HeaderLANGetSerialNumber {
			keepAlives <- struct{}{}
			w.Reply(protocol.EncodeSerialNumber(1))
		}
	})
	srv := servertest.NewServer(station, server.WithSessionTimeout(sessionTimeout))
	t.Cleanup(srv.Close)

	c, err := srv.Dial()
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	t.Cleanup(func() { c.Close() })
	if err := c.Send(t.Context(), protocol.SetBroadcastFlags(protocol.BroadcastFlagXpressNet)); err != nil {
		t.Fatalf("SetBroadcastFlags: %v", err)
	}

	out, outW := io.Pipe()
	lines := make(chan string)
	go func() {
		defer close(lines)
		s := bufio.NewScanner(out)
		for s.Scan() {
			lines <- s.Text()
		}
	}()

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- runMonitor(ctx, c, outW, keepAlive)
		outW.Close()
	}()

	timeout := time.After(5 * time.Second)
	next := func() string {
		t.Helper()
		select {
		case line := <-lines:
			return line
		case <-timeout:
			t.Fatal("monitor printed nothing")
			return ""
		}
	}

	if got := next(); got != "Waiting for Z21 events ..." {
		t.Fatalf("first line = %q, want the waiting message", got)
	}
	// Ten keep-alives span more than twice the session timeout.
	for range 10 {
		select {
		case <-keepAlives:
		case <-timeout:
			t.Fatal("keep-alives did not arrive")
		}
	}

	if n := srv.Server.Broadcast(protocol.BroadcastFlagXpressNet, protocol.EncodeTrackPowerBC(false)); n != 1 {
		t.Fatalf("Broadcast reached %d clients, want 1 (session kept)", n)
	}
	if got, want := next(), formatTrackPowerBCEvent(false); got != want {
		t.Fatalf("event line = %q, want %q", got, want)
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("runMonitor after cancel = %v, want nil", err)
	}
}
