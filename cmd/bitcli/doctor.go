// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()

			out := cmd.OutOrStdout()

			// ── Header ─────────────────────────────────────────────────────
			fmt.Fprintln(out)
			fmt.Fprintln(out, "  ╔══════════════════════════════════════════════════════╗")
			fmt.Fprintln(out, "  ║              BitCLI System Diagnostics               ║")
			fmt.Fprintln(out, "  ╚══════════════════════════════════════════════════════╝")
			fmt.Fprintln(out)

			// ── Configuration ──────────────────────────────────────────────
			printSection(out, "Configuration")
			printRow(out, "Config", a.paths.ConfigFile)
			printRow(out, "Database", a.paths.DatabaseFile)
			printRow(out, "Runtime", runtime.Version())
			fmt.Fprintln(out)

			// ── Toolchain ──────────────────────────────────────────────────
			printSection(out, "Toolchain")
			checkToolFancy(out, "Python", "python")
			checkToolFancy(out, "CMake", "cmake", filepath.Join(a.paths.Root, "tools", "cmake", "bin"), filepath.Join(a.paths.Root, "tools", "cmake"))
			checkToolFancy(out, "Ninja", "ninja", filepath.Join(a.paths.Root, "tools", "ninja"))
			checkToolFancy(out, "Clang", "clang", filepath.Join(a.paths.Root, "tools", "clang", "bin"), filepath.Join(a.paths.Root, "tools", "clang"))
			checkToolFancy(out, "Git", "git")
			fmt.Fprintln(out)

			// ── BitNet Backend ──────────────────────────────────────────────
			printSection(out, "BitNet Backend")
			status, err := bitnet.New(a.cfg, a.paths, a.runner, a.log).Detect(ctx)
			if err != nil {
				printStatus(out, "bitnet.cpp", false, fmt.Sprintf("error: %v", err))
			} else if status.Ready {
				printStatus(out, "bitnet.cpp", true, status.Path)
			} else {
				printStatus(out, "bitnet.cpp", false, status.Message)
				printHint(out, "Run: bitcli setup")
			}
			fmt.Fprintln(out)

			// ── Hardware ───────────────────────────────────────────────────
			printSection(out, "Hardware")
			report := a.hardware.Report(ctx)

			cpuLabel := report.CPUName
			if cpuLabel == "" || cpuLabel == "amd64" || cpuLabel == "arm64" {
				cpuLabel = runtime.GOARCH
			}
			printRow(out, "CPU", fmt.Sprintf("%s", cpuLabel))
			printRow(out, "Cores / Threads", fmt.Sprintf("%d threads", report.CPUThreads))
			printRow(out, "Architecture", report.Arch)

			// RAM
			ramGiB := float64(report.RAMBytes) / (1024 * 1024 * 1024)
			if ramGiB > 0.1 {
				printRow(out, "RAM", fmt.Sprintf("%.1f GiB", ramGiB))
			} else {
				printRow(out, "RAM", "could not detect")
			}
			fmt.Fprintln(out)

			// ── Instruction Sets ───────────────────────────────────────────
			printSection(out, "Instruction Sets")
			printFeature(out, "AVX2", report.AVX2)
			printFeature(out, "AVX-512", report.AVX512)
			fmt.Fprintln(out)

			// ── Accelerators ───────────────────────────────────────────────
			printSection(out, "Accelerators")
			printFeature(out, "CUDA (NVIDIA)", report.CUDA)
			printFeature(out, "Metal (Apple)", report.Metal)
			printFeature(out, "ROCm (AMD)", report.ROCm)
			if len(report.GPUs) > 0 {
				for _, gpu := range report.GPUs {
					vram := ""
					if gpu.VRAMBytes > 0 {
						vram = fmt.Sprintf(" (%.1f GiB VRAM)", float64(gpu.VRAMBytes)/(1024*1024*1024))
					}
					printRow(out, "  GPU", fmt.Sprintf("%s [%s]%s", gpu.Name, gpu.Backend, vram))
				}
			}
			fmt.Fprintln(out)

			// ── Model Recommendation ───────────────────────────────────────
			printSection(out, "Recommendation")
			printRow(out, "Model", report.RecommendedModel)
			printRow(out, "Performance", report.EstimatedTokSec)
			printRow(out, "Model Cache", a.paths.ModelDir)
			fmt.Fprintln(out)

			// ── Warnings ───────────────────────────────────────────────────
			if len(report.Warnings) > 0 {
				printSection(out, "Warnings")
				for _, w := range report.Warnings {
					fmt.Fprintf(out, "    ⚠  %s\n", w)
				}
				fmt.Fprintln(out)
			}

			// ── Cache ──────────────────────────────────────────────────────
			if err := a.cache.Ensure(); err != nil {
				fmt.Fprintf(out, "    ✗  Model cache error: %v\n\n", err)
			}

			// ── Overall Status ─────────────────────────────────────────────
			if _, err := os.Stat(a.paths.ConfigFile); err == nil {
				fmt.Fprintln(out, "  ──────────────────────────────────────────────────────")
				fmt.Fprintln(out, "  ✓  System check complete — configuration OK")
				fmt.Fprintln(out)
			}

			return nil
		},
	}
}

// ── Display Helpers ─────────────────────────────────────────────────────────

func printSection(out io.Writer, title string) {
	fmt.Fprintf(out, "  ── %s ──\n", title)
}

func printRow(out io.Writer, label, value string) {
	fmt.Fprintf(out, "    %-18s %s\n", label+":", value)
}

func printStatus(out io.Writer, label string, ok bool, detail string) {
	icon := "✓"
	if !ok {
		icon = "✗"
	}
	fmt.Fprintf(out, "    %s  %-14s %s\n", icon, label, detail)
}

func printFeature(out io.Writer, label string, supported bool) {
	if supported {
		fmt.Fprintf(out, "    ✓  %-18s supported\n", label)
	} else {
		fmt.Fprintf(out, "    ·  %-18s not detected\n", label)
	}
}

func printHint(out io.Writer, hint string) {
	fmt.Fprintf(out, "       → %s\n", hint)
}

func checkToolFancy(out io.Writer, display, name string, candidateDirs ...string) {
	// First check candidate dirs
	for _, dir := range candidateDirs {
		candidate := filepath.Join(dir, name)
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			short := candidate
			home, _ := os.UserHomeDir()
			if home != "" {
				short = strings.Replace(short, home, "~", 1)
			}
			printStatus(out, display, true, short)
			return
		}
	}
	path, err := exec.LookPath(name)
	if err != nil {
		printStatus(out, display, false, "missing")
		return
	}
	// Shorten path for readability.
	short := path
	home, _ := os.UserHomeDir()
	if home != "" {
		short = strings.Replace(short, home, "~", 1)
	}
	printStatus(out, display, true, short)
}
