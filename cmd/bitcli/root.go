// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"os"
	"path/filepath"

	"github.com/bitcli/bitcli/internal/setup"
	"github.com/spf13/cobra"
)

type rootOptions struct {
	configPath string
	verbose    bool
	version    string
	commit     string
	date       string
}

func newRootCommand(version, commit, date string) *cobra.Command {
	opts := &rootOptions{version: version, commit: commit, date: date}
	cmd := &cobra.Command{
		Use:          "bitcli",
		Short:        "An Ollama-like runtime and model manager for Microsoft BitNet models",
		SilenceUsage:  true,
		SilenceErrors: true,
		// Prepend ~/.bitcli/tools/*/bin to PATH so bundled cmake, clang, and uv
		// are always found without the user needing to source env.ps1 / env.sh.
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err == nil {
				setup.InjectPATH(filepath.Join(home, ".bitcli"))
			}
			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&opts.configPath, "config", "", "Path to a BitCLI config file")
	cmd.PersistentFlags().BoolVarP(&opts.verbose, "verbose", "v", false, "Enable debug logging")

	cmd.AddCommand(newPullCommand(opts))
	cmd.AddCommand(newRunCommand(opts))
	cmd.AddCommand(newChatCommand(opts))
	cmd.AddCommand(newServeCommand(opts))
	cmd.AddCommand(newListCommand(opts))
	cmd.AddCommand(newRemoveCommand(opts))
	cmd.AddCommand(newDoctorCommand(opts))
	cmd.AddCommand(newVersionCommand(opts))
	cmd.AddCommand(newConfigCommand(opts))
	cmd.AddCommand(newUpdateCommand(opts))
	cmd.AddCommand(newSetupCommand(opts))
	return cmd
}

