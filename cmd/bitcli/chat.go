// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	bitruntime "github.com/bitcli/bitcli/internal/runtime"
	"github.com/spf13/cobra"
)

func newChatCommand(opts *rootOptions) *cobra.Command {
	var modelID string
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Start or manage local BitCLI chat history",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			return runChat(cmd, a, modelID)
		},
	}
	cmd.Flags().StringVarP(&modelID, "model", "m", "", "Model to chat with")
	cmd.AddCommand(newChatHistoryCommand(opts))
	cmd.AddCommand(newChatDeleteCommand(opts))
	cmd.AddCommand(&cobra.Command{
		Use:   "new",
		Short: "Start a new chat session",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			return runChat(cmd, a, modelID)
		},
	})
	return cmd
}

func runChat(cmd *cobra.Command, a *app, modelID string) error {
	m, err := a.ensureModel(cmd.Context(), modelID, os.Stdout)
	if err != nil {
		return err
	}
	session, err := a.chats.CreateSession(cmd.Context(), "New chat", m.ID)
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Chatting with %s. Type /exit to leave.\n", m.UserID)
	reader := bufio.NewScanner(cmd.InOrStdin())
	messages := make([]bitruntime.Message, 0, 16)
	for {
		fmt.Fprint(cmd.OutOrStdout(), "> ")
		if !reader.Scan() {
			break
		}
		text := strings.TrimSpace(reader.Text())
		if text == "" {
			continue
		}
		if text == "/exit" || text == "/quit" {
			break
		}
		_, _ = a.chats.AddMessage(cmd.Context(), session.ID, "user", text, 0)
		messages = append(messages, bitruntime.Message{Role: "user", Content: text})
		events, errs := a.runtime.Chat(cmd.Context(), bitruntime.ChatRequest{
			ModelID:  m.UserID,
			Messages: messages,
			Options:  runtimeOptions(a.cfg),
		})
		var response strings.Builder
		for ev := range events {
			if ev.Text != "" {
				response.WriteString(ev.Text)
				fmt.Fprint(cmd.OutOrStdout(), ev.Text)
			}
		}
		if err := <-errs; err != nil {
			return err
		}
		answer := strings.TrimSpace(response.String())
		_, _ = a.chats.AddMessage(cmd.Context(), session.ID, "assistant", answer, 0)
		messages = append(messages, bitruntime.Message{Role: "assistant", Content: answer})
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return reader.Err()
}

func newChatHistoryCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "history",
		Short: "List saved chat sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			sessions, err := a.chats.ListSessions(cmd.Context())
			if err != nil {
				return err
			}
			for _, s := range sessions {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", s.ID, s.ModelID, s.UpdatedAt.Format("2006-01-02 15:04:05"))
			}
			return nil
		},
	}
}

func newChatDeleteCommand(opts *rootOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "delete SESSION_ID",
		Short: "Delete a saved chat session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, cleanup, err := newApp(cmd, opts)
			if err != nil {
				return err
			}
			defer cleanup()
			return a.chats.DeleteSession(cmd.Context(), args[0])
		},
	}
}

