// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"context"
	"fmt"
	"path/filepath"

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
			target := filepath.Join(a.paths.BackendDir, "bitnet", "current")
			installer := bitnet.Installer{Runner: a.runner}
			if err := installer.EnsureClone(context.Background(), a.cfg.Backend.BitNet.RepoURL, target); err != nil {
				return err
			}
			if err := installer.CheckoutRevision(context.Background(), target, a.cfg.Backend.BitNet.Revision); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "bitnet.cpp ready at %s\n", target)
			return nil
		},
	})
	return cmd
}

