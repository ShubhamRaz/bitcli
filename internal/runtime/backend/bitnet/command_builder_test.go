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

func TestGenerateCommandFormatting(t *testing.T) {
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
	}, false)

	if !strings.Contains(cmd.Name, "llama-cli") {
		t.Fatalf("expected command name to contain llama-cli, got %s", cmd.Name)
	}
	argsStr := strings.Join(cmd.Args, " ")

	// Must NOT use -cnv and -st
	if strings.Contains(argsStr, "-cnv") {
		t.Fatalf("unexpected -cnv in args, got %s", argsStr)
	}
	if strings.Contains(argsStr, "-st") {
		t.Fatalf("unexpected -st in args, got %s", argsStr)
	}

	// -p should be the user message formatted as one-shot
	var pArg string
	for i, arg := range cmd.Args {
		if arg == "-p" && i+1 < len(cmd.Args) {
			pArg = cmd.Args[i+1]
			break
		}
	}
	expectedPrompt := "User: hello<|eot_id|>Assistant: "
	if pArg != expectedPrompt {
		t.Fatalf("expected user message %q in -p, got %q", expectedPrompt, pArg)
	}

	if !strings.Contains(argsStr, "-ngl 99") {
		t.Fatalf("expected -ngl 99 in args, got %s", argsStr)
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
	expected := "User: Hello<|eot_id|>Assistant: "
	if prompt != expected {
		t.Fatalf("expected prompt:\n got: %q\nwant: %q", prompt, expected)
	}
}

func TestPromptFromMessagesWithSystemMessage(t *testing.T) {
	prompt := PromptFromMessages([]backend.Message{
		{Role: "system", Content: "Be brief."},
		{Role: "user", Content: "Hello"},
	})
	expected := "System: Be brief.<|eot_id|>User: Hello<|eot_id|>Assistant: "
	if prompt != expected {
		t.Fatalf("unexpected prompt:\n got: %q\nwant: %q", prompt, expected)
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

func TestGenerateCommandSysAndPromptSeparation(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)
	m := model.Model{Path: filepath.Join("models", "ggml-model-i2_s.gguf")}
	cmd := builder.GenerateCommand(m, backend.GenerateRequest{
		Prompt: "User: What is the capital of France?<|eot_id|>Assistant: ",
	}, false)

	// -p should be exactly what was passed if it already contains <|eot_id|>
	var pArg string
	for i, arg := range cmd.Args {
		if arg == "-p" && i+1 < len(cmd.Args) {
			pArg = cmd.Args[i+1]
			break
		}
	}
	if pArg != "User: What is the capital of France?<|eot_id|>Assistant: " {
		t.Fatalf("-p should contain exact user message, got %q", pArg)
	}
}

func TestPromptFromMessagesJinja(t *testing.T) {
	prompt := PromptFromMessagesJinja([]backend.Message{
		{Role: "user", Content: "Hello"},
	})
	expected := "<|begin_of_text|><|start_header_id|>user<|end_header_id|>\n\nHello<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n"
	if prompt != expected {
		t.Fatalf("expected Jinja prompt:\n got:  %q\nwant: %q", prompt, expected)
	}
}

func TestPromptFromMessagesJinjaWithSystem(t *testing.T) {
	prompt := PromptFromMessagesJinja([]backend.Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hi"},
	})
	expected := "<|begin_of_text|>" +
		"<|start_header_id|>system<|end_header_id|>\n\nYou are helpful.<|eot_id|>" +
		"<|start_header_id|>user<|end_header_id|>\n\nHi<|eot_id|>" +
		"<|start_header_id|>assistant<|end_header_id|>\n\n"
	if prompt != expected {
		t.Fatalf("unexpected Jinja prompt:\n got:  %q\nwant: %q", prompt, expected)
	}
}

func TestGenerateCommandJinjaFlag(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)
	m := model.Model{
		Path:     filepath.Join("models", "ggml-model-i2_s.gguf"),
		ChatMode: model.ChatModeJinja,
	}
	cmd := builder.GenerateCommand(m, backend.GenerateRequest{Prompt: "hello"}, false)
	argsStr := strings.Join(cmd.Args, " ")
	if !strings.Contains(argsStr, "--jinja") {
		t.Fatalf("expected --jinja in args for chat model, got: %s", argsStr)
	}
}

func TestGenerateCommandNoJinjaForBaseModel(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)
	m := model.Model{
		Path:     filepath.Join("models", "ggml-model-i2_s.gguf"),
		ChatMode: model.ChatModePlain,
	}
	cmd := builder.GenerateCommand(m, backend.GenerateRequest{Prompt: "hello"}, false)
	argsStr := strings.Join(cmd.Args, " ")
	if strings.Contains(argsStr, "--jinja") {
		t.Fatalf("expected no --jinja in args for base model, got: %s", argsStr)
	}
}
