// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/runtime/backend"
)



// Command describes an external process invocation without shell interpolation.
type Command struct {
	Dir   string
	Name  string
	Args  []string
	Stdin []byte // optional data to feed to the process stdin
}

// Builder creates official bitnet.cpp command invocations.
type Builder struct {
	cfg   config.Config
	paths config.Paths
}

// NewBuilder creates a BitNet command builder.
func NewBuilder(cfg config.Config, paths config.Paths) Builder {
	return Builder{cfg: cfg, paths: paths}
}

// BackendDir returns the configured or managed BitNet checkout path.
func (b Builder) BackendDir() string {
	if b.cfg.Backend.BitNet.Path != "" {
		return b.cfg.Backend.BitNet.Path
	}
	return filepath.Join(b.paths.BackendDir, "bitnet", "current")
}

// Python returns the configured Python executable or a platform-neutral default.
func (b Builder) Python() string {
	if b.cfg.Backend.BitNet.Python != "" {
		return b.cfg.Backend.BitNet.Python
	}
	return "python"
}

// LlamaCLIPath resolves the compiled llama-cli binary inside the backend build directory.
// On Windows it checks build/bin/Release/ first (MSVC/ClangCL layout), then build/bin/.
func (b Builder) LlamaCLIPath() string {
	backendDir := b.BackendDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	candidates := []string{
		filepath.Join(backendDir, "build", "bin", "Release", "llama-cli"+ext),
		filepath.Join(backendDir, "build", "bin", "llama-cli"+ext),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Return the most likely path even if not yet built.
	return candidates[0]
}

// SetupCommand builds the official setup_env.py invocation (kept as fallback).
func (b Builder) SetupCommand(m model.Model, opts backend.PrepareOptions) Command {
	quant := opts.Quantization
	if quant == "" {
		quant = b.cfg.Backend.BitNet.QuantType
	}
	repoID := m.RepoID
	if repoID == "" {
		repoID = m.ID
	}
	args := []string{"setup_env.py"}
	if repoID != "" {
		args = append(args, "-hr", repoID)
	}
	args = append(args, "-md", filepath.Dir(m.Path), "-q", quant)
	return Command{
		Dir:  b.BackendDir(),
		Name: b.Python(),
		Args: args,
	}
}

// CodegenCommand builds the Python kernel codegen script invocation.
// This generates optimized BitNet kernels without needing pip.
func (b Builder) CodegenCommand() Command {
	args := []string{
		"utils/codegen_tl2.py",
		"--model", "bitnet_b1_58-3B",
		"--BM", "160,320,320",
		"--BK", "96,96,96",
		"--bm", "32,32,32",
	}
	return Command{Dir: b.BackendDir(), Name: b.Python(), Args: args}
}

// CMakeConfigureCommand builds the cmake configuration step.
func (b Builder) CMakeConfigureCommand() Command {
	args := []string{
		"-B", "build",
		"-G", "Ninja",
		"-DBITNET_X86_TL2=OFF",
		"-DCMAKE_C_COMPILER=clang",
		"-DCMAKE_CXX_COMPILER=clang++",
		"-DLLAMA_BUILD_TOOLS=ON",
		"-DLLAMA_BUILD_EXAMPLES=ON",
		"-DLLAMA_BUILD_COMMON=ON",
		"-DLLAMA_BUILD_SERVER=OFF",
	}
	if runtime.GOOS == "windows" {
		args = append(args,
			"-DCMAKE_C_FLAGS=-D_WIN32_WINNT=0x0A00",
			"-DCMAKE_CXX_FLAGS=-D_WIN32_WINNT=0x0A00",
		)
	}
	return Command{Dir: b.BackendDir(), Name: "cmake", Args: args}
}

// CMakeBuildCommand builds the cmake compilation step.
func (b Builder) CMakeBuildCommand() Command {
	return Command{
		Dir:  b.BackendDir(),
		Name: "cmake",
		Args: []string{"--build", "build", "--config", "Release", "--target", "llama-cli"},
	}
}

// GenerateCommand builds a llama-cli invocation.
// For base models (ChatMode plain) the prompt is wrapped in a one-shot User:/Assistant: format.
// For instruct/chat models (ChatMode jinja) --jinja is passed so llama-cli applies the
// Jinja chat template embedded in the GGUF file.
func (b Builder) GenerateCommand(m model.Model, req backend.GenerateRequest, conversation bool) Command {
	prompt := req.Prompt

	isJinja := m.ChatMode == model.ChatModeJinja

	if isJinja {
		// Prompt is already Llama 3 instruct-formatted by PromptFromMessagesJinja;
		// leave it as-is and let llama-cli + --jinja handle it.
	} else {
		// Plain base-model path: wrap if not already formatted.
		if !strings.Contains(prompt, "<|eot_id|>") {
			prompt = fmt.Sprintf("User: %s<|eot_id|>Assistant: ", strings.TrimSpace(prompt))
		}
	}

	args := []string{
		"-m", m.Path,
		"-p", prompt,
		"--no-warmup",
		"--simple-io",
		"--repeat-penalty", "1.2",
		"--repeat-last-n", "64",
	}
	if isJinja {
		args = append(args, "--jinja")
	}
	if req.Options.MaxTokens > 0 {
		args = append(args, "-n", fmt.Sprintf("%d", req.Options.MaxTokens))
	}
	if req.Options.Threads > 0 {
		args = append(args, "-t", fmt.Sprintf("%d", req.Options.Threads))
	}
	if req.Options.ContextLength > 0 {
		args = append(args, "-c", fmt.Sprintf("%d", req.Options.ContextLength))
	}
	if req.Options.Temperature > 0 {
		args = append(args, "--temp", fmt.Sprintf("%.4f", req.Options.Temperature))
	}
	if req.Options.TopP > 0 {
		args = append(args, "--top-p", fmt.Sprintf("%.4f", req.Options.TopP))
	}
	if req.Options.TopK > 0 {
		args = append(args, "--top-k", fmt.Sprintf("%d", req.Options.TopK))
	}
	if req.Options.MinP > 0 {
		args = append(args, "--min-p", fmt.Sprintf("%.4f", req.Options.MinP))
	}
	// GPU layers: 0 = CPU only, >0 = offload layers to GPU
	ngl := 0
	if req.Options.GPULayers > 0 {
		ngl = req.Options.GPULayers
	}
	args = append(args, "-ngl", fmt.Sprintf("%d", ngl))

	return Command{Dir: b.BackendDir(), Name: b.LlamaCLIPath(), Args: args}
}

// PromptFromMessages converts a chat transcript into the plain BitNet chat format:
//
//	User: {msg}<|eot_id|>Assistant: {reply}<|eot_id|>...
//
// Use this for base models (ChatMode == plain).
func PromptFromMessages(messages []backend.Message) string {
	var sb strings.Builder

	for _, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "user"
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		if role == "system" {
			sb.WriteString(fmt.Sprintf("System: %s<|eot_id|>", content))
		} else if role == "user" {
			sb.WriteString(fmt.Sprintf("User: %s<|eot_id|>", content))
		} else if role == "assistant" {
			sb.WriteString(fmt.Sprintf("Assistant: %s<|eot_id|>", content))
		}
	}
	sb.WriteString("Assistant: ")
	return sb.String()
}

// PromptFromMessagesJinja converts a chat transcript into the Llama 3 instruct
// chat template format understood by llama-cli's --jinja mode:
//
//	<|begin_of_text|><|start_header_id|>system<|end_header_id|>
//	{system}<|eot_id|><|start_header_id|>user<|end_header_id|>
//	{msg}<|eot_id|><|start_header_id|>assistant<|end_header_id|>
//
// Use this for instruct/chat models (ChatMode == jinja).
func PromptFromMessagesJinja(messages []backend.Message) string {
	var sb strings.Builder
	sb.WriteString("<|begin_of_text|>")
	for _, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "user"
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf(
			"<|start_header_id|>%s<|end_header_id|>\n\n%s<|eot_id|>",
			role, content,
		))
	}
	// Open the assistant turn so the model continues from here.
	sb.WriteString("<|start_header_id|>assistant<|end_header_id|>\n\n")
	return sb.String()
}
