// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"fmt"

	"github.com/bitcli/bitcli/internal/runtime/backend/bitnet"
	"github.com/spf13/cobra"
)

func newUpdateCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update BitCLI or managed backend assets",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(cmd.OutOrStdout(), "bitcli %s is installed. Use your package manager or release binary to update BitCLI.\n", opts.version)
			return nil
		},
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "backend bitnet",
		Short: "Clone or update the managed official Microsoft BitNet backend",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "bitnet" {
				return fmt.Errorf("unsupported backend %q", args[0])
			}
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			// Same directory the runtime adapter uses, honoring
			// backend.bitnet.path when configured.
			target := bitnet.NewBuilder(a.cfg, a.paths).BackendDir()
			installer := bitnet.Installer{Runner: a.runner}
			allowClean := a.cfg.Backend.BitNet.Path == ""
			if err := installer.EnsureClone(cmd.Context(), a.cfg.Backend.BitNet.RepoURL, target, allowClean); err != nil {
				return err
			}
			if err := installer.CheckoutRevision(cmd.Context(), target, a.cfg.Backend.BitNet.Revision); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "bitnet.cpp ready at %s\n", target)
			return nil
		},
	})
	return cmd
}
