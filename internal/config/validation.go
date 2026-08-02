// Package config owns BitCLI's YAML configuration schema and default path resolution.
package config

import (
	"fmt"

	"github.com/bitcli/bitcli/internal/utils"
)

// Validate checks the loaded configuration before any command uses it.
func Validate(cfg Config) error {
	if cfg.Version <= 0 {
		return utils.NewError(utils.CodeConfigInvalid, "config version must be positive")
	}
	if cfg.DefaultBackend == "" {
		return utils.NewError(utils.CodeConfigInvalid, "default backend is required")
	}
	if cfg.API.Port <= 0 || cfg.API.Port > 65535 {
		return utils.NewError(utils.CodeConfigInvalid, fmt.Sprintf("api port %d is out of range", cfg.API.Port))
	}
	if cfg.Runtime.Temperature < 0 || cfg.Runtime.Temperature > 2 {
		return utils.NewError(utils.CodeConfigInvalid, "temperature must be between 0 and 2")
	}
	if cfg.Runtime.TopP < 0 || cfg.Runtime.TopP > 1 {
		return utils.NewError(utils.CodeConfigInvalid, "top_p must be between 0 and 1")
	}
	if cfg.Runtime.ContextLength <= 0 {
		return utils.NewError(utils.CodeConfigInvalid, "context length must be positive")
	}
	if cfg.Download.Mirror == "" {
		return utils.NewError(utils.CodeConfigInvalid, "download mirror is required")
	}
	return nil
}

