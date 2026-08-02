// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import "github.com/spf13/cobra"

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
	return cmd
}

