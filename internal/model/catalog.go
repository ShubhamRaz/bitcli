// Package model defines BitCLI's model catalog, metadata, and persistence contracts.
package model

import "strings"

// Catalog resolves friendly user-facing model names into concrete backend artifacts.
type Catalog struct {
	artifacts map[string]Artifact
}

// DefaultCatalog returns the initial BitNet-aware catalog.
func DefaultCatalog() Catalog {
	base := Artifact{
		UserID:        "microsoft/BitNet-b1.58-2B-4T",
		CanonicalID:   "microsoft/bitnet-b1.58-2B-4T-gguf",
		Backend:       "bitnet",
		RepoID:        "microsoft/bitnet-b1.58-2B-4T-gguf",
		Revision:      "main",
		Filename:      "ggml-model-i2_s.gguf",
		Quantization:  "i2_s",
		Family:        "BitNet b1.58",
		Parameters:    "2.4B",
		ContextLength: 4096,
		SizeBytes:     1190000000,
		ChatMode:      ChatModePlain,
	}

	// chat is the same GGUF used with llama-cli's --jinja flag so the embedded
	// chat template (Llama 3 instruct format) is applied automatically.
	chat := base
	chat.UserID = "microsoft/BitNet-b1.58-2B-4T-chat"
	chat.CanonicalID = "microsoft/bitnet-b1.58-2B-4T-gguf-chat"
	chat.ChatMode = ChatModeJinja

	baseKeys := []string{
		base.UserID,
		strings.ToLower(base.UserID),
		base.CanonicalID,
		strings.ToLower(base.CanonicalID),
		"microsoft/BitNet-b1.58-2B-4T-gguf",
		"microsoft/bitnet-b1.58-2b-4t",
	}
	chatKeys := []string{
		chat.UserID,
		strings.ToLower(chat.UserID),
		"microsoft/bitnet-b1.58-2b-4t-chat",
	}

	artifacts := make(map[string]Artifact, len(baseKeys)+len(chatKeys))
	for _, key := range baseKeys {
		artifacts[key] = base
	}
	for _, key := range chatKeys {
		artifacts[key] = chat
	}
	return Catalog{artifacts: artifacts}
}


// Resolve returns a concrete artifact for a user-supplied model ID.
func (c Catalog) Resolve(id string) (Artifact, bool) {
	id = strings.TrimSpace(id)
	if artifact, ok := c.artifacts[id]; ok {
		return artifact, true
	}
	artifact, ok := c.artifacts[strings.ToLower(id)]
	return artifact, ok
}

