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
			out := cmd.OutOrStdout()
			fmt.Fprintln(out)
			fmt.Fprintf(out, "  ==> Pulling model %s\n", args[0])
			m, err := a.downloads.PullModel(cmd.Context(), artifact, os.Stdout)
			if err != nil {
				return err
			}
			fmt.Fprintln(out)
			fmt.Fprintf(out, "  ✓  Model successfully downloaded to %s\n", m.Path)
			fmt.Fprintf(out, "  ✓  Run prompt: bitcli run %s -p \"Your prompt\"\n", m.UserID)
			fmt.Fprintln(out)
			return nil
		},
	}
}

