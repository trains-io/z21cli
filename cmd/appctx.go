package cmd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
	"github.com/trains-io/z21cli/internal/contextstore"
)

type ctxKey struct{}

// AppContext holds the active Z21 connection and selected configuration context.
type AppContext struct {
	Client  *client.Client
	Context contextstore.Info
	Resumed bool
}

func InitAppContext(cmd *cobra.Command) error {
	store, err := contextstore.Load("")
	if err != nil {
		return err
	}

	current, err := contextstore.Current(store)
	if err != nil {
		return err
	}

	port := current.Port
	if port == 0 {
		port = client.DefaultPort
	}
	current.Port = port

	localPort := 0
	resumed := false
	if current.Session != nil && current.Session.LocalPort > 0 {
		localPort = current.Session.LocalPort
		resumed = true
	}

	addr := net.JoinHostPort(current.Host, strconv.Itoa(port))
	z21, err := client.DialLocal(addr, localPort)
	fallback := false
	if resumed && errors.Is(err, syscall.EADDRINUSE) {
		if needsSessionPort(cmd) {
			return fmt.Errorf("%s needs local UDP port %d of context %q, which is in use, probably by z21 monitor: stop monitor first",
				cmd.CommandPath(), localPort, current.Name)
		}
		z21, err = client.DialLocal(addr, 0)
		fallback = true
	}
	if err != nil {
		return fmt.Errorf("failed to connect to Z21: %w", err)
	}
	configureClientTrace(cmd, z21)

	app := &AppContext{
		Client:  z21,
		Context: *current,
		Resumed: resumed,
	}
	if fallback {
		// As with --host: the command does not act on the context's session.
		app.Context.Session = nil
	}

	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	// A fallback session ends with the command; subscribing it would only send
	// broadcasts to a closed port.
	if shouldApplyBroadcastFlags(cmd) && !fallback {
		z21.SetPrependMessages(protocol.SetBroadcastFlags(effectiveBroadcastFlags(current.Session)))
	}

	if !resumed {
		host, port, err := localEndpoint(z21)
		if err != nil {
			z21.Close()
			return err
		}
		session := &contextstore.Session{
			LocalHost: host,
			LocalPort: port,
		}
		if err := contextstore.SaveSession(store, current.Name, session); err != nil {
			z21.Close()
			return err
		}
		if err := contextstore.Save("", store); err != nil {
			z21.Close()
			return err
		}
		app.Context.Session = session
	}

	cmd.SetContext(context.WithValue(parent, ctxKey{}, app))
	return nil
}

func GetAppContext(cmd *cobra.Command) *AppContext {
	if cmd == nil || cmd.Context() == nil {
		return nil
	}
	app, _ := cmd.Context().Value(ctxKey{}).(*AppContext)
	return app
}

func CloseAppContext(cmd *cobra.Command) {
	app := GetAppContext(cmd)
	if app != nil && app.Client != nil {
		_ = app.Client.Close()
	}
}

func needsAppContext(cmd *cobra.Command, args []string) bool {
	if cmd == nil {
		return false
	}

	if requiresSubcommand(cmd) {
		return false
	}

	if isUnderContext(cmd) && cmd.Name() != "reset" {
		return false
	}

	if usesDirectDial(cmd) {
		return false
	}
	if isUnderBroadcast(cmd) {
		switch cmd.Name() {
		case "set", "get":
			return true
		}
	}

	switch cmd.Name() {
	case "ping", "serial", "version", "firmware", "code", "hwinfo", "status", "system", "info", "logoff", "reset", "monitor", "mon", "stop":
		return true
	default:
		if isUnderPower(cmd) {
			return true
		}
		if isUnderCan(cmd) {
			return true
		}
		if isUnderSubscribe(cmd) {
			return true
		}
		return false
	}
}

func isUnderBroadcast(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "broadcast" {
			return true
		}
	}
	return false
}

func shouldApplyBroadcastFlags(cmd *cobra.Command) bool {
	if isUnderBroadcast(cmd) {
		return false
	}
	if isUnderSubscribe(cmd) {
		switch cmd.Name() {
		case "add", "remove", "rm", "del":
			return false
		}
	}
	if isUnderPower(cmd) {
		switch cmd.Name() {
		case "on", "off":
			return false
		}
	}
	if cmd.Name() == "stop" {
		return false
	}
	return true
}

// needsSessionPort reports whether cmd must use the context's pinned local
// port: the Z21 keeps broadcast flags and sessions per IP address and port, so
// these commands would act on the wrong session from another port. Other
// commands fall back to a free port while the pinned one is in use.
func needsSessionPort(cmd *cobra.Command) bool {
	if isUnderSubscribe(cmd) || isUnderBroadcast(cmd) {
		return true
	}
	if isUnderContext(cmd) && cmd.Name() == "reset" {
		return true
	}
	switch cmd.Name() {
	case "logoff", "monitor":
		return true
	}
	return false
}

func isUnderSubscribe(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "subscribe" || c.Name() == "sub" {
			return true
		}
	}
	return false
}

func isUnderPower(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "power" {
			return true
		}
	}
	return false
}

func isUnderCan(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "can" {
			return true
		}
	}
	return false
}

func isUnderContext(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "context" {
			return true
		}
	}
	return false
}

func localEndpoint(c *client.Client) (string, int, error) {
	addr := c.LocalAddr()
	if addr == nil {
		return "", 0, fmt.Errorf("invalid connection")
	}

	host, portStr, err := net.SplitHostPort(addr.String())
	if err != nil {
		return "", 0, err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid UDP port: %s", portStr)
	}
	return host, port, nil
}
