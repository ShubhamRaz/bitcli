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

