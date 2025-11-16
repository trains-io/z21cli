package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

type ctxkey string

type AppContext struct {
	Conn        *z21.Conn
	ContextName string
	Host        string
	Port        int
	Session     *SessionInfo
	Resumed     bool
	Logger      zerolog.Logger
}

const appCtxKey = ctxkey("app")

var verbose bool

var rootCmd = &cobra.Command{
	Use:           "z21",
	Short:         "CLI for Roco Z21 control station.",
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if err := InitAppContext(cmd); err != nil {
			return err
		}
		return nil
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		CloseAppContext(cmd)
	},
}

func Execute(v, c, d string) {
	Version = v
	Commit = c
	Date = d
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}

func CloseAppContext(cmd *cobra.Command) {
	appCtx := GetAppContext(cmd)
	if appCtx.Conn != nil {
		appCtx.Conn.Close()
	}
}

func InitAppContext(cmd *cobra.Command) error {
	v, _ := cmd.Flags().GetBool("verbose")
	verbose = v

	appCtx := &AppContext{}
	if verbose {
		initLogger(appCtx)
	}

	c, err := loadCurrentContext()
	if err != nil {
		return err
	}
	appCtx.Logger.Debug().Msgf("Z21 context: %s", c.Name)

	var dialer z21.CustomDialer
	var resumed bool
	session := c.Session
	if session != nil && session.LocalPort > 0 {
		resumed = true
		appCtx.Logger.Debug().Msgf("Z21 session: resuming ...")
		dialer = &net.Dialer{
			LocalAddr: &net.UDPAddr{
				IP:   net.IPv4zero,
				Port: session.LocalPort,
			},
		}
	} else {
		appCtx.Logger.Debug().Msgf("Z21 session: starting new ...")
		dialer = &net.Dialer{}
	}

	conn, err := z21.Connect(
		c.Host,
		z21.Verbose(verbose),
		z21.SetCustomDialer(dialer),
	)
	if err != nil {
		return fmt.Errorf("failed to connect to Z21: %w", err)
	}

	localHost, localPort, err := getLocalPort(conn)
	if err != nil {
		return err
	}

	appCtx.Conn = conn
	appCtx.ContextName = c.Name
	appCtx.Host = c.Host
	appCtx.Port = c.Port
	appCtx.Session = &SessionInfo{
		LocalHost: localHost,
		LocalPort: localPort,
	}
	ctx := context.Background()
	cmd.SetContext(context.WithValue(ctx, appCtxKey, appCtx))

	if !resumed {
		if err := saveSessionInfo(appCtx.Session, c); err != nil {
			return err
		}
		appCtx.Logger.Debug().Msgf("Z21 session: session saved")
	}

	return nil
}

func GetAppContext(cmd *cobra.Command) *AppContext {
	return cmd.Context().Value(appCtxKey).(*AppContext)
}

func Req[T z21.Serializable](conn *z21.Conn, msg T) (T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	var empty T
	res, err := conn.SendRcv(ctx, msg)
	if err != nil {
		return empty, err
	}
	m, ok := res.(T)
	if !ok {
		return empty, nil
	}

	return m, nil
}

func initLogger(ctx *AppContext) {
	ctx.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		Level(zerolog.DebugLevel).
		With().
		Timestamp().
		Logger()
}

func getLocalPort(c *z21.Conn) (string, int, error) {
	if c == nil {
		return "", 0, fmt.Errorf("invalid connection")
	}

	host, portStr, err := net.SplitHostPort(c.GetLocalPort().String())
	if err != nil {
		return "", 0, err
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid UDP port: %s", portStr)
	}
	return host, port, err
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(subCmd)
	rootCmd.AddCommand(monitorCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(powerCmd)
	rootCmd.AddCommand(canCmd)
}
