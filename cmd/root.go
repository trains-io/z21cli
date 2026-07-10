package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "z21",
	Short:        "CLI for z21 model railway command stations",
	SilenceUsage: true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if !needsAppContext(cmd, args) {
			return nil
		}
		return InitAppContext(cmd)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if !needsAppContext(cmd, args) {
			return
		}
		CloseAppContext(cmd)
	},
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		if printSubcommandRequired(err) || printUsageError(err) {
			return err
		}
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return err
	}
	return nil
}

func init() {
	initCLIVersionFlag(rootCmd)
	registerTraceFlag(rootCmd)

	rootCmd.AddGroup(&cobra.Group{
		ID:    "version",
		Title: "Version commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "context",
		Title: "Context commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "track",
		Title: "Track commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "system",
		Title: "System commands:",
	})
	rootCmd.AddGroup(&cobra.Group{
		ID:    "monitor",
		Title: "Monitor commands:",
	})

	for _, cmd := range []*cobra.Command{
		newVersionCmd(),
		newFirmwareCmd(),
		newCodeCmd(),
		newHWInfoCmd(),
	} {
		cmd.GroupID = "version"
		rootCmd.AddCommand(cmd)
	}
	rootCmd.AddCommand(newInfoCmd())
	for _, cmd := range []*cobra.Command{
		newPingCmd(),
		newSerialCmd(),
		newStatusCmd(),
		newSystemCmd(),
		newDiscoverCmd(),
		newCanCmd(),
	} {
		cmd.GroupID = "system"
		rootCmd.AddCommand(cmd)
	}
	for _, cmd := range []*cobra.Command{
		newBroadcastCmd(),
		newSubscribeCmd(),
		newMonitorCmd(),
	} {
		cmd.GroupID = "monitor"
		rootCmd.AddCommand(cmd)
	}
	for _, cmd := range []*cobra.Command{
		newLogoffCmd(),
		newContextCmd(),
	} {
		cmd.GroupID = "context"
		rootCmd.AddCommand(cmd)
	}
	for _, cmd := range []*cobra.Command{
		newPowerCmd(),
		newStopCmd(),
	} {
		cmd.GroupID = "track"
		rootCmd.AddCommand(cmd)
	}
	setUsageOnError(rootCmd)
}
