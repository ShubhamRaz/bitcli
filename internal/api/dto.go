// Package api exposes BitCLI REST API with OpenAI-compatible endpoints.
package api

import "github.com/bitcli/bitcli/internal/runtime"

// GenerateRequest is the request body for POST /api/generate.
type GenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  *bool                  `json:"stream,omitempty"`
	Options map[string]any         `json:"options,omitempty"`
}

// ChatRequest is the request body for POST /api/chat.
type ChatRequest struct {
	Model    string            `json:"model"`
	Messages []runtime.Message `json:"messages"`
	Stream   *bool             `json:"stream,omitempty"`
	Options  map[string]any    `json:"options,omitempty"`
}

// OpenAIChatCompletionRequest is the request body for POST /v1/chat/completions.
type OpenAIChatCompletionRequest struct {
	Model       string            `json:"model"`
	Messages    []runtime.Message `json:"messages"`
	Stream      bool              `json:"stream,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
	TopP        float64           `json:"top_p,omitempty"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
}

// ErrorResponse is a stable JSON error envelope.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError describes a request failure.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

