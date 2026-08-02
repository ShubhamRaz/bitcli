// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"context"
	"fmt"
	"os"
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
		status.Message = "official BitNet checkout was not found; run bitcli setup"
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

// Prepare ensures the llama-cli binary is compiled. It first checks for
// an existing binary, and if missing, builds it using cmake directly
// (bypassing setup_env.py which has pip dependency issues).
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

	// Check if llama-cli is already compiled.
	llamaCLI := b.builder.LlamaCLIPath()
	if _, err := os.Stat(llamaCLI); err == nil {
		_ = os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339Nano)), 0o644)
		return nil
	}

	// ── Build llama-cli from source using cmake directly ──
	// This bypasses setup_env.py which fails due to pip issues on some systems.
	b.log.Info("llama-cli not found, building from source via cmake")

	// Step 1: Run kernel codegen (Python script, no pip needed).
	codegen := b.builder.CodegenCommand()
	b.log.Info("running kernel codegen", zap.Strings("args", codegen.Args))
	if output, err := b.runner.RunWaitVerbose(ctx, codegen.Dir, codegen.Name, codegen.Args...); err != nil {
		b.log.Warn("kernel codegen failed (non-fatal, using defaults)", zap.Error(err), zap.String("output", output))
		// Non-fatal: the build may still succeed with default kernels.
	}

	// Step 2: cmake configure.
	configure := b.builder.CMakeConfigureCommand()
	b.log.Info("cmake configure", zap.Strings("args", configure.Args))
	if output, err := b.runner.RunWaitVerbose(ctx, configure.Dir, configure.Name, configure.Args...); err != nil {
		return utils.NewError(utils.CodeUnavailable,
			fmt.Sprintf("cmake configure failed: %v\n%s", err, output))
	}

	// Step 3: cmake build.
	build := b.builder.CMakeBuildCommand()
	b.log.Info("cmake build", zap.Strings("args", build.Args))
	if output, err := b.runner.RunWaitVerbose(ctx, build.Dir, build.Name, build.Args...); err != nil {
		return utils.NewError(utils.CodeUnavailable,
			fmt.Sprintf("cmake build failed: %v\n%s", err, output))
	}

	// Verify binary was produced.
	if _, err := os.Stat(b.builder.LlamaCLIPath()); err != nil {
		return utils.NewError(utils.CodeUnavailable,
			"cmake build completed but llama-cli binary was not found; check build output above")
	}

	_ = os.WriteFile(marker, []byte(time.Now().UTC().Format(time.RFC3339Nano)), 0o644)
	b.log.Info("llama-cli built successfully", zap.String("path", b.builder.LlamaCLIPath()))
	return nil
}

// Generate streams output from llama-cli.
func (b *Backend) Generate(ctx context.Context, m model.Model, req backend.GenerateRequest) (<-chan backend.TokenEvent, <-chan error) {
	cmd := b.builder.GenerateCommand(m, req, false)
	return b.runInference(ctx, cmd, req.Prompt)
}

// Chat streams output from llama-cli using the full chat history
// formatted with the model's trained chat template. Each call
// re-sends the entire conversation in one-shot mode.
func (b *Backend) Chat(ctx context.Context, m model.Model, req backend.ChatRequest) (<-chan backend.TokenEvent, <-chan error) {
	prompt := PromptFromMessages(req.Messages)
	genReq := backend.GenerateRequest{
		ModelID: req.ModelID,
		Prompt:  prompt,
		Options: req.Options,
	}
	cmd := b.builder.GenerateCommand(m, genReq, false)
	return b.runInference(ctx, cmd, prompt)
}

// Stop is reserved for long-lived backend sessions.
func (b *Backend) Stop(ctx context.Context, sessionID string) error {
	return nil
}

// Version reports the current git revision for the official BitNet checkout when available.
func (b *Backend) Version(ctx context.Context) (string, error) {
	dir := b.builder.BackendDir()
	output, err := b.runner.RunWaitVerbose(ctx, "", "git", "-C", dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (b *Backend) runInference(ctx context.Context, cmd Command, prompt string) (<-chan backend.TokenEvent, <-chan error) {
	events := make(chan backend.TokenEvent, 32)
	errs := make(chan error, 1)
	procEvents, procErrs := b.runner.RunStream(ctx, cmd.Dir, cmd.Name, cmd.Args...)
	filter := NewStreamFilter(prompt)

	go func() {
		defer close(events)
		defer close(errs)
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
