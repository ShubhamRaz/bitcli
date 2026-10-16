// Package model tests BitCLI model catalog resolution.
package model

import (
	"strings"
	"testing"
)

func TestDefaultCatalogResolvesLowercase(t *testing.T) {
	catalog := DefaultCatalog()
	artifact, ok := catalog.Resolve("microsoft/bitnet-b1.58-2b-4t")
	if !ok {
		t.Fatal("lowercase alias should resolve")
	}
	if artifact.RepoID == "" {
		t.Fatal("resolved artifact should have a repo ID")
	}
}

func TestDefaultCatalogResolvesCanonicalID(t *testing.T) {
	catalog := DefaultCatalog()
	artifact, ok := catalog.Resolve("microsoft/bitnet-b1.58-2B-4T-gguf")
	if !ok {
		t.Fatal("canonical ID alias should resolve")
	}
	if artifact.Filename != "ggml-model-i2_s.gguf" {
		t.Fatalf("unexpected filename: %s", artifact.Filename)
	}
}

func TestDefaultCatalogArtifactFieldsPopulated(t *testing.T) {
	catalog := DefaultCatalog()
	artifact, ok := catalog.Resolve("microsoft/BitNet-b1.58-2B-4T")
	if !ok {
		t.Fatal("should resolve official alias")
	}
	if artifact.Backend != "bitnet" {
		t.Fatalf("expected backend 'bitnet', got %q", artifact.Backend)
	}
	if artifact.ContextLength <= 0 {
		t.Fatal("context length should be positive")
	}
	if artifact.Parameters == "" {
		t.Fatal("parameters should not be empty")
	}
	if !strings.Contains(artifact.RepoID, "gguf") {
		t.Fatalf("repo ID should reference gguf, got %q", artifact.RepoID)
	}
}

func TestDefaultCatalogUnknownModel(t *testing.T) {
	catalog := DefaultCatalog()
	_, ok := catalog.Resolve("completely/unknown-model")
	if ok {
		t.Fatal("unknown model should not resolve")
	}
}

func TestDefaultCatalogEmptyStringReturnsNotFound(t *testing.T) {
	catalog := DefaultCatalog()
	_, ok := catalog.Resolve("")
	if ok {
		t.Fatal("empty string should not resolve to a model")
	}
}
