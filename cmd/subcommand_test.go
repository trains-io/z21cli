package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func captureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	origStdout := os.Stdout
	origStderr := os.Stderr
	os.Stdout = stdoutW
	os.Stderr = stderrW

	fn()

	stdoutW.Close()
	stderrW.Close()
	os.Stdout = origStdout
	os.Stderr = origStderr

	var stdoutBuf, stderrBuf bytes.Buffer
	if _, err := io.Copy(&stdoutBuf, stdoutR); err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(&stderrBuf, stderrR); err != nil {
		t.Fatal(err)
	}
	return stdoutBuf.String(), stderrBuf.String()
}

func TestRequireSubcommandShowsErrorAndUsage(t *testing.T) {
	root := &cobra.Command{Use: "z21", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(newContextCmd())

	stdout, stderr := captureOutput(t, func() {
		root.SetArgs([]string{"context"})
		if err := root.Execute(); err == nil {
			t.Fatal("expected error")
		} else if !printSubcommandRequired(err) {
			t.Fatalf("expected subcommandRequiredError, got %T: %v", err, err)
		}
	})

	if !strings.Contains(stderr, "error: a subcommand from the list below is required") {
		t.Fatalf("stderr missing error message:\n%s", stderr)
	}
	if !strings.Contains(stdout, "Usage:") {
		t.Fatalf("stdout missing usage:\n%s", stdout)
	}
	if !strings.Contains(stdout, "add") || !strings.Contains(stdout, "list") {
		t.Fatalf("stdout missing available commands:\n%s", stdout)
	}
}

func TestUsageErrorShowsSubcommandHelp(t *testing.T) {
	root := &cobra.Command{Use: "z21", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(newContextCmd())

	stdout, stderr := captureOutput(t, func() {
		root.SetArgs([]string{"context", "use"})
		err := root.Execute()
		if err == nil {
			t.Fatal("expected error")
		}
		if !printUsageError(err) {
			t.Fatalf("expected usageError, got %T: %v", err, err)
		}
	})

	if strings.Contains(stderr, "Error: accepts") {
		t.Fatalf("stderr should not contain generic arg error:\n%s", stderr)
	}
	if !strings.Contains(stderr, "error: accepts 1 arg(s), received 0") {
		t.Fatalf("stderr missing arg error message:\n%s", stderr)
	}
	if !strings.Contains(stdout, "use NAME") {
		t.Fatalf("stdout missing use subcommand help:\n%s", stdout)
	}
}

func TestRequireSubcommandSkipsAppContext(t *testing.T) {
	cmd := newPowerCmd()
	if needsAppContext(cmd, nil) {
		t.Fatal("expected power parent command to skip app context")
	}
}

func TestInfoWithoutSubcommandStillNeedsAppContext(t *testing.T) {
	cmd := newInfoCmd()
	if !needsAppContext(cmd, nil) {
		t.Fatal("expected info without subcommand to need app context")
	}
}
