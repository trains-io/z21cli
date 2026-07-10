package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/protocol"
)

func newSubscribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subscribe",
		Aliases: []string{"sub"},
		Short:   "Manage z21 broadcast subscriptions",
		Long: fmt.Sprintf(`Subscribe to named z21 broadcast event groups (spec §2.16).

Subscriptions are stored on the current context session and applied to the z21
command station when you connect. Use monitor to receive subscribed events.

Known subscriptions:
%s`, subscriptionHelpText()),
	}
	cmd.AddCommand(
		newSubscribeAddCmd(),
		newSubscribeRemoveCmd(),
		newSubscribeListCmd(),
	)
	requireSubcommand(cmd)
	return cmd
}

func newSubscribeAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add NAME",
		Short: "Subscribe to a broadcast event group",
		Long: fmt.Sprintf(`Enable one named broadcast group on the current session.

Supported subscription groups:
%s`, subscriptionHelpText()),
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flag, err := lookupSubscription(args[0])
			if err != nil {
				return err
			}
			return mutateSubscription(cmd, flag, true, args[0])
		},
	}
}

func newSubscribeRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove NAME",
		Aliases: []string{"rm", "del"},
		Short:   "Unsubscribe from a broadcast event group",
		Long: fmt.Sprintf(`Disable one named broadcast group on the current session.

Supported subscription groups:
%s`, subscriptionHelpText()),
		Args: ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flag, err := lookupSubscription(args[0])
			if err != nil {
				return err
			}
			return mutateSubscription(cmd, flag, false, args[0])
		},
	}
	return cmd
}

func newSubscribeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List active subscriptions for this session",
		Args:    NoArgs(),
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)
			if app == nil || app.Client == nil {
				return fmt.Errorf("Z21 connection not initialized")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			defer cancel()

			msgs, err := app.Client.Call(ctx, protocol.GetBroadcastFlags())
			if err != nil {
				return fmt.Errorf("subscribe list: %w", err)
			}

			flags, err := protocol.BroadcastFlagsFromMessages(msgs)
			if err != nil {
				return fmt.Errorf("subscribe list: %w", err)
			}

			printSubscriptionList(flags)
			return nil
		},
	}
}

func mutateSubscription(cmd *cobra.Command, flag uint32, add bool, name string) error {
	app := GetAppContext(cmd)
	if app == nil || app.Client == nil {
		return fmt.Errorf("Z21 connection not initialized")
	}
	if app.Context.Session == nil {
		return fmt.Errorf("Z21 connection not initialized")
	}

	mask := effectiveBroadcastFlags(app.Context.Session)
	if add {
		mask |= flag
	} else {
		mask &^= flag
	}

	ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
	defer cancel()

	if err := app.Client.Send(ctx, protocol.SetBroadcastFlags(mask)); err != nil {
		return fmt.Errorf("subscribe %s: %w", app.Client.Addr(), err)
	}

	if err := saveContextBroadcastFlags(app.Context.Name, app.Context.Session, mask); err != nil {
		return err
	}
	app.Context.Session.BroadcastFlags = &mask

	if add {
		fmt.Printf("Subscribed to %q\n", stringsToSubscriptionName(name))
	} else {
		fmt.Printf("Unsubscribed from %q\n", stringsToSubscriptionName(name))
	}
	return nil
}

func stringsToSubscriptionName(name string) string {
	flag, err := lookupSubscription(name)
	if err != nil {
		return name
	}
	for _, sub := range knownSubscriptions {
		if sub.Flag == flag {
			return sub.Name
		}
	}
	return name
}

func printSubscriptionList(mask uint32) {
	fmt.Printf("Broadcast mask: %s\n", protocol.FormatBroadcastFlags(mask))

	var rows []labeledValue
	for _, sub := range knownSubscriptions {
		if !protocol.HasBroadcastFlag(mask, sub.Flag) {
			continue
		}
		rows = append(rows, labeledValue{sub.Name, subscriptionDescription(sub.Flag)})
	}

	if len(rows) == 0 {
		fmt.Println("\nNo active subscriptions.")
		return
	}

	fmt.Print("\nActive subscriptions:\n\n")
	printContextFields("", rows...)
}
