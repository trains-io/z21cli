package cmd

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21cli/internal/contextstore"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newBroadcastCmd() *cobra.Command {
	var (
		host    string
		port    int
		timeout time.Duration
	)

	cmd := &cobra.Command{
		Use:     "broadcast",
		Aliases: []string{"bc"},
		Short:   "View or change LAN broadcast subscription flags",
		Long: `Broadcast flags control which z21 UDP events this client receives (spec §2.16).

set sends LAN_SET_BROADCASTFLAGS; get sends LAN_GET_BROADCASTFLAGS. Flags are per client
(IP address + local UDP port). With a saved context, both commands reuse the same session.`,
	}
	registerZ21PersistentDialFlags(cmd, &host, &port, &timeout)
	cmd.AddCommand(newBroadcastGetCmd(), newBroadcastSetCmd())
	requireSubcommand(cmd)
	return cmd
}

func newBroadcastGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Show active broadcast flags for this client session",
		Long:  "Sends LAN_GET_BROADCASTFLAGS (spec §2.17). Without --host, uses the current context session. With --host, queries directly without saving a session.",
		Args:  NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			directHost, err := directHost(cmd)
			if err != nil {
				return err
			}
			port, err := dialPort(cmd)
			if err != nil {
				return err
			}
			timeout, err := dialTimeout(cmd)
			if err != nil {
				return err
			}

			return runZ21Op(cmd, directHost, port, timeout, func(ctx context.Context, c *client.Client) error {
				msgs, err := c.Call(ctx, protocol.GetBroadcastFlags())
				if err != nil {
					return fmt.Errorf("broadcast get %s: %w", c.Addr(), err)
				}

				flags, err := protocol.BroadcastFlagsFromMessages(msgs)
				if err != nil {
					return fmt.Errorf("broadcast get %s: %w", c.Addr(), err)
				}

				printBroadcastFlags(flags)
				return nil
			})
		},
	}
}

func newBroadcastSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set FLAGS",
		Short: "Set broadcast subscription flags",
		Long: `FLAGS is a hex (0x101) or decimal bitmask. Sends LAN_SET_BROADCASTFLAGS (spec §2.16).
This replaces the current mask; use subscribe add to enable one more group without clearing others.

Without --host, uses the current context and saves the mask for future sessions.

Examples:
  z21 broadcast set 0x101
  z21 broadcast set 0x103 --host 192.168.0.42
  z21 broadcast set 0x103 --host 192.168.0.42 --port 21105`,
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := parseBroadcastFlags(args[0])
			if err != nil {
				return err
			}

			directHost, err := directHost(cmd)
			if err != nil {
				return err
			}
			port, err := dialPort(cmd)
			if err != nil {
				return err
			}
			timeout, err := dialTimeout(cmd)
			if err != nil {
				return err
			}

			return runZ21Op(cmd, directHost, port, timeout, func(ctx context.Context, c *client.Client) error {
				if err := c.Send(ctx, protocol.SetBroadcastFlags(flags)); err != nil {
					return fmt.Errorf("broadcast set %s: %w", c.Addr(), err)
				}

				if directHost == "" {
					app := GetAppContext(cmd)
					if app == nil || app.Context.Session == nil {
						return fmt.Errorf("Z21 connection not initialized")
					}
					if err := saveContextBroadcastFlags(app.Context.Name, app.Context.Session, flags); err != nil {
						return err
					}
					app.Context.Session.BroadcastFlags = &flags
				}

				printBroadcastFlags(flags)
				return nil
			})
		},
	}
}

func parseBroadcastFlags(s string) (uint32, error) {
	v, err := strconv.ParseUint(s, 0, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid broadcast flags %q: %w", s, err)
	}
	return uint32(v), nil
}

func effectiveBroadcastFlags(session *contextstore.Session) uint32 {
	if session != nil && session.BroadcastFlags != nil {
		return *session.BroadcastFlags
	}
	return protocol.DefaultBroadcastFlags
}

func printBroadcastFlags(mask uint32) {
	rows := []labeledValue{{
		label: "Summary",
		value: protocol.FormatBroadcastFlags(mask),
	}}
	for _, flag := range protocol.KnownBroadcastFlags {
		state := "off"
		if protocol.HasBroadcastFlag(mask, flag.Value) {
			state = "on"
		}
		rows = append(rows, labeledValue{flag.Name, state})
	}
	printLabelValueTable("FLAG", "STATE", rows...)
}

func saveContextBroadcastFlags(contextName string, session *contextstore.Session, flags uint32) error {
	if session == nil {
		return fmt.Errorf("no active session for context %q", contextName)
	}

	store, err := contextstore.Load("")
	if err != nil {
		return err
	}

	updated := *session
	updated.BroadcastFlags = &flags
	if err := contextstore.SaveSession(store, contextName, &updated); err != nil {
		return err
	}
	return contextstore.Save("", store)
}
