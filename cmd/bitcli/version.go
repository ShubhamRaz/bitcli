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
			out := cmd.OutOrStdout()
			ver := opts.version
			if ver == "" {
				ver = "v0.1.1-dev"
			}
			fmt.Fprintf(out, "bitcli %s (%s, %s)\n", ver, opts.commit, opts.date)
			return nil
		},
	}
}

