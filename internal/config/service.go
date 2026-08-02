// Package config owns BitCLI's YAML configuration schema and default path resolution.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Service loads, validates, and writes BitCLI configuration files.
type Service struct {
	paths Paths
}

// NewService creates a config service with the default BitCLI home layout.
func NewService() (*Service, error) {
	paths, err := DefaultPaths()
	if err != nil {
		return nil, err
	}
	return &Service{paths: paths}, nil
}

// Paths returns the default paths known by the service.
func (s *Service) Paths() Paths {
	return s.paths
}

// Load reads configuration from disk, creating a default file when one is absent.
func (s *Service) Load(configPath string) (Config, Paths, error) {
	paths := s.paths
	if configPath != "" {
		paths.ConfigFile = configPath
		paths.Root = filepath.Dir(configPath)
		paths.DatabaseFile = filepath.Join(paths.Root, "bitcli.db")
		paths.LogDir = filepath.Join(paths.Root, "logs")
		paths.ModelDir = filepath.Join(paths.Root, "models")
		paths.DownloadDir = filepath.Join(paths.Root, "downloads")
		paths.BackendDir = filepath.Join(paths.Root, "backends")
		paths.ChatDir = filepath.Join(paths.Root, "chats")
	}

	if err := paths.Ensure(); err != nil {
		return Config{}, paths, err
	}
	if _, err := os.Stat(paths.ConfigFile); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := s.Write(paths.ConfigFile, cfg); err != nil {
			return Config{}, paths, err
		}
		return cfg, paths, nil
	}

	v := newViper()
	v.SetConfigFile(paths.ConfigFile)
	if err := v.ReadInConfig(); err != nil {
		return Config{}, paths, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, paths, err
	}
	if err := Validate(cfg); err != nil {
		return Config{}, paths, err
	}
	return cfg, paths, nil
}

// Write persists a configuration file using Viper's YAML writer.
func (s *Service) Write(path string, cfg Config) error {
	v := newViper()
	setConfigValues(v, cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return v.WriteConfigAs(path)
	}
	return v.WriteConfigAs(path)
}

// Set updates a simple dotted configuration key in the YAML file.
func (s *Service) Set(configPath, key string, value any) error {
	cfg, _, err := s.Load(configPath)
	if err != nil {
		return err
	}
	v := newViper()
	setConfigValues(v, cfg)
	v.Set(key, value)
	var updated Config
	if err := v.Unmarshal(&updated); err != nil {
		return err
	}
	if err := Validate(updated); err != nil {
		return err
	}
	target := configPath
	if target == "" {
		target = s.paths.ConfigFile
	}
	return s.Write(target, updated)
}

// Get returns a simple dotted configuration key from the loaded config.
func (s *Service) Get(configPath, key string) (any, error) {
	cfg, _, err := s.Load(configPath)
	if err != nil {
		return nil, err
	}
	v := newViper()
	setConfigValues(v, cfg)
	if !v.IsSet(key) {
		return nil, fmt.Errorf("unknown config key %q", key)
	}
	return v.Get(key), nil
}

func newViper() *viper.Viper {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetEnvPrefix("BITCLI")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	setConfigDefaults(v, DefaultConfig())
	return v
}

func setConfigDefaults(v *viper.Viper, cfg Config) {
	v.SetDefault("version", cfg.Version)
	v.SetDefault("default_model", cfg.DefaultModel)
	v.SetDefault("default_backend", cfg.DefaultBackend)
	v.SetDefault("api.host", cfg.API.Host)
	v.SetDefault("api.port", cfg.API.Port)
	v.SetDefault("api.allow_origins", cfg.API.AllowOrigins)
	v.SetDefault("runtime.temperature", cfg.Runtime.Temperature)
	v.SetDefault("runtime.top_p", cfg.Runtime.TopP)
	v.SetDefault("runtime.top_k", cfg.Runtime.TopK)
	v.SetDefault("runtime.threads", cfg.Runtime.Threads)
	v.SetDefault("runtime.gpu_layers", cfg.Runtime.GPULayers)
	v.SetDefault("runtime.context_length", cfg.Runtime.ContextLength)
	v.SetDefault("runtime.max_tokens", cfg.Runtime.MaxTokens)
	v.SetDefault("backend.bitnet.install_mode", cfg.Backend.BitNet.InstallMode)
	v.SetDefault("backend.bitnet.path", cfg.Backend.BitNet.Path)
	v.SetDefault("backend.bitnet.repo_url", cfg.Backend.BitNet.RepoURL)
	v.SetDefault("backend.bitnet.revision", cfg.Backend.BitNet.Revision)
	v.SetDefault("backend.bitnet.python", cfg.Backend.BitNet.Python)
	v.SetDefault("backend.bitnet.quant_type", cfg.Backend.BitNet.QuantType)
	v.SetDefault("download.mirror", cfg.Download.Mirror)
	v.SetDefault("download.concurrency", cfg.Download.Concurrency)
	v.SetDefault("download.retries", cfg.Download.Retries)
	v.SetDefault("download.token_env", cfg.Download.TokenEnv)
	v.SetDefault("theme", cfg.Theme)
	v.SetDefault("logging.level", cfg.Logging.Level)
	v.SetDefault("logging.file", cfg.Logging.File)
}

func setConfigValues(v *viper.Viper, cfg Config) {
	v.Set("version", cfg.Version)
	v.Set("default_model", cfg.DefaultModel)
	v.Set("default_backend", cfg.DefaultBackend)
	v.Set("api.host", cfg.API.Host)
	v.Set("api.port", cfg.API.Port)
	v.Set("api.allow_origins", cfg.API.AllowOrigins)
	v.Set("runtime.temperature", cfg.Runtime.Temperature)
	v.Set("runtime.top_p", cfg.Runtime.TopP)
	v.Set("runtime.top_k", cfg.Runtime.TopK)
	v.Set("runtime.threads", cfg.Runtime.Threads)
	v.Set("runtime.gpu_layers", cfg.Runtime.GPULayers)
	v.Set("runtime.context_length", cfg.Runtime.ContextLength)
	v.Set("runtime.max_tokens", cfg.Runtime.MaxTokens)
	v.Set("backend.bitnet.install_mode", cfg.Backend.BitNet.InstallMode)
	v.Set("backend.bitnet.path", cfg.Backend.BitNet.Path)
	v.Set("backend.bitnet.repo_url", cfg.Backend.BitNet.RepoURL)
	v.Set("backend.bitnet.revision", cfg.Backend.BitNet.Revision)
	v.Set("backend.bitnet.python", cfg.Backend.BitNet.Python)
	v.Set("backend.bitnet.quant_type", cfg.Backend.BitNet.QuantType)
	v.Set("download.mirror", cfg.Download.Mirror)
	v.Set("download.concurrency", cfg.Download.Concurrency)
	v.Set("download.retries", cfg.Download.Retries)
	v.Set("download.token_env", cfg.Download.TokenEnv)
	v.Set("theme", cfg.Theme)
	v.Set("logging.level", cfg.Logging.Level)
	v.Set("logging.file", cfg.Logging.File)
}
