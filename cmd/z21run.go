package cmd

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
)

func registerZ21DialFlags(cmd *cobra.Command, host *string, port *int, timeout *time.Duration) {
	cmd.Flags().StringVar(host, "host", "", "z21 host address (direct mode; skips context)")
	cmd.Flags().IntVarP(port, "port", "p", 0, "z21 UDP port (overrides context or defaults to 21105 with --host)")
	cmd.Flags().DurationVar(timeout, "timeout", 5*time.Second, "request timeout")
}

func registerZ21PersistentDialFlags(cmd *cobra.Command, host *string, port *int, timeout *time.Duration) {
	cmd.PersistentFlags().StringVar(host, "host", "", "z21 host address (direct mode; skips context)")
	cmd.PersistentFlags().IntVarP(port, "port", "p", 0, "z21 UDP port (overrides context or defaults to 21105 with --host)")
	cmd.PersistentFlags().DurationVar(timeout, "timeout", 5*time.Second, "request timeout")
}

func directHost(cmd *cobra.Command) (string, error) {
	return lookupStringFlag(cmd, "host")
}

func dialPort(cmd *cobra.Command) (int, error) {
	port, ok, err := lookupIntFlag(cmd, "port")
	if err != nil {
		return 0, err
	}
	if !ok || port == 0 {
		return 0, nil
	}
	return port, nil
}

func dialTimeout(cmd *cobra.Command) (time.Duration, error) {
	timeout, ok, err := lookupDurationFlag(cmd, "timeout")
	if err != nil {
		return 0, err
	}
	if !ok {
		return 5 * time.Second, nil
	}
	return timeout, nil
}

func usesDirectDial(cmd *cobra.Command) bool {
	host, err := directHost(cmd)
	return err == nil && host != ""
}

func lookupStringFlag(cmd *cobra.Command, name string) (string, error) {
	for c := cmd; c != nil; c = c.Parent() {
		if flag := c.Flags().Lookup(name); flag != nil && flag.Changed {
			return c.Flags().GetString(name)
		}
		if flag := c.PersistentFlags().Lookup(name); flag != nil && flag.Changed {
			return c.PersistentFlags().GetString(name)
		}
	}
	for c := cmd; c != nil; c = c.Parent() {
		if c.Flags().Lookup(name) != nil {
			return c.Flags().GetString(name)
		}
		if c.PersistentFlags().Lookup(name) != nil {
			return c.PersistentFlags().GetString(name)
		}
	}
	return "", nil
}

func lookupIntFlag(cmd *cobra.Command, name string) (int, bool, error) {
	for c := cmd; c != nil; c = c.Parent() {
		if flag := c.Flags().Lookup(name); flag != nil {
			v, err := c.Flags().GetInt(name)
			return v, true, err
		}
		if flag := c.PersistentFlags().Lookup(name); flag != nil {
			v, err := c.PersistentFlags().GetInt(name)
			return v, true, err
		}
	}
	return 0, false, nil
}

func lookupDurationFlag(cmd *cobra.Command, name string) (time.Duration, bool, error) {
	for c := cmd; c != nil; c = c.Parent() {
		if flag := c.Flags().Lookup(name); flag != nil {
			v, err := c.Flags().GetDuration(name)
			return v, true, err
		}
		if flag := c.PersistentFlags().Lookup(name); flag != nil {
			v, err := c.PersistentFlags().GetDuration(name)
			return v, true, err
		}
	}
	return 0, false, nil
}

func runZ21Op(
	cmd *cobra.Command,
	host string,
	port int,
	timeout time.Duration,
	op func(context.Context, *client.Client) error,
) error {
	reqCtx, cancel := context.WithTimeout(cmd.Context(), timeout)
	defer cancel()
	return runZ21Client(cmd, host, port, reqCtx, op)
}

func runZ21Session(
	cmd *cobra.Command,
	host string,
	port int,
	op func(context.Context, *client.Client) error,
) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	return runZ21Client(cmd, host, port, ctx, op)
}

func runZ21Client(
	cmd *cobra.Command,
	host string,
	port int,
	ctx context.Context,
	op func(context.Context, *client.Client) error,
) error {
	plan, err := planZ21Dial(host, port, GetAppContext(cmd))
	if err != nil {
		return err
	}
	if plan.useApp {
		app := GetAppContext(cmd)
		configureClientTrace(cmd, app.Client)
		return op(ctx, app.Client)
	}

	c, err := client.Dial(net.JoinHostPort(plan.host, strconv.Itoa(plan.port)))
	if err != nil {
		return err
	}
	defer c.Close()
	configureClientTrace(cmd, c)
	return op(ctx, c)
}

type z21DialPlan struct {
	host   string
	port   int
	useApp bool
}

func planZ21Dial(host string, port int, app *AppContext) (z21DialPlan, error) {
	if host != "" {
		dialPort := port
		if dialPort == 0 {
			dialPort = client.DefaultPort
		}
		return z21DialPlan{host: host, port: dialPort}, nil
	}

	if app == nil || app.Client == nil {
		return z21DialPlan{}, fmt.Errorf("Z21 connection not initialized")
	}

	dialPort := app.Context.Port
	if port != 0 {
		dialPort = port
	}
	if dialPort == app.Context.Port {
		return z21DialPlan{host: app.Context.Host, port: dialPort, useApp: true}, nil
	}
	return z21DialPlan{host: app.Context.Host, port: dialPort}, nil
}
