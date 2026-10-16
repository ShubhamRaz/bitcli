// Package runtime coordinates backend-neutral model execution.
package runtime

import (
	"strings"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/runtime/backend"
)

// Options are common generation settings accepted by BitCLI.
type Options = backend.Options

// Message is a chat message in a backend-neutral representation.
type Message = backend.Message

// GenerateRequest asks a backend to generate text from a prompt.
type GenerateRequest = backend.GenerateRequest

// ChatRequest asks a backend to continue a chat transcript.
type ChatRequest = backend.ChatRequest

// TokenEvent is one streamed backend event.
type TokenEvent = backend.TokenEvent

// OptionsFromConfig resolves the effective runtime options from the persisted
// configuration. It is the single source of truth shared by the CLI and every
// API surface (native + OpenAI-compatible), so `bitcli device gpu` and
// `runtime.min_p` behave identically everywhere.
//
// Device semantics:
//   - device "gpu" with no explicit gpu_layers offloads all layers (-ngl 99)
//   - device "cpu" always forces CPU inference (-ngl 0), even when a stale
//     runtime.gpu_layers value remains in the config file
func OptionsFromConfig(cfg config.Config) Options {
	gpuLayers := cfg.Runtime.GPULayers
	switch {
	case strings.EqualFold(cfg.Runtime.Device, "gpu") && gpuLayers == 0:
		gpuLayers = 99
	case strings.EqualFold(cfg.Runtime.Device, "cpu"):
		gpuLayers = 0
	}
	return Options{
		Temperature:   cfg.Runtime.Temperature,
		TopP:          cfg.Runtime.TopP,
		TopK:          cfg.Runtime.TopK,
		MinP:          cfg.Runtime.MinP,
		Threads:       cfg.Runtime.Threads,
		GPULayers:     gpuLayers,
		ContextLength: cfg.Runtime.ContextLength,
		MaxTokens:     cfg.Runtime.MaxTokens,
	}
}
