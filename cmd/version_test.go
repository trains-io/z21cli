package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
)

func TestPrintCLIVersion(t *testing.T) {
	origVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() {
		Version = origVersion
	})

	var buf bytes.Buffer
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	printCLIVersion()

	w.Close()
	os.Stdout = orig
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	want := "z21 cli version 1.2.3\n"
	if buf.String() != want {
		t.Fatalf("output = %q, want %q", buf.String(), want)
	}
}

func TestCLIVersionFlag(t *testing.T) {
	origVersion := Version
	Version = "1.2.3"
	t.Cleanup(func() {
		Version = origVersion
		initCLIVersionFlag(rootCmd)
	})

	initCLIVersionFlag(rootCmd)

	var buf bytes.Buffer
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}

	w.Close()
	os.Stdout = orig
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}

	want := "z21 cli version 1.2.3\n"
	if buf.String() != want {
		t.Fatalf("output = %q, want %q", buf.String(), want)
	}
}

func TestVersionCmdNeedsAppContext(t *testing.T) {
	cmd := newVersionCmd()
	if !needsAppContext(cmd, nil) {
		t.Fatal("expected version to need app context without --host")
	}
	if err := cmd.Flags().Set("host", "127.0.0.1"); err != nil {
		t.Fatal(err)
	}
	if needsAppContext(cmd, nil) {
		t.Fatal("expected version to skip app context with --host")
	}
}
