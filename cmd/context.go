package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21cli/internal/contextstore"
	"github.com/trains-io/z21.go/client"
	"github.com/trains-io/z21.go/protocol"
)

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "context",
		Aliases: []string{"ctx"},
		Short:   "Manage z21 configuration contexts",
	}

	cmd.AddCommand(
		newContextAddCmd(),
		newContextListCmd(),
		newContextShowCmd(),
		newContextUseCmd(),
		newContextRmCmd(),
		newContextResetCmd(),
	)
	requireSubcommand(cmd)
	return cmd
}

func contextConfigFileRow() (labeledValue, error) {
	path, err := contextstore.DefaultPath()
	if err != nil {
		return labeledValue{}, err
	}
	return labeledValue{label: "Config file", value: path}, nil
}

func printContextConfigFile() error {
	row, err := contextConfigFileRow()
	if err != nil {
		return err
	}
	printContextFields("", row)
	return nil
}

func newContextAddCmd() *cobra.Command {
	var host string
	var port int
	var use bool
	var description string

	cmd := &cobra.Command{
		Use:   "add NAME [HOST]",
		Short: "Add a new z21 context",
		Args:  RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			resolvedHost := host
			if len(args) == 2 {
				if cmd.Flags().Changed("host") {
					return fmt.Errorf("cannot specify host as both argument and --host flag")
				}
				resolvedHost = args[1]
			}
			if resolvedHost == "" {
				resolvedHost = "127.0.0.1"
			}

			store, err := contextstore.Load("")
			if err != nil {
				return err
			}

			for _, c := range store.Contexts {
				if c.Name == name {
					return fmt.Errorf("context %q already exists", name)
				}
			}

			store.Contexts = append(store.Contexts, contextstore.Info{
				Name:        name,
				Host:        resolvedHost,
				Port:        port,
				Description: description,
			})

			if len(store.Contexts) == 1 || use {
				store.Current = name
			}

			if err := contextstore.Save("", store); err != nil {
				return err
			}

			fmt.Printf("z21 Configuration Context %q\n\n", name)
			var rows []labeledValue
			if description != "" {
				rows = append(rows, labeledValue{label: "Description", value: description})
			}
			rows = append(rows, labeledValue{
				label: "Host",
				value: fmt.Sprintf("%s:%d", resolvedHost, port),
			})
			configRow, err := contextConfigFileRow()
			if err != nil {
				return err
			}
			rows = append(rows, configRow)
			printContextFields("", rows...)
			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "z21 host address")
	cmd.Flags().IntVar(&port, "port", client.DefaultPort, "z21 port")
	cmd.Flags().BoolVar(&use, "use", false, "use as default")
	cmd.Flags().StringVar(&description, "description", "", "optional context description")
	return cmd
}

func newContextListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all saved contexts",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := contextstore.Load("")
			if err != nil {
				return err
			}

			if len(store.Contexts) == 0 {
				fmt.Println("No contexts saved")
				return printContextConfigFile()
			}

			fmt.Printf("Known contexts:\n\n")
			rows := make([][]string, 0, len(store.Contexts))
			for _, c := range store.Contexts {
				marker := "( )"
				if store.Current == c.Name {
					marker = "(*)"
				}
				rows = append(rows, []string{marker, c.Name, fmt.Sprintf("%s:%d", c.Host, c.Port), c.Description})
			}
			printTable("", []string{"", "NAME", "ENDPOINT", "DESCRIPTION"}, rows)
			fmt.Println()
			return printContextConfigFile()
		},
	}
}

func newContextShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current context",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := contextstore.Load("")
			if err != nil {
				return err
			}

			current, err := contextstore.Current(store)
			if err != nil {
				return err
			}

			fmt.Printf("z21 Configuration Context %q\n\n", current.Name)
			rows := []labeledValue{{
				label: "Host",
				value: fmt.Sprintf("%s:%d", current.Host, current.Port),
			}}
			if current.Description != "" {
				rows = append(rows, labeledValue{label: "Description", value: current.Description})
			}
			if current.Session != nil {
				rows = append(rows, labeledValue{
					label: "Session",
					value: fmt.Sprintf("%s:%d", current.Session.LocalHost, current.Session.LocalPort),
				})
			} else {
				rows = append(rows, labeledValue{label: "Session", value: "-"})
			}
			configRow, err := contextConfigFileRow()
			if err != nil {
				return err
			}
			rows = append(rows, configRow)
			printContextFields("", rows...)
			return nil
		},
	}
}

func newContextUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use NAME",
		Short: "Select a saved context",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			store, err := contextstore.Load("")
			if err != nil {
				return err
			}

			found := false
			for _, c := range store.Contexts {
				if c.Name == name {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("context %q not found", name)
			}

			store.Current = name
			if err := contextstore.Save("", store); err != nil {
				return err
			}

			fmt.Printf("Current context set to %q\n", name)
			return nil
		},
	}
}

func newContextRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm NAME",
		Short: "Remove a saved context",
		Args:  ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			store, err := contextstore.Load("")
			if err != nil {
				return err
			}

			newContexts := make([]contextstore.Info, 0, len(store.Contexts))
			found := false
			for _, c := range store.Contexts {
				if c.Name == name {
					found = true
					continue
				}
				newContexts = append(newContexts, c)
			}
			if !found {
				return fmt.Errorf("context %q not found", name)
			}

			store.Contexts = newContexts
			if store.Current == name {
				store.Current = ""
			}

			if err := contextstore.Save("", store); err != nil {
				return err
			}

			fmt.Printf("Context %q removed\n", name)
			return nil
		},
	}
}

func newContextResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Reset current context and clear its session data",
		RunE: func(cmd *cobra.Command, args []string) error {
			app := GetAppContext(cmd)
			if app == nil || app.Client == nil {
				return fmt.Errorf("Z21 connection not initialized")
			}

			if app.Context.Session != nil {
				ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
				defer cancel()
				if err := app.Client.Send(ctx, protocol.Logoff()); err != nil {
					return fmt.Errorf("logoff: %w", err)
				}
			}

			if err := clearCurrentContextSession(cmd); err != nil {
				return err
			}

			fmt.Printf("Context %q reset and session data cleared\n", app.Context.Name)
			return nil
		},
	}
}
