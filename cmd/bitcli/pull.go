// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newPullCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "pull MODEL",
		Short: "Download a BitNet model from Hugging Face",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			artifact, err := a.models.Resolve(args[0])
			if err != nil {
				return err
			}
			m, err := a.downloads.PullModel(cmd.Context(), artifact, os.Stdout)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pulled %s -> %s\n", m.UserID, m.Path)
			return nil
		},
	}
}

