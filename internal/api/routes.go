// Package api exposes BitCLI REST API with OpenAI-compatible endpoints.
package api

import "time"

const httpShutdownTimeout = 5 * time.Second

func (s *Server) registerRoutes() {
	// Liveness / discovery (like ollama's GET / text response).
	s.engine.GET("/", s.handleRoot)
	s.engine.GET("/health", s.handleHealth)

	// Native API.
	s.engine.POST("/api/generate", s.handleGenerate)
	s.engine.POST("/api/chat", s.handleChat)
	s.engine.GET("/api/models", s.handleModels)
	s.engine.DELETE("/api/models/*id", s.handleDeleteModel)
	s.engine.GET("/api/version", s.handleVersion)
	s.engine.GET("/api/hardware", s.handleHardware)

	// ollama-compatible aliases so ollama clients work unchanged.
	s.engine.GET("/api/tags", s.handleModels) // ollama's primary model list endpoint
	s.engine.POST("/api/pull", s.handlePull)
	s.engine.POST("/api/delete", s.handleOllamaDelete)
	s.engine.GET("/api/show", s.handleShow)
	s.engine.POST("/api/show", s.handleShow)

	// OpenAI-compatible endpoints.
	s.engine.POST("/v1/chat/completions", s.handleOpenAIChatCompletions)
	s.engine.POST("/v1/completions", s.handleOpenAICompletions)
	s.engine.GET("/v1/models", s.handleOpenAIModels)
}
