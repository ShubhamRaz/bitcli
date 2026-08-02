// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/bitcli/bitcli/internal/api"
	"github.com/bitcli/bitcli/internal/process"
	"github.com/spf13/cobra"
)

func newServeCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Start the local BitCLI API server",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, process.InterruptSignal())
			defer stop()
			server := api.NewServer(api.Dependencies{
				Config:    a.cfg,
				Models:    a.models,
				Downloads: a.downloads,
				Runtime:   a.runtime,
				Cache:     a.cache,
				Hardware:  a.hardware,
				Version:   opts.version,
				Logger:    a.log,
			})
			fmt.Fprintf(cmd.OutOrStdout(), "BitCLI listening on http://%s:%d\n", a.cfg.API.Host, a.cfg.API.Port)
			return server.Run(ctx)
		},
	}
}
