// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/spf13/cobra"
)

func newConfigCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and update BitCLI configuration",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "path",
		Short: "Print the active config path",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := config.NewService()
			if err != nil {
				return err
			}
			_, paths, err := svc.Load(opts.configPath)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), paths.ConfigFile)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Create a default config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := config.NewService()
			if err != nil {
				return err
			}
			cfg, paths, err := svc.Load(opts.configPath)
			if err != nil {
				return err
			}
			if err := svc.Write(paths.ConfigFile, cfg); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", paths.ConfigFile)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "get KEY",
		Short: "Print a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := config.NewService()
			if err != nil {
				return err
			}
			value, err := svc.Get(opts.configPath, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%v\n", value)
			return nil
		},
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "set KEY VALUE",
		Short: "Set a config value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := config.NewService()
			if err != nil {
				return err
			}
			value := parseConfigValue(args[1])
			if err := svc.Set(opts.configPath, args[0], value); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "set %s\n", args[0])
			return nil
		},
	})
	return cmd
}

func parseConfigValue(raw string) any {
	if i, err := strconv.Atoi(raw); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(raw, 64); err == nil {
		return f
	}
	if b, err := strconv.ParseBool(raw); err == nil {
		return b
	}
	if strings.Contains(raw, ",") {
		parts := strings.Split(raw, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		return parts
	}
	return raw
}

