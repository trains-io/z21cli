package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	var buf bytes.Buffer
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = orig
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestPrintContextFields(t *testing.T) {
	got := captureStdout(t, func() {
		fmt.Printf("z21 Configuration Context %q\n\n", "sim")
		printContextFields("", labeledValue{"Host", "127.0.0.1:21105"})
	})

	want := "" +
		`z21 Configuration Context "sim"` + "\n\n" +
		"Host: 127.0.0.1:21105\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPrintContextFieldsAlignsLabels(t *testing.T) {
	got := captureStdout(t, func() {
		printContextFields("  ",
			labeledValue{"Host", "192.168.2.6:21105"},
			labeledValue{"Session", "172.20.236.167:38991"},
		)
	})

	want := "" +
		"  Host:    192.168.2.6:21105\n" +
		"  Session: 172.20.236.167:38991\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPrintLabeledValuesAlignsColumns(t *testing.T) {
	got := captureStdout(t, func() {
		printLabeledValues(
			labeledValue{"Track power", "OFF"},
			labeledValue{"Emergency stop", "OFF"},
		)
	})

	want := "" +
		"FIELD          VALUE\n" +
		"--------------------\n" +
		"Track power    OFF\n" +
		"Emergency stop OFF\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}

func TestPrintLabeledValuesEmpty(t *testing.T) {
	printLabeledValues()
}

func TestPrintTableSingleColumn(t *testing.T) {
	got := captureStdout(t, func() {
		printTable("", []string{"SUBSCRIPTION"}, [][]string{{"SYSTEM_UPDATES"}})
	})

	want := "" +
		"SUBSCRIPTION\n" +
		"--------------\n" +
		"SYSTEM_UPDATES\n"
	if got != want {
		t.Fatalf("got:\n%q\nwant:\n%q", got, want)
	}
}
