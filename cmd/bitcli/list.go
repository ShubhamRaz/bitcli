// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newListCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List downloaded BitNet models",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			models, err := a.models.List(cmd.Context())
			if err != nil {
				return err
			}
			if len(models) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No models installed.")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "NAME\tBACKEND\tQUANT\tPATH")
			for _, m := range models {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", m.UserID, m.Backend, m.Quantization, m.Path)
			}
			return nil
		},
	}
}

