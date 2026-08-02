// Package bitnet tests the official bitnet.cpp command adapter.
package bitnet

import (
	"path/filepath"
	"testing"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/runtime/backend"
)

func TestGenerateCommandUsesOfficialRunInferenceScript(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)
	m := model.Model{Path: filepath.Join("models", "ggml-model-i2_s.gguf")}
	cmd, unsupported := builder.GenerateCommand(m, backend.GenerateRequest{
		Prompt: "hello",
		Options: backend.Options{
			Temperature:   0.7,
			Threads:       4,
			ContextLength: 4096,
			MaxTokens:     32,
			TopP:          0.9,
		},
	}, true)

	if cmd.Name != "python" {
		t.Fatalf("unexpected command name: %s", cmd.Name)
	}
	if cmd.Args[0] != "run_inference.py" {
		t.Fatalf("expected run_inference.py, got %s", cmd.Args[0])
	}
	if len(unsupported) != 0 {
		t.Fatalf("unexpected unsupported options: %#v", unsupported)
	}
}

func TestPromptFromMessages(t *testing.T) {
	prompt := PromptFromMessages([]backend.Message{{Role: "user", Content: "Hello"}})
	expected := "<|begin_of_text|>System: " + defaultSystemPrompt + "<|eot_id|>\nUser: Hello<|eot_id|>\nAssistant:"
	if prompt != expected {
		t.Fatalf("unexpected prompt:\n got: %q\nwant: %q", prompt, expected)
	}
}

func TestPromptFromMessagesWithSystemMessage(t *testing.T) {
	prompt := PromptFromMessages([]backend.Message{
		{Role: "system", Content: "Be brief."},
		{Role: "user", Content: "Hello"},
	})
	expected := "<|begin_of_text|>System: Be brief.<|eot_id|>\nUser: Hello<|eot_id|>\nAssistant:"
	if prompt != expected {
		t.Fatalf("unexpected prompt:\n got: %q\nwant: %q", prompt, expected)
	}
}

