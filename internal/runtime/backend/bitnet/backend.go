// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/process"
	"github.com/bitcli/bitcli/internal/runtime/backend"
	"github.com/bitcli/bitcli/internal/utils"
	"go.uber.org/zap"
)

// Backend wraps official bitnet.cpp scripts as a BitCLI backend.
type Backend struct {
	cfg     config.Config
	paths   config.Paths
	builder Builder
	runner  *process.Runner
	log     *zap.Logger
}

// New creates a BitNet backend adapter.
func New(cfg config.Config, paths config.Paths, runner *process.Runner, log *zap.Logger) *Backend {
	if log == nil {
		log = zap.NewNop()
	}
	return &Backend{
		cfg:     cfg,
		paths:   paths,
		builder: NewBuilder(cfg, paths),
		runner:  runner,
		log:     log,
	}
}

// ID returns the stable backend ID.
func (b *Backend) ID() string {
	return "bitnet"
}

// Detect checks whether the official BitNet checkout is present and runnable.
func (b *Backend) Detect(ctx context.Context) (backend.Status, error) {
	dir := b.builder.BackendDir()
	status := backend.Status{ID: b.ID(), Path: dir}
	if !HealthFilesExist(dir) {
		status.Ready = false
		status.Message = "official BitNet checkout was not found; run bitcli doctor or bitcli update backend bitnet"
		return status, nil
	}
	version, err := b.Version(ctx)
	if err != nil {
		status.Message = err.Error()
	} else {
		status.Version = version
	}
	status.Ready = true
	return status, nil
}

// Prepare invokes setup_env.py once per model and quantization.
func (b *Backend) Prepare(ctx context.Context, m model.Model, opts backend.PrepareOptions) error {
	status, err := b.Detect(ctx)
	if err != nil {
		return err
	}
	if !status.Ready {
		return utils.NewError(utils.CodeBackendNotFound, status.Message)
	}
	if _, err := os.Stat(m.Path); err != nil {
		return utils.WrapError(utils.CodeModelNotFound, "model artifact is missing from cache", err)
	}

	quant := opts.Quantization
	if quant == "" {
		quant = b.cfg.Backend.BitNet.QuantType
	}
	marker := filepath.Join(filepath.Dir(m.Path), ".bitcli-prepared-"+quant)
	if _, err := os.Stat(marker); err == nil {
		return nil
	}

	// If backend binaries are already compiled and model file is present, mark as prepared
	backendDir := b.builder.BackendDir()
	candidateBins := []string{
		filepath.Join(backendDir, "build", "bin", "llama-cli.exe"),
		filepath.Join(backendDir, "build", "bin", "Release", "llama-cli.exe"),
		filepath.Join(backendDir, "build", "bin", "llama-cli"),
		filepath.Join(backendDir, "build", "bin", "Release", "llama-cli"),
	}
	for _, bin := range candidateBins {
		if _, err := os.Stat(bin); err == nil {
			_ = os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339Nano)), 0o644)
			return nil
		}
	}

	cmd := b.builder.SetupCommand(m, opts)
	b.log.Info("preparing bitnet model", zap.String("dir", cmd.Dir), zap.Strings("args", cmd.Args))
	if err := b.runner.RunWait(ctx, cmd.Dir, cmd.Name, cmd.Args...); err != nil {
		return utils.WrapError(utils.CodeUnavailable, "bitnet.cpp setup failed", err)
	}
	return os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339Nano)), 0o644)
}

// Generate streams output from official run_inference.py.
func (b *Backend) Generate(ctx context.Context, m model.Model, req backend.GenerateRequest) (<-chan backend.TokenEvent, <-chan error) {
	cmd, unsupported := b.builder.GenerateCommand(m, req, false)
	return b.runInference(ctx, cmd, unsupported, req.Prompt)
}

// Chat streams output from official run_inference.py in conversational mode.
func (b *Backend) Chat(ctx context.Context, m model.Model, req backend.ChatRequest) (<-chan backend.TokenEvent, <-chan error) {
	prompt := PromptFromMessages(req.Messages)
	genReq := backend.GenerateRequest{
		ModelID: req.ModelID,
		Prompt:  prompt,
		Options: req.Options,
	}
	cmd, unsupported := b.builder.GenerateCommand(m, genReq, true)
	return b.runInference(ctx, cmd, unsupported, prompt)
}

// Stop is reserved for long-lived backend sessions.
func (b *Backend) Stop(ctx context.Context, sessionID string) error {
	return nil
}

// Version reports the current git revision for the official BitNet checkout when available.
func (b *Backend) Version(ctx context.Context) (string, error) {
	dir := b.builder.BackendDir()
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--short", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (b *Backend) runInference(ctx context.Context, cmd Command, unsupported []string, prompt string) (<-chan backend.TokenEvent, <-chan error) {
	events := make(chan backend.TokenEvent, 32)
	errs := make(chan error, 1)
	procEvents, procErrs := b.runner.RunStream(ctx, cmd.Dir, cmd.Name, cmd.Args...)
	filter := NewStreamFilter(prompt)

	go func() {
		defer close(events)
		defer close(errs)
		for _, name := range unsupported {
			b.log.Debug("bitnet.cpp option is not exposed by current run_inference.py", zap.String("option", name))
		}
		for ev := range procEvents {
			if ev.Stream == "stderr" {
				if process.LooksLikeDiagnostic(ev.Text) {
					b.log.Debug("bitnet.cpp diagnostic", zap.String("text", strings.TrimSpace(ev.Text)))
				}
				continue
			}
			cleaned := filter.Filter(ev.Text)
			if cleaned != "" {
				events <- backend.TokenEvent{Text: cleaned, CreatedAt: time.Now().UTC()}
			}
		}
		if err := <-procErrs; err != nil {
			errs <- fmt.Errorf("bitnet.cpp inference failed: %w", err)
			return
		}
		events <- backend.TokenEvent{Done: true, CreatedAt: time.Now().UTC()}
		errs <- nil
	}()

	return events, errs
}

