// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"fmt"
	"strings"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/spf13/cobra"
)

func newDeviceCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device [cpu|gpu]",
		Short: "View or switch compute device (CPU or GPU) for model inference",
		Long: `View or switch the compute device used for BitNet model inference.

Usage:
  bitcli device         # Print current device setting
  bitcli device gpu     # Switch to GPU acceleration (CUDA / Metal / Vulkan)
  bitcli device cpu     # Switch to CPU inference`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := config.NewService()
			if err != nil {
				return err
			}
			cfg, paths, err := svc.Load(opts.configPath)
			if err != nil {
				return err
			}

			// If no args provided, print current device
			if len(args) == 0 {
				dev := cfg.Runtime.Device
				if dev == "" {
					dev = "cpu"
				}
				gpuLayers := cfg.Runtime.GPULayers
				if strings.EqualFold(dev, "gpu") {
					fmt.Fprintf(cmd.OutOrStdout(), "Compute device: gpu (layers offloaded to GPU)\n")
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Compute device: cpu (gpu_layers: %d, threads: %d)\n", gpuLayers, cfg.Runtime.Threads)
				}
				return nil
			}

			// Setting device
			target := strings.ToLower(strings.TrimSpace(args[0]))
			if target != "cpu" && target != "gpu" {
				return fmt.Errorf("invalid device %q: must be 'cpu' or 'gpu'", args[0])
			}

			if err := svc.Set(opts.configPath, "runtime.device", target); err != nil {
				return err
			}

			if target == "gpu" {
				fmt.Fprintln(cmd.OutOrStdout(), "✓ Switched compute device to: GPU (hardware acceleration enabled)")
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "✓ Switched compute device to: CPU")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated config: %s\n", paths.ConfigFile)
			return nil
		},
	}
	return cmd
}
