// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	bitruntime "github.com/bitcli/bitcli/internal/runtime"
	"github.com/spf13/cobra"
)

func newRunCommand(opts *rootOptions) *cobra.Command {
	var prompt string
	var modelID string
	var runtimeOpts bitruntime.Options
	cmd := &cobra.Command{
		Use:   "run [MODEL]",
		Short: "Run a prompt against a local or automatically downloaded BitNet model",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			if len(args) > 0 {
				modelID = args[0]
			}
			if modelID == "" {
				modelID = a.cfg.DefaultModel
			}
			if prompt == "" {
				prompt, err = promptFromInput(cmd.InOrStdin(), cmd.OutOrStdout())
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(prompt) == "" {
				return fmt.Errorf("prompt is required; pass --prompt or type a prompt")
			}
			m, err := a.ensureModel(cmd.Context(), modelID, os.Stdout)
			if err != nil {
				return err
			}
			merged := mergeOptions(runtimeOptions(a.cfg), runtimeOpts)
			events, errs := a.runtime.Generate(cmd.Context(), bitruntime.GenerateRequest{ModelID: m.UserID, Prompt: prompt, Options: merged})
			for ev := range events {
				if ev.Text != "" {
					fmt.Fprint(cmd.OutOrStdout(), ev.Text)
				}
			}
			if err := <-errs; err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout())
			return nil
		},
	}
	cmd.Flags().StringVarP(&prompt, "prompt", "p", "", "Prompt to generate from")
	cmd.Flags().Float64Var(&runtimeOpts.Temperature, "temperature", 0, "Sampling temperature")
	cmd.Flags().Float64Var(&runtimeOpts.TopP, "top-p", 0, "Nucleus sampling value")
	cmd.Flags().IntVar(&runtimeOpts.TopK, "top-k", 0, "Top-k sampling value")
	cmd.Flags().IntVarP(&runtimeOpts.Threads, "threads", "t", 0, "CPU threads")
	cmd.Flags().IntVar(&runtimeOpts.GPULayers, "gpu-layers", 0, "GPU layers when supported by the backend")
	cmd.Flags().IntVarP(&runtimeOpts.ContextLength, "context", "c", 0, "Context length")
	cmd.Flags().IntVarP(&runtimeOpts.MaxTokens, "max-tokens", "n", 0, "Maximum generated tokens")
	return cmd
}

func promptFromInput(in io.Reader, out io.Writer) (string, error) {
	if f, ok := in.(*os.File); ok {
		info, err := f.Stat()
		if err == nil && info.Mode()&os.ModeCharDevice == 0 {
			data, err := io.ReadAll(in)
			return string(data), err
		}
	}
	fmt.Fprint(out, ">>> ")
	text, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(text), nil
}

func mergeOptions(base, override bitruntime.Options) bitruntime.Options {
	if override.Temperature != 0 {
		base.Temperature = override.Temperature
	}
	if override.TopP != 0 {
		base.TopP = override.TopP
	}
	if override.TopK != 0 {
		base.TopK = override.TopK
	}
	if override.Threads != 0 {
		base.Threads = override.Threads
	}
	if override.GPULayers != 0 {
		base.GPULayers = override.GPULayers
	}
	if override.ContextLength != 0 {
		base.ContextLength = override.ContextLength
	}
	if override.MaxTokens != 0 {
		base.MaxTokens = override.MaxTokens
	}
	return base
}

