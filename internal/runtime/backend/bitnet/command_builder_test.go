// Package bitnet tests the official bitnet.cpp command adapter.
package bitnet

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/runtime/backend"
)

func TestGenerateCommandDirectLlamaCLI(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)
	m := model.Model{Path: filepath.Join("models", "ggml-model-i2_s.gguf")}
	cmd := builder.GenerateCommand(m, backend.GenerateRequest{
		Prompt: "hello",
		Options: backend.Options{
			Temperature:   0.7,
			Threads:       4,
			ContextLength: 4096,
			MaxTokens:     32,
			TopP:          0.9,
			GPULayers:     99,
		},
	}, true)

	if !strings.Contains(cmd.Name, "llama-cli") {
		t.Fatalf("expected command name to contain llama-cli, got %s", cmd.Name)
	}
	// Verify flags
	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "-m") || !strings.Contains(argsStr, "-p hello") {
		t.Fatalf("expected model and prompt in args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "-ngl 99") {
		t.Fatalf("expected -ngl 99 in args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "-cnv") {
		t.Fatalf("expected -cnv in args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "-st") {
		t.Fatalf("expected -st in args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "--repeat-penalty") {
		t.Fatalf("expected --repeat-penalty in args, got %s", argsStr)
	}
}

func TestBuildCommands(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)

	codegen := builder.CodegenCommand()
	if codegen.Name != "python" || len(codegen.Args) == 0 || codegen.Args[0] != "utils/codegen_tl2.py" {
		t.Fatalf("unexpected codegen command: %#v", codegen)
	}

	cmakeConf := builder.CMakeConfigureCommand()
	if cmakeConf.Name != "cmake" || cmakeConf.Args[0] != "-B" {
		t.Fatalf("unexpected cmake configure command: %#v", cmakeConf)
	}

	cmakeBuild := builder.CMakeBuildCommand()
	if cmakeBuild.Name != "cmake" || cmakeBuild.Args[0] != "--build" {
		t.Fatalf("unexpected cmake build command: %#v", cmakeBuild)
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

