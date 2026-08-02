// Package config owns BitCLI's YAML configuration schema and default path resolution.
package config

import (
	"os"
	"path/filepath"
)

// Paths contains the canonical filesystem locations used by BitCLI.
type Paths struct {
	Root         string
	ConfigFile   string
	DatabaseFile string
	LogDir       string
	ModelDir     string
	DownloadDir  string
	BackendDir   string
	ChatDir      string
}

// DefaultPaths resolves the platform-neutral ~/.bitcli layout requested by the project.
func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	root := filepath.Join(home, ".bitcli")
	return Paths{
		Root:         root,
		ConfigFile:   filepath.Join(root, "config.yaml"),
		DatabaseFile: filepath.Join(root, "bitcli.db"),
		LogDir:       filepath.Join(root, "logs"),
		ModelDir:     filepath.Join(root, "models"),
		DownloadDir:  filepath.Join(root, "downloads"),
		BackendDir:   filepath.Join(root, "backends"),
		ChatDir:      filepath.Join(root, "chats"),
	}, nil
}

// Ensure creates the required directories for the BitCLI home layout.
func (p Paths) Ensure() error {
	for _, dir := range []string{p.Root, p.LogDir, p.ModelDir, p.DownloadDir, p.BackendDir, p.ChatDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

