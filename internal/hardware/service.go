// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
package hardware

import "context"

// Service exposes hardware detection to CLI and API callers.
type Service struct {
	detector *Detector
}

// NewService creates a hardware service.
func NewService() *Service {
	return &Service{detector: NewDetector()}
}

// Report returns a best-effort hardware report.
func (s *Service) Report(ctx context.Context) Report {
	return s.detector.Detect(ctx)
}

