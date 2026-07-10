package main

import (
	"os"

	"github.com/trains-io/z21cli/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
