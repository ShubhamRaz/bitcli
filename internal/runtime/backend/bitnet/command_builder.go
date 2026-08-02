// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/runtime/backend"
)

// Command describes an external process invocation without shell interpolation.
type Command struct {
	Dir  string
	Name string
	Args []string
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

// SetupCommand builds the official setup_env.py invocation.
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

// GenerateCommand builds the official run_inference.py invocation.
func (b Builder) GenerateCommand(m model.Model, req backend.GenerateRequest, conversation bool) (Command, []string) {
	args := []string{
		"run_inference.py",
		"-m", m.Path,
		"-p", req.Prompt,
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
		args = append(args, "-temp", fmt.Sprintf("%.4f", req.Options.Temperature))
	}
	if req.Options.TopP > 0 {
		args = append(args, "--top-p", fmt.Sprintf("%.4f", req.Options.TopP))
	}
	if req.Options.TopK > 0 {
		args = append(args, "--top-k", fmt.Sprintf("%d", req.Options.TopK))
	}
	if conversation {
		args = append(args, "-cnv")
	}

	unsupported := make([]string, 0, 1)
	if req.Options.GPULayers != 0 {
		unsupported = append(unsupported, "gpu_layers")
	}
	return Command{Dir: b.BackendDir(), Name: b.Python(), Args: args}, unsupported
}

// defaultSystemPrompt is injected when no system message is provided so the
// model behaves as a helpful assistant rather than a raw text completer.
const defaultSystemPrompt = "You are a helpful, concise, and friendly AI assistant. Answer the user's questions clearly and accurately."

// PromptFromMessages converts a chat transcript into the chat template format
// that BitNet-b1.58-2B-4T was trained on:
//
//	<|begin_of_text|>System: {system}<|eot_id|>
//	User: {msg}<|eot_id|>
//	Assistant: {msg}<|eot_id|>
//	...
//	Assistant:
func PromptFromMessages(messages []backend.Message) string {
	var sb strings.Builder
	sb.WriteString("<|begin_of_text|>")

	// Inject a default system prompt when no explicit one is provided.
	hasSystem := false
	for _, msg := range messages {
		if strings.EqualFold(strings.TrimSpace(msg.Role), "system") {
			hasSystem = true
			break
		}
	}
	if !hasSystem {
		sb.WriteString("System: ")
		sb.WriteString(defaultSystemPrompt)
		sb.WriteString("<|eot_id|>\n")
	}

	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "user"
		}
		sb.WriteString(strings.ToUpper(role[:1]))
		if len(role) > 1 {
			sb.WriteString(strings.ToLower(role[1:]))
		}
		sb.WriteString(": ")
		sb.WriteString(strings.TrimSpace(msg.Content))
		sb.WriteString("<|eot_id|>\n")
	}
	sb.WriteString("Assistant:")
	return sb.String()
}

