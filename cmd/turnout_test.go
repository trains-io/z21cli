package cmd

import (
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/protocol"
	"github.com/trains-io/z21.go/server"
	"github.com/trains-io/z21.go/server/servertest"
)

type turnoutSwitchRequest struct {
	addr uint16
	sw   protocol.TurnoutSwitch
	at   time.Time
}

// switchingStation is an in-process Z21 that keeps turnout positions and
// accessory values and records the set requests it receives.
type switchingStation struct {
	srv *servertest.Server

	mu          sync.Mutex
	turnouts    map[uint16]protocol.TurnoutPosition
	accessories map[uint16]byte
	switches    []turnoutSwitchRequest
	values      []protocol.ExtAccessoryInfo
}

func startSwitchingStation(t *testing.T) *switchingStation {
	t.Helper()
	st := &switchingStation{
		turnouts:    make(map[uint16]protocol.TurnoutPosition),
		accessories: make(map[uint16]byte),
	}
	st.srv = servertest.NewServer(server.HandlerFunc(st.serve))
	t.Cleanup(st.srv.Close)
	return st
}

func (st *switchingStation) serve(w server.ResponseWriter, r *server.Request) {
	if r.Message.Header != protocol.HeaderLANX {
		return
	}
	data := r.Message.Data
	st.mu.Lock()
	defer st.mu.Unlock()
	if addr, err := protocol.ParseGetTurnoutInfo(data); err == nil {
		w.Reply(protocol.EncodeTurnoutInfo(protocol.TurnoutInfo{Address: addr, Position: st.turnouts[addr]}))
	} else if addr, sw, err := protocol.ParseSetTurnout(data); err == nil {
		st.switches = append(st.switches, turnoutSwitchRequest{addr: addr, sw: sw, at: time.Now()})
		if sw.Activate {
			st.turnouts[addr] = protocol.TurnoutPositionOutput1
			if sw.Output2 {
				st.turnouts[addr] = protocol.TurnoutPositionOutput2
			}
		}
	} else if addr, err := protocol.ParseGetExtAccessoryInfo(data); err == nil {
		info := protocol.ExtAccessoryInfo{Address: addr, Status: protocol.ExtAccessoryStatusUnknown}
		if v, ok := st.accessories[addr]; ok {
			info.Value, info.Status = v, protocol.ExtAccessoryStatusValid
		}
		w.Reply(protocol.EncodeExtAccessoryInfo(info))
	} else if addr, value, err := protocol.ParseSetExtAccessory(data); err == nil {
		st.values = append(st.values, protocol.ExtAccessoryInfo{Address: addr, Value: value})
		st.accessories[addr] = value
	}
}

// requests returns the set requests received so far.
func (st *switchingStation) requests() ([]turnoutSwitchRequest, []protocol.ExtAccessoryInfo) {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.switches, st.values
}

// run executes "z21 args..." with the turnout and accessory commands against
// the station and returns its standard output.
func (st *switchingStation) run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	host, port, err := net.SplitHostPort(st.srv.Addr)
	if err != nil {
		t.Fatal(err)
	}
	root := &cobra.Command{Use: "z21", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(newTurnoutCmd(), newAccessoryCmd())
	root.SetArgs(append(args, "--host", host, "--port", port))
	var runErr error
	out := captureStdout(t, func() { runErr = root.Execute() })
	return out, runErr
}

func (st *switchingStation) mustRun(t *testing.T, args ...string) string {
	t.Helper()
	out, err := st.run(t, args...)
	if err != nil {
		t.Fatalf("z21 %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func TestTurnoutGet(t *testing.T) {
	st := startSwitchingStation(t)

	got := st.mustRun(t, "turnout", "get", "12")
	want := "" +
		"TURNOUT  STATE\n" +
		"---------------------\n" +
		"Address  12\n" +
		"Position not switched\n"
	if got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestTurnoutSet(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantSw    []protocol.TurnoutSwitch
		wantPos   string
		wantPulse time.Duration
	}{
		{
			name:      "activates and deactivates after the default pulse",
			args:      []string{"0", "2"},
			wantSw:    []protocol.TurnoutSwitch{{Activate: true, Output2: true}, {Output2: true}},
			wantPos:   "output 2",
			wantPulse: defaultTurnoutPulse,
		},
		{
			name:      "custom pulse, hex address",
			args:      []string{"0x10", "1", "--pulse", "200ms"},
			wantSw:    []protocol.TurnoutSwitch{{Activate: true}, {}},
			wantPos:   "output 1",
			wantPulse: 200 * time.Millisecond,
		},
		{
			name:    "pulse 0 leaves the output activated",
			args:    []string{"2047", "1", "--pulse", "0"},
			wantSw:  []protocol.TurnoutSwitch{{Activate: true}},
			wantPos: "output 1",
		},
		{
			name:    "queue",
			args:    []string{"5", "2", "--queue"},
			wantSw:  []protocol.TurnoutSwitch{{Activate: true, Output2: true, Queue: true}, {Output2: true, Queue: true}},
			wantPos: "output 2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := startSwitchingStation(t)

			out := st.mustRun(t, append([]string{"turnout", "set"}, tt.args...)...)
			if !strings.Contains(out, "Position "+tt.wantPos+"\n") {
				t.Fatalf("output:\n%s\nwant position %q", out, tt.wantPos)
			}

			switches, _ := st.requests()
			if len(switches) != len(tt.wantSw) {
				t.Fatalf("got %d LAN_X_SET_TURNOUT, want %d: %+v", len(switches), len(tt.wantSw), switches)
			}
			addr, _ := parseAccessoryAddress(tt.args[0])
			for i, want := range tt.wantSw {
				if switches[i].addr != addr || switches[i].sw != want {
					t.Fatalf("LAN_X_SET_TURNOUT #%d = %d %+v, want %d %+v", i+1, switches[i].addr, switches[i].sw, addr, want)
				}
			}
			if len(switches) == 2 {
				if pulse := switches[1].at.Sub(switches[0].at); pulse < tt.wantPulse {
					t.Fatalf("deactivated after %s, want at least %s", pulse, tt.wantPulse)
				}
			}
		})
	}
}

func TestAccessorySet(t *testing.T) {
	st := startSwitchingStation(t)

	got := st.mustRun(t, "accessory", "set", "3", "0x12")
	want := "" +
		"ACCESSORY STATE\n" +
		"-------------------\n" +
		"Address   3\n" +
		"Value     18 (0x12)\n"
	if got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
	if _, values := st.requests(); len(values) != 1 || values[0] != (protocol.ExtAccessoryInfo{Address: 3, Value: 0x12}) {
		t.Fatalf("LAN_X_SET_EXT_ACCESSORY = %+v, want address 3 value 0x12", values)
	}

	got = st.mustRun(t, "accessory", "get", "3")
	if got != want {
		t.Fatalf("get output:\n%s\nwant:\n%s", got, want)
	}
}

func TestAccessoryGetUnknown(t *testing.T) {
	st := startSwitchingStation(t)

	got := st.mustRun(t, "accessory", "get", "0x7ff")
	if !strings.Contains(got, "Address   2047\n") || !strings.Contains(got, "Value     unknown\n") {
		t.Fatalf("output:\n%s\nwant address 2047, value unknown", got)
	}
}

func TestSwitchingInvalidArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "turnout address too high", args: []string{"turnout", "set", "2048", "1"}, wantErr: `invalid address "2048": want 0 to 2047`},
		{name: "turnout address not a number", args: []string{"turnout", "get", "west"}, wantErr: `invalid address "west"`},
		{name: "turnout output 0", args: []string{"turnout", "set", "1", "0"}, wantErr: `invalid output "0": want 1 or 2`},
		{name: "turnout output 3", args: []string{"turnout", "set", "1", "3"}, wantErr: `invalid output "3": want 1 or 2`},
		{name: "pulse negative", args: []string{"turnout", "set", "1", "1", "--pulse", "-1ms"}, wantErr: "invalid --pulse -1ms"},
		{name: "pulse as long as timeout", args: []string{"turnout", "set", "1", "1", "--pulse", "1s", "--timeout", "1s"}, wantErr: "less than --timeout 1s"},
		{name: "accessory address too high", args: []string{"accessory", "get", "0x800"}, wantErr: `invalid address "0x800"`},
		{name: "accessory value too high", args: []string{"accessory", "set", "3", "256"}, wantErr: `invalid VALUE "256": want 0 to 255`},
		{name: "accessory value not a number", args: []string{"accessory", "set", "3", "red"}, wantErr: `invalid VALUE "red"`},
	}
	st := startSwitchingStation(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := st.run(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
	if switches, values := st.requests(); len(switches) != 0 || len(values) != 0 {
		t.Fatalf("station got %+v, %+v; want no requests for invalid arguments", switches, values)
	}
}
