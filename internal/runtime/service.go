// Package runtime coordinates backend-neutral model execution.
package runtime

import (
	"context"

	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/runtime/backend"
)

// ModelLookup loads locally installed model metadata.
type ModelLookup interface {
	Local(ctx context.Context, id string) (model.Model, error)
}

// Service coordinates generation through a selected backend.
type Service struct {
	models   ModelLookup
	backends *backend.Registry
}

// NewService creates a runtime service from local model and backend dependencies.
func NewService(models ModelLookup, backends *backend.Registry) *Service {
	return &Service{models: models, backends: backends}
}

// Generate starts text generation for a locally installed model.
func (s *Service) Generate(ctx context.Context, req GenerateRequest) (<-chan TokenEvent, <-chan error) {
	m, err := s.models.Local(ctx, req.ModelID)
	if err != nil {
		return failedStream(err)
	}
	b, err := s.backends.Get(m.Backend)
	if err != nil {
		return failedStream(err)
	}
	if err := b.Prepare(ctx, m, backend.PrepareOptions{Quantization: m.Quantization}); err != nil {
		return failedStream(err)
	}
	return b.Generate(ctx, m, req)
}

// Chat starts chat generation for a locally installed model.
func (s *Service) Chat(ctx context.Context, req ChatRequest) (<-chan TokenEvent, <-chan error) {
	m, err := s.models.Local(ctx, req.ModelID)
	if err != nil {
		return failedStream(err)
	}
	b, err := s.backends.Get(m.Backend)
	if err != nil {
		return failedStream(err)
	}
	if err := b.Prepare(ctx, m, backend.PrepareOptions{Quantization: m.Quantization}); err != nil {
		return failedStream(err)
	}
	return b.Chat(ctx, m, req)
}

func failedStream(err error) (<-chan TokenEvent, <-chan error) {
	events := make(chan TokenEvent, 1)
	errs := make(chan error, 1)
	errs <- err
	close(events)
	close(errs)
	return events, errs
}

