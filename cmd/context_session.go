package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/trains-io/z21cli/internal/contextstore"
)

func clearCurrentContextSession(cmd *cobra.Command) error {
	app := GetAppContext(cmd)
	if app == nil {
		return nil
	}

	store, err := contextstore.Load("")
	if err != nil {
		return err
	}

	if !contextstore.ClearSession(store, app.Context.Name) {
		return fmt.Errorf("context %q not found in saved contexts", app.Context.Name)
	}

	if err := contextstore.Save("", store); err != nil {
		return err
	}

	app.Context.Session = nil
	return nil
}
