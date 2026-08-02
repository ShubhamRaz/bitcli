// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bitcli/bitcli/internal/process"
)

// Installer manages an official Microsoft BitNet checkout without modifying its source.
type Installer struct {
	Runner *process.Runner
}

// EnsureClone clones the official BitNet repository when the managed checkout is absent.
func (i Installer) EnsureClone(ctx context.Context, repoURL, target string) error {
	if HealthFilesExist(target) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return i.Runner.RunWait(ctx, filepath.Dir(target), "git", "clone", "--recursive", repoURL, target)
}

// CheckoutRevision checks out a requested revision in the managed repository.
func (i Installer) CheckoutRevision(ctx context.Context, target, revision string) error {
	if revision == "" {
		return nil
	}
	if err := i.Runner.RunWait(ctx, target, "git", "fetch", "--all", "--tags"); err != nil {
		return err
	}
	return i.Runner.RunWait(ctx, target, "git", "checkout", revision)
}

