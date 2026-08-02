// Command bitcli is the entry point for the BitCLI application.
package main

import (
	"fmt"
	"os"
)

var (
	version = "0.1.0-dev"
	commit  = "unknown"
	date    = "unknown"
)

func main() {
	if err := newRootCommand(version, commit, date).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

