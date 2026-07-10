package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const requiresSubcommandAnnotation = "requiresSubcommand"

type subcommandRequiredError struct {
	cmd *cobra.Command
}

func (e *subcommandRequiredError) Error() string {
	return "a subcommand from the list below is required, use --help for full help including flags and arguments"
}

type usageError struct {
	cmd    *cobra.Command
	reason string
}

func (e *usageError) Error() string {
	if e.reason != "" {
		return e.reason
	}
	return "invalid command usage"
}

func requireSubcommand(cmd *cobra.Command) {
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations[requiresSubcommandAnnotation] = "true"
	cmd.RunE = func(c *cobra.Command, args []string) error {
		return &subcommandRequiredError{cmd: c}
	}
}

func requiresSubcommand(cmd *cobra.Command) bool {
	return cmd.Annotations != nil && cmd.Annotations[requiresSubcommandAnnotation] == "true"
}

func wrapArgs(validator cobra.PositionalArgs) cobra.PositionalArgs {
	if validator == nil {
		return nil
	}
	return func(cmd *cobra.Command, args []string) error {
		if err := validator(cmd, args); err != nil {
			return &usageError{cmd: cmd, reason: err.Error()}
		}
		return nil
	}
}

// NoArgs rejects unexpected positional arguments and shows command help on failure.
func NoArgs() cobra.PositionalArgs {
	return wrapArgs(cobra.NoArgs)
}

// ExactArgs requires exactly n positional arguments and shows command help on failure.
func ExactArgs(n int) cobra.PositionalArgs {
	return wrapArgs(cobra.ExactArgs(n))
}

// RangeArgs requires between min and max positional arguments and shows command help on failure.
func RangeArgs(min, max int) cobra.PositionalArgs {
	return wrapArgs(cobra.RangeArgs(min, max))
}

func setUsageOnError(cmd *cobra.Command) {
	cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
		return &usageError{cmd: c, reason: err.Error()}
	})
	for _, sub := range cmd.Commands() {
		setUsageOnError(sub)
	}
}

func printSubcommandRequired(err error) bool {
	var subErr *subcommandRequiredError
	if !errors.As(err, &subErr) {
		return false
	}

	fmt.Fprintf(os.Stderr, "error: %s\n\n", subErr.Error())
	_ = subErr.cmd.Help()
	return true
}

func printUsageError(err error) bool {
	var usageErr *usageError
	if !errors.As(err, &usageErr) {
		return false
	}

	fmt.Fprintf(os.Stderr, "error: %s\n\n", usageErr.Error())
	_ = usageErr.cmd.Help()
	return true
}
