package cmd

import (
	"testing"

	"github.com/trains-io/z21.go/protocol"
)

func TestParseDeviceInfoFromMessages(t *testing.T) {
	msgs := []protocol.Message{
		{
			Header: protocol.HeaderLANGetSerialNumber,
			Data:   []byte{0x0E, 0xB0, 0x04, 0x00},
		},
		{
			Header: protocol.HeaderLANGetHWInfo,
			Data:   []byte{0x01, 0x02, 0x00, 0x00, 0x42, 0x01, 0x00, 0x00},
		},
		{
			Header: protocol.HeaderLANX,
			Data:   []byte{0x63, 0x21, 0x30, 0x12, 0x60},
		},
		{
			Header: protocol.HeaderLANX,
			Data:   []byte{0xF3, 0x0A, 0x01, 0x42, 0xB8},
		},
		{
			Header: protocol.HeaderLANGetCode,
			Data:   []byte{0x00},
		},
	}

	info, avail, failures := parseDeviceInfoFromMessages(msgs)
	if len(failures) != 0 {
		t.Fatalf("unexpected failures: %v", failures)
	}
	if !avail.any() {
		t.Fatal("expected all sections available")
	}
	if info.Serial == "" {
		t.Fatal("expected serial number")
	}
	if info.LockCode != protocol.CodeNoLock {
		t.Fatalf("LockCode = %#x, want %#x", info.LockCode, protocol.CodeNoLock)
	}
}

func TestParseDeviceInfoFromMessagesPartial(t *testing.T) {
	msgs := []protocol.Message{
		{
			Header: protocol.HeaderLANGetHWInfo,
			Data:   []byte{0x01, 0x02, 0x00, 0x00, 0x42, 0x01, 0x00, 0x00},
		},
	}

	_, avail, failures := parseDeviceInfoFromMessages(msgs)
	if !avail.hw {
		t.Fatal("expected hw to be available")
	}
	if avail.serial {
		t.Fatal("did not expect serial to be available")
	}
	if len(failures) != 4 {
		t.Fatalf("failures len = %d, want 4", len(failures))
	}
}
