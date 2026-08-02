// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"fmt"
	"os"

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
			out := cmd.OutOrStdout()
			if len(models) == 0 {
				fmt.Fprintln(out)
				fmt.Fprintln(out, "  No models installed yet.")
				fmt.Fprintln(out, "  Download the recommended 1-bit model:")
				fmt.Fprintln(out, "    bitcli run microsoft/BitNet-b1.58-2B-4T")
				fmt.Fprintln(out)
				return nil
			}

			fmt.Fprintln(out)
			fmt.Fprintln(out, "  ╔═══════════════════════════════════════════════════════════════════════════════════╗")
			fmt.Fprintln(out, "  ║                                Installed BitNet Models                            ║")
			fmt.Fprintln(out, "  ╚═══════════════════════════════════════════════════════════════════════════════════╝")
			fmt.Fprintln(out)
			fmt.Fprintf(out, "    %-34s %-8s %-10s %-16s\n", "MODEL NAME", "QUANT", "SIZE", "BACKEND")
			fmt.Fprintln(out, "    ───────────────────────────────────────────────────────────────────────────────")
			for _, m := range models {
				sizeStr := "-"
				if fi, err := os.Stat(m.Path); err == nil {
					sizeBytes := fi.Size()
					if sizeBytes >= 1024*1024*1024 {
						sizeStr = fmt.Sprintf("%.2f GiB", float64(sizeBytes)/(1024*1024*1024))
					} else {
						sizeStr = fmt.Sprintf("%.1f MiB", float64(sizeBytes)/(1024*1024))
					}
				}
				quant := m.Quantization
				if quant == "" {
					quant = "i2_s"
				}
				backendName := m.Backend
				if backendName == "" {
					backendName = "bitnet (1-bit)"
				}
				fmt.Fprintf(out, "    %-34s %-8s %-10s %-16s\n", m.UserID, quant, sizeStr, backendName)
			}
			fmt.Fprintln(out, "    ───────────────────────────────────────────────────────────────────────────────")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "  Usage Tips:")
			fmt.Fprintln(out, "    Run a prompt : bitcli run <model> -p \"Your prompt\"")
			fmt.Fprintln(out, "    Start chat   : bitcli chat -m <model>")
			fmt.Fprintln(out, "    Start server : bitcli serve")
			fmt.Fprintln(out)
			return nil
		},
	}
}

