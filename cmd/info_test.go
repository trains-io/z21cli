package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/trains-io/z21.go/protocol"
)

func TestPrintAllDeviceInfo(t *testing.T) {
	var buf bytes.Buffer
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	printAllDeviceInfo(deviceInfo{
		Serial: "265070",
		HW: protocol.HWInfo{
			HwType:          protocol.HwTypeZ21New,
			FirmwareVersion: 0x00000142,
		},
		XVersion: protocol.XVersion{
			XBusVersion:      0x24,
			CommandStationID: 0x12,
		},
		XFirmware: protocol.XFirmware{
			VersionMSB: 0x01,
			VersionLSB: 0x42,
		},
		LockCode: protocol.CodeNoLock,
	}, infoAvailability{
		serial: true, hw: true, version: true, firmware: true, code: true,
	})

	w.Close()
	os.Stdout = orig
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	for _, want := range []string{
		"FIELD              VALUE",
		"Serial number      265070",
		"Hardware type      black Z21 (2013)",
		"Firmware (HW)      1.42",
		"Firmware (X-Bus)   1.42",
		"X-Bus version      2.4",
		"Command station ID Z21",
		"Lock code          0x00",
		"Status             all features permitted",
	} {
		if !bytes.Contains(buf.Bytes(), []byte(want)) {
			t.Fatalf("output missing %q:\n%s", want, out)
		}
	}
}

func TestPrintAllDeviceInfoPartial(t *testing.T) {
	var buf bytes.Buffer
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	printAllDeviceInfo(deviceInfo{
		HW: protocol.HWInfo{
			HwType:          protocol.HwTypeZ21New,
			FirmwareVersion: 0x00000142,
		},
	}, infoAvailability{hw: true})

	w.Close()
	os.Stdout = orig
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if bytes.Contains(buf.Bytes(), []byte("Serial number")) {
		t.Fatalf("did not expect serial in partial output:\n%s", out)
	}
	if !bytes.Contains(buf.Bytes(), []byte("Hardware type")) {
		t.Fatalf("expected hardware type in partial output:\n%s", out)
	}
}

func TestNeedsAppContextInfo(t *testing.T) {
	infoCmd := newInfoCmd()
	if !needsAppContext(infoCmd, nil) {
		t.Fatal("expected info to need app context")
	}
}
