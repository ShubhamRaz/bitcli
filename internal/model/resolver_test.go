// Package model tests BitCLI model identity resolution.
package model

import "testing"

func TestDefaultCatalogResolvesOfficialAlias(t *testing.T) {
	resolver := NewResolver(DefaultCatalog())
	artifact, err := resolver.Resolve("microsoft/BitNet-b1.58-2B-4T")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if artifact.RepoID != "microsoft/bitnet-b1.58-2B-4T-gguf" {
		t.Fatalf("unexpected repo id: %s", artifact.RepoID)
	}
	if artifact.Filename != "ggml-model-i2_s.gguf" {
		t.Fatalf("unexpected filename: %s", artifact.Filename)
	}
}

func TestDefaultCatalogRejectsUnknownModel(t *testing.T) {
	resolver := NewResolver(DefaultCatalog())
	if _, err := resolver.Resolve("unknown/model"); err == nil {
		t.Fatal("expected unknown model to fail")
	}
}

