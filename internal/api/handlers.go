// Package api exposes BitCLI REST API with OpenAI-compatible endpoints.
package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/runtime"
	"github.com/gin-gonic/gin"
)

func (s *Server) handleGenerate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	m, err := s.ensureModel(c.Request.Context(), req.Model)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	runReq := runtime.GenerateRequest{ModelID: m.UserID, Prompt: req.Prompt, Options: s.optionsFromMap(req.Options)}
	events, errs := s.runtime.Generate(c.Request.Context(), runReq)
	if shouldStream(req.Stream) {
		streamJSONLines(c, events, errs, func(ev runtime.TokenEvent) any {
			return map[string]any{
				"model":      m.UserID,
				"created_at": ev.CreatedAt.Format(time.RFC3339Nano),
				"response":   ev.Text,
				"done":       ev.Done,
			}
		})
		return
	}
	text, err := collect(events, errs)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"model": m.UserID, "response": text, "done": true})
}

func (s *Server) handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	m, err := s.ensureModel(c.Request.Context(), req.Model)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	runReq := runtime.ChatRequest{ModelID: m.UserID, Messages: req.Messages, Options: s.optionsFromMap(req.Options)}
	events, errs := s.runtime.Chat(c.Request.Context(), runReq)
	if shouldStream(req.Stream) {
		streamJSONLines(c, events, errs, func(ev runtime.TokenEvent) any {
			return map[string]any{
				"model":      m.UserID,
				"created_at": ev.CreatedAt.Format(time.RFC3339Nano),
				"message":    map[string]any{"role": "assistant", "content": ev.Text},
				"done":       ev.Done,
			}
		})
		return
	}
	text, err := collect(events, errs)
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"model": m.UserID, "message": gin.H{"role": "assistant", "content": text}, "done": true})
}

func (s *Server) handleModels(c *gin.Context) {
	models, err := s.models.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	items := make([]gin.H, 0, len(models))
	for _, m := range models {
		items = append(items, modelSummary(m))
	}
	c.JSON(http.StatusOK, gin.H{"models": items})
}

func (s *Server) handleDeleteModel(c *gin.Context) {
	id := strings.TrimPrefix(c.Param("id"), "/")
	m, err := s.models.Local(c.Request.Context(), id)
	if err != nil {
		writeError(c, http.StatusNotFound, err)
		return
	}
	if err := s.cache.RemoveModel(m); err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	if _, err := s.models.Remove(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) handleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": s.version})
}

func (s *Server) handleHardware(c *gin.Context) {
	c.JSON(http.StatusOK, s.hardware.Report(c.Request.Context()))
}

func (s *Server) optionsFromMap(values map[string]any) runtime.Options {
	opts := runtime.Options{
		Temperature:   s.cfg.Runtime.Temperature,
		TopP:          s.cfg.Runtime.TopP,
		TopK:          s.cfg.Runtime.TopK,
		Threads:       s.cfg.Runtime.Threads,
		GPULayers:     s.cfg.Runtime.GPULayers,
		ContextLength: s.cfg.Runtime.ContextLength,
		MaxTokens:     s.cfg.Runtime.MaxTokens,
	}
	for key, value := range values {
		switch key {
		case "temperature":
			opts.Temperature = number(value, opts.Temperature)
		case "top_p":
			opts.TopP = number(value, opts.TopP)
		case "top_k":
			opts.TopK = int(number(value, float64(opts.TopK)))
		case "num_thread", "threads":
			opts.Threads = int(number(value, float64(opts.Threads)))
		case "num_ctx", "context_length":
			opts.ContextLength = int(number(value, float64(opts.ContextLength)))
		case "num_predict", "max_tokens":
			opts.MaxTokens = int(number(value, float64(opts.MaxTokens)))
		}
	}
	return opts
}

func shouldStream(value *bool) bool {
	return value == nil || *value
}

func number(value any, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case jsonNumber:
		f, err := v.Float64()
		if err == nil {
			return f
		}
	}
	return fallback
}

type jsonNumber interface {
	Float64() (float64, error)
}

func modelSummary(m model.Model) gin.H {
	return gin.H{
		"name":       m.UserID,
		"model":      m.UserID,
		"modified_at": m.UpdatedAt.Format(time.RFC3339Nano),
		"size":       0,
		"digest":     "",
		"details": gin.H{
			"family":             m.Family,
			"parameter_size":     m.Parameters,
			"quantization_level": m.Quantization,
		},
	}
}
