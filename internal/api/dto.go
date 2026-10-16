// Package api exposes BitCLI REST API with OpenAI-compatible endpoints.
package api

import "github.com/bitcli/bitcli/internal/runtime"

// GenerateRequest is the request body for POST /api/generate.
type GenerateRequest struct {
	Model   string         `json:"model"`
	Prompt  string         `json:"prompt"`
	Stream  *bool          `json:"stream,omitempty"`
	Options map[string]any `json:"options,omitempty"`
}

// ChatRequest is the request body for POST /api/chat.
type ChatRequest struct {
	Model    string            `json:"model"`
	Messages []runtime.Message `json:"messages"`
	Stream   *bool             `json:"stream,omitempty"`
	Options  map[string]any    `json:"options,omitempty"`
}

// OpenAIChatCompletionRequest is the request body for POST /v1/chat/completions.
// Sampling fields are pointers so an explicit 0 (e.g. temperature 0 for greedy
// decoding) is distinguishable from an absent field.
type OpenAIChatCompletionRequest struct {
	Model       string            `json:"model"`
	Messages    []runtime.Message `json:"messages"`
	Stream      bool              `json:"stream,omitempty"`
	Temperature *float64          `json:"temperature,omitempty"`
	TopP        *float64          `json:"top_p,omitempty"`
	MaxTokens   *int              `json:"max_tokens,omitempty"`
}

// OpenAICompletionRequest is the request body for POST /v1/completions.
type OpenAICompletionRequest struct {
	Model       string   `json:"model"`
	Prompt      any      `json:"prompt"`
	Stream      bool     `json:"stream,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Echo        bool     `json:"echo,omitempty"`
}

// PullRequest is the request body for POST /api/pull (ollama-compatible).
type PullRequest struct {
	Model  string `json:"model"`
	Stream *bool  `json:"stream,omitempty"`
}

// DeleteRequest is the request body for POST /api/delete (ollama-compatible).
type DeleteRequest struct {
	Model string `json:"model"`
}

// ShowRequest is the request body for /api/show (ollama-compatible).
type ShowRequest struct {
	Model string `json:"model"`
}

// ErrorResponse is a stable JSON error envelope.
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// APIError describes a request failure. Type mirrors OpenAI's error shape
// ("invalid_request_error" / "server_error") so OpenAI clients can parse it.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}
