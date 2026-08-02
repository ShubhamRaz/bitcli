// Package config owns BitCLI's YAML configuration schema and default path resolution.
package config

// DefaultConfig returns production-oriented defaults that keep the API local and conservative.
func DefaultConfig() Config {
	return Config{
		Version:        1,
		DefaultModel:   "microsoft/BitNet-b1.58-2B-4T",
		DefaultBackend: "bitnet",
		API: APIConfig{
			Host:         "127.0.0.1",
			Port:         11434,
			AllowOrigins: []string{},
		},
		Runtime: RuntimeConfig{
			Temperature:   0.2,
			TopP:          0.9,
			TopK:          40,
			Threads:       0,
			GPULayers:     0,
			ContextLength: 4096,
			MaxTokens:     512,
			Device:        "cpu",
		},
		Backend: BackendConfig{
			BitNet: BitNetConfig{
				InstallMode: "auto",
				Path:        "",
				RepoURL:     "https://github.com/microsoft/BitNet.git",
				Revision:    "",
				Python:      "",
				QuantType:   "i2_s",
			},
		},
		Download: DownloadConfig{
			Mirror:      "https://huggingface.co",
			Concurrency: 4,
			Retries:     3,
			TokenEnv:    "HF_TOKEN",
		},
		Theme: "auto",
		Logging: LoggingConfig{
			Level: "info",
			File:  true,
		},
	}
}
