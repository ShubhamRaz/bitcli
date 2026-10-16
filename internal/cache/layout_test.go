// Package cache tests BitCLI cache layout safety.
package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/model"
)

func TestIsInsideModelRoot(t *testing.T) {
	layout := Layout{ModelRoot: t.TempDir()}
	if !layout.IsInsideModelRoot(layout.ModelRoot) {
		t.Fatal("root should be inside itself")
	}
	if layout.IsInsideModelRoot("/definitely/outside/bitcli") {
		t.Fatal("outside path should be rejected")
	}
}

// Regression test: the base and -chat catalog variants share one GGUF file.
// Removing one variant must keep the shared artifact when the sibling still
// references it, and must remove it when no sibling remains.
func TestRemoveModelKeepsSharedArtifactForSibling(t *testing.T) {
	root := t.TempDir()
	paths := config.Paths{
		ModelDir:    filepath.Join(root, "models"),
		DownloadDir: filepath.Join(root, "downloads"),
		BackendDir:  filepath.Join(root, "backends"),
	}
	svc := NewService(paths)
	if err := svc.Ensure(); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(paths.ModelDir, "microsoft", "bitnet-b1.58-2B-4T-gguf", "main", "ggml-model-i2_s.gguf")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte("gguf-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	base := model.Model{ID: "base-id", UserID: "microsoft/BitNet-b1.58-2B-4T", Path: artifactPath}
	chat := model.Model{ID: "chat-id", UserID: "microsoft/BitNet-b1.58-2B-4T-chat", Path: artifactPath}

	// Removing the base variant while the chat variant still exists must
	// NOT delete the shared file.
	if err := svc.RemoveModel(base, []model.Model{chat}); err != nil {
		t.Fatalf("RemoveModel with sibling: %v", err)
	}
	if _, err := os.Stat(artifactPath); err != nil {
		t.Fatalf("shared artifact should survive sibling removal: %v", err)
	}

	// Removing with no sibling must delete the file and clean the dir.
	if err := svc.RemoveModel(base, nil); err != nil {
		t.Fatalf("RemoveModel without sibling: %v", err)
	}
	if _, err := os.Stat(artifactPath); !os.IsNotExist(err) {
		t.Fatal("artifact should be removed when no sibling references it")
	}
	if _, err := os.Stat(filepath.Dir(artifactPath)); !os.IsNotExist(err) {
		t.Fatal("empty model directory should be pruned")
	}
}
