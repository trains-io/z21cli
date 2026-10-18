package cmd

import (
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/protocol"
	"github.com/trains-io/z21.go/server"
	"github.com/trains-io/z21.go/server/servertest"
)

type locoFunctionRequest struct {
	fn     uint8
	action protocol.LocoFunctionAction
}

// locoStation is an in-process Z21 with one loco state, shared by all
// addresses, that records the loco requests it receives.
type locoStation struct {
	srv *servertest.Server

	mu   sync.Mutex
	loco protocol.LocoInfo
	got  locoRequests
}

// locoRequests are the loco requests a locoStation received.
type locoRequests struct {
	getInfos  int
	drives    []protocol.LocoDrive
	functions []locoFunctionRequest
	eStops    int
}

func startLocoStation(t *testing.T, loco protocol.LocoInfo) *locoStation {
	t.Helper()
	st := &locoStation{loco: loco}
	st.srv = servertest.NewServer(server.HandlerFunc(st.serve))
	t.Cleanup(st.srv.Close)
	return st
}

func (st *locoStation) serve(w server.ResponseWriter, r *server.Request) {
	if r.Message.Header != protocol.HeaderLANX {
		return
	}
	data := r.Message.Data
	st.mu.Lock()
	defer st.mu.Unlock()
	if addr, err := protocol.ParseGetLocoInfo(data); err == nil {
		st.got.getInfos++
		info := st.loco
		info.Address = addr
		w.Reply(protocol.EncodeLocoInfo(info))
	} else if drive, err := protocol.ParseSetLocoDrive(data); err == nil {
		st.got.drives = append(st.got.drives, drive)
		st.loco.SpeedSteps = protocol.LocoInfoSpeedSteps(drive.SpeedSteps)
		st.loco.Forward = drive.Forward
		st.loco.Speed = drive.Speed
	} else if _, fn, action, err := protocol.ParseSetLocoFunction(data); err == nil {
		st.got.functions = append(st.got.functions, locoFunctionRequest{fn, action})
	} else if _, err := protocol.ParseSetLocoEStop(data); err == nil {
		st.got.eStops++
		st.loco.Speed = protocol.LocoSpeedEStop
	}
}

// requests returns the loco requests received so far.
func (st *locoStation) requests() locoRequests {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.got
}

// run executes "z21 loco args..." against the station and returns its
// standard output.
func (st *locoStation) run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	host, port, err := net.SplitHostPort(st.srv.Addr)
	if err != nil {
		t.Fatal(err)
	}
	root := &cobra.Command{Use: "z21", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(newLocoCmd())
	root.SetArgs(append(append([]string{"loco"}, args...), "--host", host, "--port", port))
	var runErr error
	out := captureStdout(t, func() { runErr = root.Execute() })
	return out, runErr
}

func (st *locoStation) mustRun(t *testing.T, args ...string) string {
	t.Helper()
	out, err := st.run(t, args...)
	if err != nil {
		t.Fatalf("z21 loco %s: %v", strings.Join(args, " "), err)
	}
	return out
}

func TestLocoInfo(t *testing.T) {
	st := startLocoStation(t, protocol.LocoInfo{
		SpeedSteps:    protocol.LocoInfoSpeedSteps28,
		Speed:         0x12,
		Headlight:     true,
		FunctionsF1F4: 0x04,
		Busy:          true,
	})

	got := st.mustRun(t, "info", "3")
	want := "" +
		"LOCO        STATE\n" +
		"----------------------------------\n" +
		"Address     3\n" +
		"Format      DCC\n" +
		"Speed steps 28\n" +
		"Direction   reverse\n" +
		"Speed       2/28\n" +
		"Functions   F0 F3\n" +
		"Busy        yes (another throttle)\n"
	if got != want {
		t.Fatalf("output:\n%s\nwant:\n%s", got, want)
	}
}

func TestLocoDrive(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantDrive    protocol.LocoDrive
		wantGetInfos int
		wantSpeed    string
	}{
		{
			name:         "keeps direction and speed steps",
			args:         []string{"drive", "3", "10"},
			wantDrive:    protocol.LocoDrive{Address: 3, SpeedSteps: protocol.LocoSpeedSteps28, Forward: false, Speed: 0x16},
			wantGetInfos: 2,
			wantSpeed:    "Speed       10/28\n",
		},
		{
			name:         "direction and speed steps given",
			args:         []string{"drive", "3", "126", "--forward", "--steps", "128"},
			wantDrive:    protocol.LocoDrive{Address: 3, SpeedSteps: protocol.LocoSpeedSteps128, Forward: true, Speed: 0x7F},
			wantGetInfos: 1,
			wantSpeed:    "Speed       126/126\n",
		},
		{
			name:         "only the direction given",
			args:         []string{"drive", "3", "0", "--forward"},
			wantDrive:    protocol.LocoDrive{Address: 3, SpeedSteps: protocol.LocoSpeedSteps28, Forward: true},
			wantGetInfos: 2,
			wantSpeed:    "Speed       0/28\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := startLocoStation(t, protocol.LocoInfo{SpeedSteps: protocol.LocoInfoSpeedSteps28, Speed: 0x12})

			out := st.mustRun(t, tt.args...)
			got := st.requests()
			if len(got.drives) != 1 || got.drives[0] != tt.wantDrive {
				t.Fatalf("drive requests = %+v, want %+v", got.drives, tt.wantDrive)
			}
			if got.getInfos != tt.wantGetInfos {
				t.Fatalf("LAN_X_GET_LOCO_INFO requests = %d, want %d", got.getInfos, tt.wantGetInfos)
			}
			if !strings.Contains(out, tt.wantSpeed) {
				t.Fatalf("output:\n%s\nwant the new state with %q", out, tt.wantSpeed)
			}
		})
	}
}

func TestLocoDriveSpeedOutOfRange(t *testing.T) {
	st := startLocoStation(t, protocol.LocoInfo{SpeedSteps: protocol.LocoInfoSpeedSteps14})

	_, err := st.run(t, "drive", "3", "15")
	got := st.requests()
	if err == nil || !strings.Contains(err.Error(), "speed step 15 out of range [0, 14]") {
		t.Fatalf("error = %v, want speed step out of range for the loco's 14 speed steps", err)
	}
	if len(got.drives) != 0 {
		t.Fatalf("drive requests = %+v, want none", got.drives)
	}
}

func TestLocoFunc(t *testing.T) {
	st := startLocoStation(t, protocol.LocoInfo{SpeedSteps: protocol.LocoInfoSpeedSteps128, Forward: true})

	out := st.mustRun(t, "func", "3", "0=on", "1=toggle", "31=off")
	got := st.requests()
	want := []locoFunctionRequest{{0, protocol.LocoFunctionOn}, {1, protocol.LocoFunctionToggle}, {31, protocol.LocoFunctionOff}}
	if len(got.functions) != len(want) {
		t.Fatalf("function requests = %+v, want %+v", got.functions, want)
	}
	for i := range want {
		if got.functions[i] != want[i] {
			t.Fatalf("function requests = %+v, want %+v", got.functions, want)
		}
	}
	if !strings.Contains(out, "Address     3\n") {
		t.Fatalf("output:\n%s\nwant the loco's state", out)
	}
}

func TestLocoStop(t *testing.T) {
	st := startLocoStation(t, protocol.LocoInfo{SpeedSteps: protocol.LocoInfoSpeedSteps128, Forward: true, Speed: 0x29})

	out := st.mustRun(t, "stop", "3")
	got := st.requests()
	if got.eStops != 1 {
		t.Fatalf("LAN_X_SET_LOCO_E_STOP requests = %d, want 1", got.eStops)
	}
	if !strings.Contains(out, "Speed       emergency stop\n") || !strings.Contains(out, "Direction   forward\n") {
		t.Fatalf("output:\n%s\nwant an emergency stop, direction kept", out)
	}
}

func TestLocoArgErrors(t *testing.T) {
	st := startLocoStation(t, protocol.LocoInfo{})

	tests := []struct {
		args    []string
		wantErr string
	}{
		{args: []string{"info", "0"}, wantErr: `invalid loco address "0"`},
		{args: []string{"info", "10000"}, wantErr: `invalid loco address "10000"`},
		{args: []string{"info", "abc"}, wantErr: `invalid loco address "abc"`},
		{args: []string{"drive", "3", "fast"}, wantErr: `invalid SPEED "fast"`},
		{args: []string{"drive", "3", "300"}, wantErr: `invalid SPEED "300"`},
		{args: []string{"drive", "3", "10", "--steps", "27"}, wantErr: "invalid --steps 27"},
		{args: []string{"drive", "3", "10", "--forward", "--reverse"}, wantErr: "none of the others can be"},
		{args: []string{"func", "3", "32=on"}, wantErr: `invalid function "32=on"`},
		{args: []string{"func", "3", "1"}, wantErr: `invalid function "1"`},
		{args: []string{"func", "3", "1=maybe"}, wantErr: `invalid function "1=maybe"`},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			_, err := st.run(t, tt.args...)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
	if got := st.requests(); got.getInfos != 0 || len(got.drives) != 0 || len(got.functions) != 0 || got.eStops != 0 {
		t.Fatal("invalid commands sent requests")
	}
}

func TestLocoCommandsNeedAppContext(t *testing.T) {
	locoCmd := newLocoCmd()
	for _, name := range []string{"info", "drive", "func", "stop"} {
		sub, _, err := locoCmd.Find([]string{name})
		if err != nil {
			t.Fatalf("missing loco %s: %v", name, err)
		}
		if !needsAppContext(sub, nil) {
			t.Errorf("expected loco %s to need app context", name)
		}
		if !shouldApplyBroadcastFlags(sub) {
			t.Errorf("expected loco %s to apply the context's broadcast flags", name)
		}
		if needsSessionPort(sub) {
			t.Errorf("expected loco %s to fall back to a free port", name)
		}
	}
	if needsAppContext(locoCmd, nil) {
		t.Error("expected loco without a subcommand to skip app context")
	}
}
