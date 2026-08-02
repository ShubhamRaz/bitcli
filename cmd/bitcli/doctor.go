// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/bitcli/bitcli/internal/runtime/backend/bitnet"
	"github.com/spf13/cobra"
)

func newDoctorCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check BitCLI, bitnet.cpp, hardware, cache, and configuration health",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			ctx, cancel := context.WithTimeout(cmd.Context(), 10*time.Second)
			defer cancel()

			fmt.Fprintln(cmd.OutOrStdout(), "BitCLI doctor")
			fmt.Fprintf(cmd.OutOrStdout(), "config: %s\n", a.paths.ConfigFile)
			fmt.Fprintf(cmd.OutOrStdout(), "database: %s\n", a.paths.DatabaseFile)
			fmt.Fprintf(cmd.OutOrStdout(), "go runtime: %s\n", runtime.Version())

			checkTool(cmd, "python")
			checkTool(cmd, "cmake")
			checkTool(cmd, "clang")
			checkTool(cmd, "git")

			status, err := bitnet.New(a.cfg, a.paths, a.runner, a.log).Detect(ctx)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "bitnet.cpp: error: %v\n", err)
			} else if status.Ready {
				fmt.Fprintf(cmd.OutOrStdout(), "bitnet.cpp: ok (%s)\n", status.Path)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "bitnet.cpp: missing (%s)\n", status.Message)
			}

			report := a.hardware.Report(ctx)
			fmt.Fprintf(cmd.OutOrStdout(), "cpu: %s (%d threads)\n", report.CPUName, report.CPUThreads)
			fmt.Fprintf(cmd.OutOrStdout(), "instructions: avx2=%t avx512=%t\n", report.AVX2, report.AVX512)
			fmt.Fprintf(cmd.OutOrStdout(), "accelerators: cuda=%t metal=%t rocm=%t\n", report.CUDA, report.Metal, report.ROCm)
			fmt.Fprintf(cmd.OutOrStdout(), "ram: %.1f GiB\n", float64(report.RAMBytes)/(1024*1024*1024))
			fmt.Fprintf(cmd.OutOrStdout(), "recommended model: %s\n", report.RecommendedModel)
			fmt.Fprintf(cmd.OutOrStdout(), "estimated tokens/sec: %s\n", report.EstimatedTokSec)
			for _, warning := range report.Warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", warning)
			}

			if err := a.cache.Ensure(); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "model cache: error: %v\n", err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "model cache: %s\n", a.paths.ModelDir)
			}
			if _, err := os.Stat(a.paths.ConfigFile); err == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "configuration: ok")
			}
			return nil
		},
	}
}

func checkTool(cmd *cobra.Command, name string) {
	path, err := exec.LookPath(name)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "%s: missing\n", name)
		return
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", name, path)
}

