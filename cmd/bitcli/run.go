// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bitcli/bitcli/internal/model"
	bitruntime "github.com/bitcli/bitcli/internal/runtime"
	"github.com/spf13/cobra"
)

func newRunCommand(opts *rootOptions) *cobra.Command {
	var prompt string
	var modelID string
	var runtimeOpts bitruntime.Options
	cmd := &cobra.Command{
		Use:   "run [MODEL] [PROMPT...]",
		Short: "Run a prompt or start an interactive chat session with a BitNet model",
		Args:  cobra.ArbitraryArgs,
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

			if prompt == "" && len(args) > 1 {
				prompt = strings.Join(args[1:], " ")
			}

			// If input is piped (e.g. echo "hello" | bitcli run ...)
			if prompt == "" && isPipedInput(cmd.InOrStdin()) {
				data, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				prompt = strings.TrimSpace(string(data))
			}

			m, err := a.ensureModel(cmd.Context(), modelID, os.Stdout)
			if err != nil {
				return err
			}

			// If a prompt was provided, run single generation and exit
			if strings.TrimSpace(prompt) != "" {
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
			}

			// Otherwise, launch clean interactive multi-turn REPL
			return runInteractiveREPL(cmd, a, m, runtimeOpts)
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

func runInteractiveREPL(cmd *cobra.Command, a *app, m model.Model, runtimeOpts bitruntime.Options) error {
	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()
	reader := bufio.NewScanner(in)
	messages := make([]bitruntime.Message, 0, 16)
	mergedOpts := mergeOptions(runtimeOptions(a.cfg), runtimeOpts)

	session, err := a.chats.CreateSession(cmd.Context(), "Interactive session", m.ID)
	if err != nil {
		return err
	}

	for {
		fmt.Fprint(out, ">>> ")
		if !reader.Scan() {
			break
		}
		text := strings.TrimSpace(reader.Text())
		if text == "" {
			continue
		}
		if text == "/exit" || text == "/quit" || text == "/bye" || text == ":q" {
			break
		}
		if text == "/clear" {
			messages = messages[:0]
			fmt.Fprintln(out, "Cleared conversation history.")
			continue
		}
		if text == "/help" || text == "/?" {
			fmt.Fprintln(out, "Available commands:")
			fmt.Fprintln(out, "  /exit, /bye, /quit   Exit interactive session")
			fmt.Fprintln(out, "  /clear               Clear conversation history")
			fmt.Fprintln(out, "  /help, /?            Show help commands")
			continue
		}

		_, _ = a.chats.AddMessage(cmd.Context(), session.ID, "user", text, 0)
		messages = append(messages, bitruntime.Message{Role: "user", Content: text})

		events, errs := a.runtime.Chat(cmd.Context(), bitruntime.ChatRequest{
			ModelID:  m.UserID,
			Messages: messages,
			Options:  mergedOpts,
		})

		var response strings.Builder
		for ev := range events {
			if ev.Text != "" {
				response.WriteString(ev.Text)
				fmt.Fprint(out, ev.Text)
			}
		}
		if err := <-errs; err != nil {
			fmt.Fprintf(out, "\n[Error: %v]\n", err)
			return err
		}
		answer := strings.TrimSpace(response.String())
		_, _ = a.chats.AddMessage(cmd.Context(), session.ID, "assistant", answer, 0)
		messages = append(messages, bitruntime.Message{Role: "assistant", Content: answer})
		fmt.Fprintln(out)
	}
	return reader.Err()
}

func isPipedInput(in io.Reader) bool {
	if f, ok := in.(*os.File); ok {
		info, err := f.Stat()
		if err == nil && (info.Mode()&os.ModeCharDevice == 0) {
			return true
		}
	}
	return false
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
