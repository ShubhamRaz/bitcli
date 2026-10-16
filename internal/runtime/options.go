// Package runtime coordinates backend-neutral model execution.
package runtime

import "github.com/bitcli/bitcli/internal/runtime/backend"

// Options are common generation settings accepted by BitCLI.
type Options = backend.Options

// Message is a chat message in a backend-neutral representation.
type Message = backend.Message

// GenerateRequest asks a backend to generate text from a prompt.
type GenerateRequest = backend.GenerateRequest

// ChatRequest asks a backend to continue a chat transcript.
type ChatRequest = backend.ChatRequest

// TokenEvent is one streamed backend event.
type TokenEvent = backend.TokenEvent
