// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitcli/bitcli/internal/process"
)

// Installer manages an official Microsoft BitNet checkout without modifying its source.
type Installer struct {
	Runner *process.Runner
}

// EnsureClone clones the official BitNet repository when the managed checkout
// is absent. A leftover partial clone (e.g. a previous run was interrupted)
// previously made every subsequent `git clone` fail with "destination path
// already exists"; such remnants are now recovered, or removed when allowClean
// is true (only for bitcli-managed directories, never user-configured paths).
func (i Installer) EnsureClone(ctx context.Context, repoURL, target string, allowClean bool) error {
	if HealthFilesExist(target) {
		return nil
	}
	if _, err := os.Stat(target); err == nil {
		// Target exists but the required scripts are missing.
		if isGitRepo(target) {
			// Attempt recovery: establish the remote default branch,
			// force-checkout it, then refresh submodules.
			_ = i.Runner.RunWait(ctx, target, "git", "remote", "set-head", "origin", "-a")
			if err := i.Runner.RunWait(ctx, target, "git", "fetch", "origin"); err == nil {
				_ = i.Runner.RunWait(ctx, target, "git", "checkout", "-f", "origin/HEAD")
				_ = i.Runner.RunWait(ctx, target, "git", "submodule", "update", "--init", "--recursive")
				if HealthFilesExist(target) {
					return nil
				}
			}
		}
		if !allowClean {
			return fmt.Errorf("%s exists but is not a usable BitNet checkout (missing setup_env.py); remove it manually or point backend.bitnet.path elsewhere", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return fmt.Errorf("incomplete BitNet checkout at %s could not be recovered: %w", target, err)
		}
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

	// Support checking out GitHub Pull Requests directly (e.g. "pr/605")
	if strings.HasPrefix(revision, "pr/") {
		prNum := strings.TrimPrefix(revision, "pr/")
		branchName := "pr" + prNum
		if err := i.Runner.RunWait(ctx, target, "git", "fetch", "origin", "pull/"+prNum+"/head:"+branchName); err != nil {
			return err
		}
		return i.Runner.RunWait(ctx, target, "git", "checkout", branchName)
	}

	if err := i.Runner.RunWait(ctx, target, "git", "fetch", "--all", "--tags"); err != nil {
		return err
	}
	return i.Runner.RunWait(ctx, target, "git", "checkout", revision)
}

// isGitRepo reports whether dir contains a usable git repository.
func isGitRepo(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err != nil {
		return false
	}
	return true
}
