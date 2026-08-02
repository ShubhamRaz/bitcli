// Package backend defines the extension contract for external inference backends.
package backend

import "time"

// Options are common generation settings accepted by BitCLI.
type Options struct {
	Temperature   float64
	TopP          float64
	TopK          int
	Threads       int
	GPULayers     int
	ContextLength int
	MaxTokens     int
}

// Message is a chat message in a backend-neutral representation.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GenerateRequest asks a backend to generate text from a prompt.
type GenerateRequest struct {
	ModelID string
	Prompt  string
	Options Options
}

// ChatRequest asks a backend to continue a chat transcript.
type ChatRequest struct {
	ModelID  string
	Messages []Message
	Options  Options
}

// TokenEvent is one streamed backend event.
type TokenEvent struct {
	Text      string
	Done      bool
	Error     string
	CreatedAt time.Time
}
