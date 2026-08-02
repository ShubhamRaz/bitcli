// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitcli/bitcli/internal/runtime/backend/bitnet"
	"github.com/bitcli/bitcli/internal/setup"
	"github.com/spf13/cobra"
)

func newSetupCommand(opts *rootOptions) *cobra.Command {
	var skipBackend bool
	var forceReinstall bool

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "One-time setup: install dependencies and initialize the BitNet backend",
		Long: `bitcli setup downloads all required tools (cmake, clang, uv) into
~/.bitcli/tools/ and clones the official Microsoft BitNet backend.

Everything is stored inside ~/.bitcli/. Deleting that directory completely
removes BitCLI and all its dependencies — nothing is written to system paths.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()

			out := cmd.OutOrStdout()
			home := a.paths.Root

			fmt.Fprintln(out, "BitCLI Setup")
			fmt.Fprintln(out, "============")
			fmt.Fprintf(out, "Install directory: %s\n\n", home)

			// ── Step 1: Detect existing tools ───────────────────────────────
			fmt.Fprintln(out, "Step 1/4  Checking installed tools...")
			detector := setup.NewDetector(home)
			report := detector.Detect()

			printToolStatus(out, "git", report.Git)
			printToolStatus(out, "cmake", report.CMake)
			printToolStatus(out, "clang", report.Clang)
			printToolStatus(out, "uv", report.UV)
			fmt.Fprintln(out)

			// ── Step 2: Install missing tools ────────────────────────────────
			missing := report.Missing()
			if len(missing) > 0 {
				fmt.Fprintf(out, "Step 2/4  Installing %d missing tool(s)...\n", len(missing))
				installer := setup.NewToolInstaller(home, out)
				for _, tool := range missing {
					if tool.Name == "git" {
						fmt.Fprintln(out, "  ✗ git is required but not installed.")
						fmt.Fprintln(out, "    Windows: https://git-scm.com/download/win")
						fmt.Fprintln(out, "    Linux:   sudo apt install git  |  brew install git")
						return fmt.Errorf("git is required; please install it and re-run bitcli setup")
					}
					if forceReinstall {
						// Wipe existing partial install so it re-downloads.
						_ = removeToolDir(home, tool.Name)
					}
					if _, err := installer.Install(cmd.Context(), tool.Name); err != nil {
						return fmt.Errorf("failed to install %s: %w", tool.Name, err)
					}
				}
			} else {
				fmt.Fprintln(out, "Step 2/4  All tools already present — skipping download.")
			}
			fmt.Fprintln(out)

			// ── Step 3: Write env activation scripts ─────────────────────────
			fmt.Fprintln(out, "Step 3/4  Writing shell activation files...")
			envWriter := setup.NewEnvWriter(home)
			if err := envWriter.Write(); err != nil {
				return fmt.Errorf("write env files: %w", err)
			}
			fmt.Fprintf(out, "  ✓ %s/env.ps1 written\n", home)
			fmt.Fprintf(out, "  ✓ %s/env.sh written\n", home)
			fmt.Fprintln(out)

			// ── Step 4: Clone / update BitNet backend ────────────────────────
			if skipBackend {
				fmt.Fprintln(out, "Step 4/4  Skipping BitNet backend (--skip-backend).")
			} else {
				fmt.Fprintln(out, "Step 4/4  Setting up Microsoft BitNet backend...")
				fmt.Fprintln(out, "  (This clones https://github.com/microsoft/BitNet — may take a few minutes)")
				target := filepath.Join(a.paths.BackendDir, "bitnet", "current")
				bi := bitnet.Installer{Runner: a.runner}
				if err := bi.EnsureClone(context.Background(), a.cfg.Backend.BitNet.RepoURL, target); err != nil {
					return fmt.Errorf("clone BitNet backend: %w", err)
				}
				if err := bi.CheckoutRevision(context.Background(), target, a.cfg.Backend.BitNet.Revision); err != nil {
					return fmt.Errorf("checkout BitNet revision: %w", err)
				}
				fmt.Fprintf(out, "  ✓ BitNet backend ready at %s\n\n", target)
			}

			// ── Done ─────────────────────────────────────────────────────────
			fmt.Fprintln(out, "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			fmt.Fprintln(out, "✓ BitCLI setup complete!")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Next steps:")
			fmt.Fprintln(out, "  bitcli doctor                                    # verify everything")
			fmt.Fprintln(out, "  bitcli pull microsoft/BitNet-b1.58-2B-4T         # download model weights")
			fmt.Fprintln(out, "  bitcli run --prompt \"Hello!\"                     # run a prompt")
			fmt.Fprintln(out, "  bitcli chat                                      # interactive chat")
			fmt.Fprintln(out, "  bitcli serve                                     # start API server")
			fmt.Fprintln(out)

			setup.PrintActivationHint(out, home)

			return nil
		},
	}

	cmd.Flags().BoolVar(&skipBackend, "skip-backend", false, "Skip cloning the Microsoft BitNet backend")
	cmd.Flags().BoolVar(&forceReinstall, "force", false, "Re-download and reinstall portable tools even if present")
	return cmd
}

func printToolStatus(out interface{ Write([]byte) (int, error) }, label string, s setup.ToolStatus) {
	switch {
	case s.Missing:
		fmt.Fprintf(out, "  ✗ %-8s missing\n", label)
	case s.Bundled:
		fmt.Fprintf(out, "  ✓ %-8s %s  (bundled)\n", label, s.Path)
	default:
		fmt.Fprintf(out, "  ✓ %-8s %s\n", label, s.Path)
	}
}

func removeToolDir(home, toolName string) error {
	return os.RemoveAll(filepath.Join(home, "tools", toolName))
}
