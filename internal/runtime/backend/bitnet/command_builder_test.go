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

func TestGenerateCommandConversationMode(t *testing.T) {
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

	// Must use -cnv and -st for conversation mode
	if !strings.Contains(argsStr, "-cnv") {
		t.Fatalf("expected -cnv in args, got %s", argsStr)
	}
	if !strings.Contains(argsStr, "-st") {
		t.Fatalf("expected -st in args, got %s", argsStr)
	}

	// -sys should be the system prompt
	var sysArg string
	for i, arg := range cmd.Args {
		if arg == "-sys" && i+1 < len(cmd.Args) {
			sysArg = cmd.Args[i+1]
			break
		}
	}
	if !strings.Contains(sysArg, "helpful assistant") {
		t.Fatalf("expected system prompt in -sys, got %q", sysArg)
	}

	// -p should be the user message
	var pArg string
	for i, arg := range cmd.Args {
		if arg == "-p" && i+1 < len(cmd.Args) {
			pArg = cmd.Args[i+1]
			break
		}
	}
	if pArg != "hello" {
		t.Fatalf("expected user message 'hello' in -p, got %q", pArg)
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

func TestGenerateCommandSysAndPromptSeparation(t *testing.T) {
	cfg := config.DefaultConfig()
	paths := config.Paths{BackendDir: filepath.Join("tmp", "backends")}
	builder := NewBuilder(cfg, paths)
	m := model.Model{Path: filepath.Join("models", "ggml-model-i2_s.gguf")}
	cmd := builder.GenerateCommand(m, backend.GenerateRequest{
		Prompt: "What is the capital of France?",
	}, false)

	// -sys should be the system prompt
	var sysArg string
	for i, arg := range cmd.Args {
		if arg == "-sys" && i+1 < len(cmd.Args) {
			sysArg = cmd.Args[i+1]
			break
		}
	}
	if !strings.Contains(sysArg, "helpful assistant") {
		t.Fatalf("-sys should contain system prompt, got %q", sysArg)
	}

	// -p should be the user message, NOT the system prompt
	var pArg string
	for i, arg := range cmd.Args {
		if arg == "-p" && i+1 < len(cmd.Args) {
			pArg = cmd.Args[i+1]
			break
		}
	}
	if pArg != "What is the capital of France?" {
		t.Fatalf("-p should contain user message, got %q", pArg)
	}

	// No Stdin needed since we use -p for user message
	if len(cmd.Stdin) != 0 {
		t.Fatalf("expected no Stdin data, got %q", string(cmd.Stdin))
	}
}

func TestExtractFromChatTemplate(t *testing.T) {
	template := "<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n\nBe brief.<|eot_id|><|start_header_id|>user<|end_header_id|>\n\nHello there<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n"
	sys, user := extractFromChatTemplate(template)
	if sys != "Be brief." {
		t.Fatalf("expected system 'Be brief.', got %q", sys)
	}
	if user != "Hello there" {
		t.Fatalf("expected user 'Hello there', got %q", user)
	}
}

func TestExtractFromChatTemplateMultiTurn(t *testing.T) {
	prompt := PromptFromMessages([]backend.Message{
		{Role: "system", Content: "Be helpful."},
		{Role: "user", Content: "Hi"},
		{Role: "assistant", Content: "Hello!"},
		{Role: "user", Content: "Tell me about dogs"},
	})
	sys, user := extractFromChatTemplate(prompt)
	if sys != "Be helpful." {
		t.Fatalf("expected system 'Be helpful.', got %q", sys)
	}
	// Should get the LAST user message
	if user != "Tell me about dogs" {
		t.Fatalf("expected last user message 'Tell me about dogs', got %q", user)
	}
}
