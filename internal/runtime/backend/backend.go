// Package backend defines the extension contract for external inference backends.
package backend

import (
	"context"

	"github.com/bitcli/bitcli/internal/model"
)

// Status describes whether a backend can currently run models.
type Status struct {
	ID       string
	Ready    bool
	Path     string
	Version  string
	Message  string
}

// PrepareOptions controls backend-specific model setup.
type PrepareOptions struct {
	Quantization string
}

// Backend executes model operations through an external runtime.
type Backend interface {
	ID() string
	Detect(ctx context.Context) (Status, error)
	Prepare(ctx context.Context, m model.Model, opts PrepareOptions) error
	Generate(ctx context.Context, m model.Model, req GenerateRequest) (<-chan TokenEvent, <-chan error)
	Chat(ctx context.Context, m model.Model, req ChatRequest) (<-chan TokenEvent, <-chan error)
	Stop(ctx context.Context, sessionID string) error
	Version(ctx context.Context) (string, error)
}
