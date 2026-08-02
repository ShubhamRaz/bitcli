// Package cache tests BitCLI cache layout safety.
package cache

import "testing"

func TestIsInsideModelRoot(t *testing.T) {
	layout := Layout{ModelRoot: t.TempDir()}
	if !layout.IsInsideModelRoot(layout.ModelRoot) {
		t.Fatal("root should be inside itself")
	}
	if layout.IsInsideModelRoot("/definitely/outside/bitcli") {
		t.Fatal("outside path should be rejected")
	}
}

