// Package api exposes Ollama-compatible and OpenAI-compatible local HTTP endpoints.
package api

import "time"

const httpShutdownTimeout = 5 * time.Second

func (s *Server) registerRoutes() {
	s.engine.POST("/api/generate", s.handleOllamaGenerate)
	s.engine.POST("/api/chat", s.handleOllamaChat)
	s.engine.GET("/api/models", s.handleModels)
	s.engine.DELETE("/api/models/*id", s.handleDeleteModel)
	s.engine.GET("/api/version", s.handleVersion)
	s.engine.GET("/api/hardware", s.handleHardware)
	s.engine.POST("/v1/chat/completions", s.handleOpenAIChatCompletions)
}

