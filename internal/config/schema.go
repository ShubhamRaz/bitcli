// Package config owns BitCLI's YAML configuration schema and default path resolution.
package config

// Config is the persisted BitCLI configuration document.
type Config struct {
	Version       int            `mapstructure:"version" yaml:"version"`
	DefaultModel string         `mapstructure:"default_model" yaml:"default_model"`
	DefaultBackend string       `mapstructure:"default_backend" yaml:"default_backend"`
	API           APIConfig     `mapstructure:"api" yaml:"api"`
	Runtime       RuntimeConfig `mapstructure:"runtime" yaml:"runtime"`
	Backend       BackendConfig `mapstructure:"backend" yaml:"backend"`
	Download      DownloadConfig `mapstructure:"download" yaml:"download"`
	Theme         string        `mapstructure:"theme" yaml:"theme"`
	Logging       LoggingConfig `mapstructure:"logging" yaml:"logging"`
}

// APIConfig controls the local HTTP server.
type APIConfig struct {
	Host         string   `mapstructure:"host" yaml:"host"`
	Port         int      `mapstructure:"port" yaml:"port"`
	AllowOrigins []string `mapstructure:"allow_origins" yaml:"allow_origins"`
}

// RuntimeConfig contains default model inference options.
type RuntimeConfig struct {
	Temperature   float64 `mapstructure:"temperature" yaml:"temperature"`
	TopP          float64 `mapstructure:"top_p" yaml:"top_p"`
	TopK          int     `mapstructure:"top_k" yaml:"top_k"`
	Threads       int     `mapstructure:"threads" yaml:"threads"`
	GPULayers     int     `mapstructure:"gpu_layers" yaml:"gpu_layers"`
	ContextLength int     `mapstructure:"context_length" yaml:"context_length"`
	MaxTokens     int     `mapstructure:"max_tokens" yaml:"max_tokens"`
	Device        string  `mapstructure:"device" yaml:"device"`
}

// BackendConfig groups backend-specific settings.
type BackendConfig struct {
	BitNet BitNetConfig `mapstructure:"bitnet" yaml:"bitnet"`
}

// BitNetConfig controls the official Microsoft BitNet checkout used by BitCLI.
type BitNetConfig struct {
	InstallMode string `mapstructure:"install_mode" yaml:"install_mode"`
	Path        string `mapstructure:"path" yaml:"path"`
	RepoURL     string `mapstructure:"repo_url" yaml:"repo_url"`
	Revision    string `mapstructure:"revision" yaml:"revision"`
	Python      string `mapstructure:"python" yaml:"python"`
	QuantType   string `mapstructure:"quant_type" yaml:"quant_type"`
}

// DownloadConfig controls Hugging Face Hub transfers.
type DownloadConfig struct {
	Mirror      string `mapstructure:"mirror" yaml:"mirror"`
	Concurrency int   `mapstructure:"concurrency" yaml:"concurrency"`
	Retries     int   `mapstructure:"retries" yaml:"retries"`
	TokenEnv    string `mapstructure:"token_env" yaml:"token_env"`
}

// LoggingConfig controls Zap log verbosity and file logging.
type LoggingConfig struct {
	Level string `mapstructure:"level" yaml:"level"`
	File  bool   `mapstructure:"file" yaml:"file"`
}

