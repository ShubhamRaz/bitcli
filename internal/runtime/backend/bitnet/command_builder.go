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

// defaultSystemPrompt provides instruction grounding for small models that
// otherwise produce free-form text completions instead of answering questions.
const defaultSystemPrompt = "You are a helpful assistant."

// GenerateCommand builds a llama-cli invocation using native conversation mode.
// The BitNet llama-cli auto-enables -cnv when the model has a chat template.
// In conversation mode:
//   - "-sys" sets the system prompt (applied via the model's jinja chat template)
//   - "-p" sets the user message (first conversational turn)
//   - "-st" makes it single-turn (process one exchange and exit)
func (b Builder) GenerateCommand(m model.Model, req backend.GenerateRequest, conversation bool) Command {
	userMessage := strings.TrimSpace(req.Prompt)
	systemPrompt := defaultSystemPrompt

	// If the prompt was pre-formatted with chat template tokens (from Chat flow),
	// extract the components. Otherwise use the raw prompt as the user message.
	if strings.Contains(req.Prompt, "<|begin_of_text|>") || strings.Contains(req.Prompt, "<|start_header_id|>") {
		// Pre-formatted prompt from PromptFromMessages — extract system + user parts
		sys, user := extractFromChatTemplate(req.Prompt)
		if sys != "" {
			systemPrompt = sys
		}
		if user != "" {
			userMessage = user
		}
	}

	args := []string{
		"-m", m.Path,
		"-sys", systemPrompt,
		"-p", userMessage,
		"-cnv",
		"-st",
		"--no-warmup",
		"--simple-io",
		"--repeat-penalty", "1.2",
		"--repeat-last-n", "64",
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

// extractFromChatTemplate parses a pre-formatted LLaMA-3 chat template prompt
// to extract the system prompt and the last user message.
func extractFromChatTemplate(prompt string) (system, user string) {
	// Extract system message
	const sysHeader = "<|start_header_id|>system<|end_header_id|>\n\n"
	if idx := strings.Index(prompt, sysHeader); idx >= 0 {
		rest := prompt[idx+len(sysHeader):]
		if eot := strings.Index(rest, "<|eot_id|>"); eot >= 0 {
			system = strings.TrimSpace(rest[:eot])
		}
	}
	// Extract last user message
	const userHeader = "<|start_header_id|>user<|end_header_id|>\n\n"
	lastUserIdx := strings.LastIndex(prompt, userHeader)
	if lastUserIdx >= 0 {
		rest := prompt[lastUserIdx+len(userHeader):]
		if eot := strings.Index(rest, "<|eot_id|>"); eot >= 0 {
			user = strings.TrimSpace(rest[:eot])
		}
	}
	return
}

// PromptFromMessages converts a chat transcript into the standard LLaMA-3 instruction chat template:
//
//	<|begin_of_text|><|start_header_id|>system<|end_header_id|>
//	{system_prompt}<|eot_id|><|start_header_id|>user<|end_header_id|>
//	{msg}<|eot_id|><|start_header_id|>assistant<|end_header_id|>
//	{reply}<|eot_id|>...
//	<|start_header_id|>assistant<|end_header_id|>
//
// If no system message is present in the conversation, a default one is injected
// to provide instruction grounding for small models.
func PromptFromMessages(messages []backend.Message) string {
	var sb strings.Builder
	sb.WriteString("<|begin_of_text|>")

	// Check if any message has role "system"; if not, inject default system prompt.
	hasSystem := false
	for _, msg := range messages {
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		sb.WriteString(fmt.Sprintf("<|start_header_id|>system<|end_header_id|>\n\n%s<|eot_id|>", defaultSystemPrompt))
	}

	for _, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		if role == "" {
			role = "user"
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("<|start_header_id|>%s<|end_header_id|>\n\n%s<|eot_id|>", role, content))
	}
	sb.WriteString("<|start_header_id|>assistant<|end_header_id|>\n\n")
	return sb.String()
}
