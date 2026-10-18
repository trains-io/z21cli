package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

// maxMonitorLocos is the number of loco subscriptions a Z21 keeps per client
// (spec §4.1).
const maxMonitorLocos = 16

func newMonitorCmd() *cobra.Command {
	var locoArgs []string

	cmd := &cobra.Command{
		Use:     "monitor",
		Aliases: []string{"mon"},
		Short:   "Watch z21 broadcast events",
		Long: `Print subscribed z21 broadcast events as they arrive.

Subscribe first with "z21 sub add <NAME>", then run monitor. Uses the current
context session so the z21 command station sends events to this client. The
subscriptions are sent again at start and with a keep-alive every 30 seconds,
so they survive a minute without requests and a restart of the z21.

--loco ADDR (repeatable, at most 16) also subscribes to the changes of a loco:
monitor asks for it with LAN_X_GET_LOCO_INFO at start and with every
keep-alive, and adds XPRESSNET_UPDATES for this session unless
ALL_LOCO_INFO_UPDATES or XPRESSNET_UPDATES is subscribed. A loco is printed
only when it changed.

Press Ctrl+C to stop.`,
		Example: `  z21 monitor
  z21 monitor --loco 3 --loco 1234`,
		Args: NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			locos, err := parseMonitorLocos(locoArgs)
			if err != nil {
				return err
			}
			app := GetAppContext(cmd)
			if app == nil || app.Client == nil {
				return fmt.Errorf("Z21 connection not initialized")
			}
			if len(locos) > 0 {
				app.Client.SetPrependMessages(monitorLocoPrepend(effectiveBroadcastFlags(app.Context.Session), locos)...)
			}

			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()

			return runMonitor(ctx, app.Client, cmd.OutOrStdout(), client.DefaultKeepAliveInterval)
		},
	}
	cmd.Flags().StringSliceVar(&locoArgs, "loco", nil, "also watch loco ADDR (1-9999); repeatable or comma-separated, at most 16")
	return cmd
}

// parseMonitorLocos parses the --loco addresses, dropping duplicates.
func parseMonitorLocos(args []string) ([]uint16, error) {
	var locos []uint16
	seen := make(map[uint16]bool, len(args))
	for _, arg := range args {
		addr, err := parseLocoAddress(arg)
		if err != nil {
			return nil, err
		}
		if !seen[addr] {
			seen[addr] = true
			locos = append(locos, addr)
		}
	}
	if len(locos) > maxMonitorLocos {
		return nil, fmt.Errorf("too many --loco: %d, the z21 keeps at most %d per client", len(locos), maxMonitorLocos)
	}
	return locos, nil
}

// monitorLocoPrepend returns the datasets each keep-alive sends to subscribe
// to flags and to locos. The z21 sends a subscribed loco's changes only with
// XPRESSNET_UPDATES or ALL_LOCO_INFO_UPDATES, so XPRESSNET_UPDATES is added
// if neither is in flags.
func monitorLocoPrepend(flags uint32, locos []uint16) []protocol.Message {
	if flags&(protocol.BroadcastFlagXpressNet|protocol.BroadcastFlagAllLocoInfo) == 0 {
		flags |= protocol.BroadcastFlagXpressNet
	}
	msgs := []protocol.Message{protocol.SetBroadcastFlags(flags)}
	for _, addr := range locos {
		msgs = append(msgs, protocol.GetLocoInfo(addr))
	}
	return msgs
}

// runMonitor prints the events read from c to w until ctx is done, sending a
// keep-alive at once and then every keepAlive. The keep-alives carry the
// datasets prepended to c, which subscribe to the broadcasts: the Z21 may have
// forgotten the client since "z21 sub add", or forget it while monitoring when
// it restarts. LAN_X_LOCO_INFO is printed only when the loco changed.
func runMonitor(ctx context.Context, c *client.Client, w io.Writer, keepAlive time.Duration) error {
	if err := c.Send(ctx, protocol.GetSerialNumber()); err != nil {
		return fmt.Errorf("monitor %s: %w", c.Addr(), err)
	}

	ctx, cancel := context.WithCancel(ctx)
	keepAliveDone := make(chan struct{})
	go func() {
		defer close(keepAliveDone)
		_ = c.KeepAlive(ctx, keepAlive)
	}()
	defer func() {
		cancel()
		<-keepAliveDone
	}()

	fmt.Fprintln(w, "Waiting for Z21 events ...")
	// The replies to LAN_X_GET_LOCO_INFO in every keep-alive repeat the state.
	locos := make(map[uint16]protocol.LocoInfo)
	for {
		msgs, err := c.ReadPacket(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("monitor %s: %w", c.Addr(), err)
		}

		for _, msg := range msgs {
			if info, err := protocol.ParseLocoInfo(msg.Data); err == nil && msg.Header == protocol.HeaderLANX {
				if last, ok := locos[info.Address]; ok && last == info {
					continue
				}
				locos[info.Address] = info
			}
			if line, ok := formatMonitorEvent(msg); ok {
				fmt.Fprintln(w, line)
			}
		}
	}
}
