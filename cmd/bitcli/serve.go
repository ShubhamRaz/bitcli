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
			out := cmd.OutOrStdout()
			fmt.Fprintln(out)
			fmt.Fprintln(out, "  ╔══════════════════════════════════════════════════════════════╗")
			fmt.Fprintln(out, "  ║                BitCLI OpenAI-Compatible API Server          ║")
			fmt.Fprintln(out, "  ╚══════════════════════════════════════════════════════════════╝")
			fmt.Fprintf(out, "    Host & Port : http://%s:%d\n", a.cfg.API.Host, a.cfg.API.Port)
			fmt.Fprintf(out, "    Chat API    : http://%s:%d/v1/chat/completions\n", a.cfg.API.Host, a.cfg.API.Port)
			fmt.Fprintf(out, "    Models API  : http://%s:%d/v1/models\n", a.cfg.API.Host, a.cfg.API.Port)
			fmt.Fprintf(out, "    Health API  : http://%s:%d/health\n", a.cfg.API.Host, a.cfg.API.Port)
			fmt.Fprintln(out, "  ──────────────────────────────────────────────────────────────")
			fmt.Fprintln(out, "  Ready to accept incoming OpenAI API requests. Press Ctrl+C to stop.")
			fmt.Fprintln(out)
			return server.Run(ctx)
		},
	}
}
