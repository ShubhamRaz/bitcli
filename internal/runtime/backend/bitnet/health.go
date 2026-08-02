// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"os"
	"path/filepath"
)

// HealthFilesExist verifies the official scripts BitCLI depends on.
func HealthFilesExist(dir string) bool {
	for _, name := range []string{"setup_env.py", "run_inference.py"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}
	return true
}

