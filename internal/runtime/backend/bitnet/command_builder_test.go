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
	if !strings.Contains(argsStr, "--no-display-prompt") {
		t.Fatalf("expected --no-display-prompt in args, got %s", argsStr)
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
	// Should inject default system prompt when no system message present.
	if !strings.Contains(prompt, "<|start_header_id|>system<|end_header_id|>") {
		t.Fatalf("expected default system prompt, got: %q", prompt)
	}
	if !strings.Contains(prompt, defaultSystemPrompt) {
		t.Fatalf("expected default system prompt content, got: %q", prompt)
	}
	if !strings.Contains(prompt, "Hello") {
		t.Fatalf("expected user message in prompt, got: %q", prompt)
	}
	if !strings.HasSuffix(prompt, "<|start_header_id|>assistant<|end_header_id|>\n\n") {
		t.Fatalf("expected assistant header at end, got: %q", prompt)
	}
}

func TestPromptFromMessagesWithSystemMessage(t *testing.T) {
	prompt := PromptFromMessages([]backend.Message{
		{Role: "system", Content: "Be brief."},
		{Role: "user", Content: "Hello"},
	})
	// Should NOT inject default system prompt when one is already present.
	expected := "<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n\nBe brief.<|eot_id|><|start_header_id|>user<|end_header_id|>\n\nHello<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n"
	if prompt != expected {
		t.Fatalf("unexpected prompt:\n got: %q\nwant: %q", prompt, expected)
	}
	// Must not contain default system prompt
	if strings.Contains(prompt, defaultSystemPrompt) {
		t.Fatalf("should not inject default system prompt when custom system message exists")
	}
}

func TestGenerateCommandMinP(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)
	m := model.Model{Path: filepath.Join("models", "ggml-model-i2_s.gguf")}
	cmd := builder.GenerateCommand(m, backend.GenerateRequest{
		Prompt: "hello",
		Options: backend.Options{
			MinP: 0.05,
		},
	}, false)

	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "--min-p 0.0500") {
		t.Fatalf("expected --min-p in args, got %s", argsStr)
	}
}

func TestGenerateCommandSystemPromptInjected(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)
	m := model.Model{Path: filepath.Join("models", "ggml-model-i2_s.gguf")}
	cmd := builder.GenerateCommand(m, backend.GenerateRequest{
		Prompt: "Hello",
	}, false)

	// Find the -p argument
	var prompt string
	for i, arg := range cmd.Args {
		if arg == "-p" && i+1 < len(cmd.Args) {
			prompt = cmd.Args[i+1]
			break
		}
	}
	if !strings.Contains(prompt, defaultSystemPrompt) {
		t.Fatalf("expected default system prompt in -p arg, got: %q", prompt)
	}
	if !strings.Contains(prompt, "Hello") {
		t.Fatalf("expected user prompt in -p arg, got: %q", prompt)
	}
}
