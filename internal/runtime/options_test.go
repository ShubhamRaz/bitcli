// Package runtime tests backend-neutral option resolution.
package runtime

import (
	"testing"

	"github.com/bitcli/bitcli/internal/config"
)

// device=gpu with no explicit layers offloads everything; device=cpu forces
// zero even when a stale gpu_layers value remains in the config.
func TestOptionsFromConfigDeviceHandling(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Runtime.Device = "cpu"
	cfg.Runtime.GPULayers = 99
	opts := OptionsFromConfig(cfg)
	if opts.GPULayers != 0 {
		t.Fatalf("device=cpu must force gpu_layers 0, got %d", opts.GPULayers)
	}

	cfg.Runtime.Device = "gpu"
	cfg.Runtime.GPULayers = 0
	opts = OptionsFromConfig(cfg)
	if opts.GPULayers != 99 {
		t.Fatalf("device=gpu with no explicit layers must offload all layers, got %d", opts.GPULayers)
	}

	// Explicit gpu_layers wins over the device default.
	cfg.Runtime.GPULayers = 7
	opts = OptionsFromConfig(cfg)
	if opts.GPULayers != 7 {
		t.Fatalf("explicit gpu_layers should be honored, got %d", opts.GPULayers)
	}

	// min_p flows through for API and CLI alike.
	if opts.MinP != cfg.Runtime.MinP {
		t.Fatalf("min_p should be carried into options, got %v", opts.MinP)
	}
}
