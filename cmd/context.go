package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21.go"
)

type ContextInfo struct {
	Name    string       `json:"name"`
	Host    string       `json:"host"`
	Port    int          `json:"port"`
	Session *SessionInfo `json:"session,omitempty"`
}

type SessionInfo struct {
	LocalHost string `json:"local_host"`
	LocalPort int    `json:"local_port"`
}

type ContextStore struct {
	Current  string        `json:"current"`
	Contexts []ContextInfo `json:"contexts"`
}

var contextFile = filepath.Join(os.Getenv("HOME"), ".z21_contexts.json")

var contextCmd = &cobra.Command{
	Use:     "context",
	Aliases: []string{"ctx"},
	Short:   "Manage Z21 configuration contexts",
}

// ---------- subcommands ----------

// add NAME --host <HOST> --port <PORT>
var contextAddCmd = &cobra.Command{
	Use:   "add NAME",
	Short: "Add a new Z21 context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")

		store, err := loadContexts()
		if err != nil {
			return err
		}

		for _, c := range store.Contexts {
			if c.Name == name {
				return fmt.Errorf("context %q already exists", name)
			}
		}

		if host == "" {
			host = "127.0.0.1"
		}
		if port == 0 {
			port = 21105
		}

		store.Contexts = append(store.Contexts, ContextInfo{Name: name, Host: host, Port: port})

		if len(store.Contexts) == 1 {
			store.Current = name
		}

		if err := saveContexts(store); err != nil {
			return err
		}

		fmt.Printf("Context %q added\n", name)
		return nil
	},
}

// list | ls
var contextListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all saved contexts",
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := loadContexts()
		if err != nil {
			return err
		}

		if len(store.Contexts) == 0 {
			fmt.Println("No contexts saved")
			return nil
		}

		maxLen := 0
		for _, c := range store.Contexts {
			if l := len(c.Name); l > maxLen {
				maxLen = l
			}
		}

		for _, c := range store.Contexts {
			current := "( )"
			if store.Current == c.Name {
				current = "(*)"
			}
			fmt.Printf("%s %-*s %s:%d\n", current, maxLen+3, c.Name, c.Host, c.Port)
		}
		return nil
	},
}

// show
var contextShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current context",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := loadCurrentContext()
		if err != nil {
			return nil
		}
		fmt.Printf("%s %s:%d\n", c.Name, c.Host, c.Port)
		return nil
	},
}

// use NAME
var contextUseCmd = &cobra.Command{
	Use:   "use NAME",
	Short: "Select a saved context",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		store, err := loadContexts()
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
		if err := saveContexts(store); err != nil {
			return err
		}

		fmt.Printf("Current context set to %q\n", name)
		return nil
	},
}

// rm NAME
var contextRmCmd = &cobra.Command{
	Use:   "rm",
	Args:  cobra.ExactArgs(1),
	Short: "Remove a saved context",
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		store, err := loadContexts()
		if err != nil {
			return err
		}

		newContexts := []ContextInfo{}
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

		if err := saveContexts(store); err != nil {
			return err
		}

		fmt.Printf("Context %q removed\n", name)
		return nil
	},
}

// reset
var contextResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset current context and clear its session data",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if err := InitAppContext(cmd); err != nil {
			return err
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		app := GetAppContext(cmd)
		if app == nil || app.Conn == nil {
			return fmt.Errorf("Z21 connection not initialized")
		}

		_, err := Req(app.Conn, &z21.Logoff{})
		if err != nil {
			return err
		}

		store, err := loadContexts()
		if err != nil {
			return err
		}

		if store.Current == "" {
			fmt.Println("No current context set")
			return nil
		}

		newContexts := []ContextInfo{}
		for _, c := range store.Contexts {
			if c.Name == store.Current {
				c.Session = nil
				fmt.Printf("Context %q reset and session data cleared\n", c.Name)
			}
			newContexts = append(newContexts, c)
		}

		store.Contexts = newContexts
		if err := saveContexts(store); err != nil {
			return err
		}

		return nil
	},
	PostRunE: func(cmd *cobra.Command, args []string) error {
		CloseAppContext(cmd)
		return nil
	},
}

// ---------- helpers ----------

func loadContexts() (*ContextStore, error) {
	store := &ContextStore{}
	if _, err := os.Stat(contextFile); errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	data, err := os.ReadFile(contextFile)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, store); err != nil {
		return nil, err
	}
	return store, nil
}

func loadCurrentContext() (*ContextInfo, error) {
	store, err := loadContexts()
	if err != nil {
		return nil, fmt.Errorf("failed to load contexts: %w", err)
	}

	if store.Current == "" {
		return nil, fmt.Errorf("no current context set, run `z21 ctx use <NAME>` first")
	}

	for _, c := range store.Contexts {
		if c.Name == store.Current {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("current context %q not found in saved contexts", store.Current)
}

func saveContexts(store *ContextStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(contextFile, data, 0644)
}

func saveSessionInfo(sess *SessionInfo, ctx *ContextInfo) error {
	store, err := loadContexts()
	if err != nil {
		return err
	}

	found := false
	for i := range store.Contexts {
		if store.Contexts[i].Name == ctx.Name {
			found = true
			store.Contexts[i].Session = sess
			break
		}
	}
	if !found {
		return fmt.Errorf("context %q not found in saved contexts", ctx.Name)
	}

	if err := saveContexts(store); err != nil {
		return err
	}
	return nil
}

// ---------- init ----------

func init() {
	contextCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error { return nil }
	contextCmd.PersistentPostRun = func(cmd *cobra.Command, args []string) {}
	contextCmd.AddCommand(
		contextAddCmd,
		contextListCmd,
		contextShowCmd,
		contextUseCmd,
		contextRmCmd,
		contextResetCmd,
	)
	contextAddCmd.Flags().String("host", "", "Z21 host address")
	contextAddCmd.Flags().Int("port", 0, "Z21 port")
}
