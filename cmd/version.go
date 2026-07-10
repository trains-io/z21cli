package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Set at link time via -ldflags (see Makefile build target).
var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

const cliVersionPrefix = "z21 cli version"

func printCLIVersion() {
	fmt.Printf("%s %s\n", cliVersionPrefix, Version)
}

func initCLIVersionFlag(cmd *cobra.Command) {
	cmd.Version = Version
	cmd.SetVersionTemplate(cliVersionPrefix + " {{.Version}}\n")
}
