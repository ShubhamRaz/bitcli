// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print BitCLI version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "bitcli %s\ncommit %s\nbuilt %s\n", opts.version, opts.commit, opts.date)
			return nil
		},
	}
}

