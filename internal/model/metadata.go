// Package model defines BitCLI's model catalog, metadata, and persistence contracts.
package model

import "time"

// State describes whether a model file is usable.
type State string

const (
	StateReady          State = "ready"
	StateDownloading    State = "downloading"
	StatePendingDelete  State = "pending_delete"
	StateFailed         State = "failed"
)

// ChatMode controls how the backend formats conversation turns.
// "" or "plain" = hand-rolled "User:/Assistant:" format (base models).
// "jinja"       = delegate to llama-cli --jinja (instruct/chat GGUFs).
type ChatMode = string

const (
	ChatModePlain ChatMode = "plain"
	ChatModeJinja ChatMode = "jinja"
)

// Artifact points at the concrete Hugging Face file used by a backend.
type Artifact struct {
	UserID        string
	CanonicalID   string
	Backend       string
	RepoID        string
	Revision      string
	Filename      string
	Quantization  string
	Family        string
	Parameters    string
	ContextLength int
	SHA256        string
	SizeBytes     int64
	ChatMode      ChatMode
}

// Model is the local metadata record for an installed model.
type Model struct {
	ID            string
	UserID        string
	CanonicalID   string
	Backend       string
	RepoID        string
	Revision      string
	Quantization  string
	Family        string
	Parameters    string
	ContextLength int
	Path          string
	State         State
	ChatMode      ChatMode
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// File stores metadata about a single artifact file on disk.
type File struct {
	ID        string
	ModelID   string
	Path      string
	Filename  string
	SizeBytes int64
	SHA256    string
	ETag      string
	State     State
	CreatedAt time.Time
	UpdatedAt time.Time
}

