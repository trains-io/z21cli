package cmd

import (
	"testing"

	"github.com/trains-io/z21.go/protocol"
)

func TestFormatLockState(t *testing.T) {
	tests := []struct {
		code byte
		want string
	}{
		{protocol.CodeNoLock, "No Lock (all features permitted)"},
		{protocol.CodeZ21StartLocked, "Locked (driving and switching blocked)"},
		{protocol.CodeZ21StartUnlocked, "Unlocked (driving and switching permitted)"},
		{0x7f, "Unknown (0x7f)"},
	}

	for _, tt := range tests {
		if got := formatLockState(tt.code); got != tt.want {
			t.Fatalf("formatLockState(%#x) = %q, want %q", tt.code, got, tt.want)
		}
	}
}
