package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPowerCommandsNeedAppContext(t *testing.T) {
	powerCmd := newPowerCmd()
	onCmd, _, _ := powerCmd.Find([]string{"on"})
	offCmd, _, _ := powerCmd.Find([]string{"off"})
	statusCmd, _, _ := powerCmd.Find([]string{"status"})

	if !needsAppContext(onCmd, nil) {
		t.Fatal("expected power on to need app context")
	}
	if !needsAppContext(offCmd, nil) {
		t.Fatal("expected power off to need app context")
	}
	if !needsAppContext(statusCmd, nil) {
		t.Fatal("expected power status to need app context")
	}

	if err := powerCmd.PersistentFlags().Set("host", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if needsAppContext(onCmd, nil) {
		t.Fatal("expected power on with --host to skip app context")
	}
}

func TestPrintPowerStatus(t *testing.T) {
	tests := []struct {
		state byte
		want  string
	}{
		{0x00, "" +
			"POWER          STATUS\n" +
			"---------------------\n" +
			"Track power    ON\n" +
			"Emergency stop OFF\n"},
		{0x03, "" +
			"POWER          STATUS\n" +
			"---------------------\n" +
			"Track power    OFF\n" +
			"Emergency stop ON\n"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		orig := os.Stdout
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}
		os.Stdout = w

		printPowerStatus(tt.state)

		w.Close()
		os.Stdout = orig
		if _, err := io.Copy(&buf, r); err != nil {
			t.Fatal(err)
		}

		if buf.String() != tt.want {
			t.Fatalf("state %#x:\ngot:\n%q\nwant:\n%q", tt.state, buf.String(), tt.want)
		}
	}
}
