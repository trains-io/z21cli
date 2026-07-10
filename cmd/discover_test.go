package cmd

import (
	"bytes"
	"io"
	"net"
	"os"
	"testing"

	"github.com/trains-io/z21.go/discovery"
	"github.com/trains-io/z21.go/protocol"
)

func TestDiscoverCmdNeedsNoAppContext(t *testing.T) {
	cmd := newDiscoverCmd()
	if needsAppContext(cmd, []string{"192.168.2.0/24"}) {
		t.Fatal("expected discover to skip app context")
	}
}

func TestPrintDiscoverResultsJSON(t *testing.T) {
	devices := []discovery.Device{{
		IP:   net.ParseIP("192.168.2.6"),
		Port: 21105,
		HWInfo: protocol.HWInfo{
			HwType:          protocol.HwTypeZ21New,
			FirmwareVersion: 0x00000143,
		},
	}}

	var buf bytes.Buffer
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	if err := printDiscoverResults(devices, "json"); err != nil {
		t.Fatal(err)
	}

	w.Close()
	os.Stdout = orig
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	want := `[{"ip":"192.168.2.6","port":21105,"reachable":true,"hw_type":513,"firmware":323}]` + "\n"
	if buf.String() != want {
		t.Fatalf("output = %q, want %q", buf.String(), want)
	}
}
