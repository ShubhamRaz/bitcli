// Package api exposes Ollama-compatible and OpenAI-compatible local HTTP endpoints.
package api

import (
	"net/http"
	"time"

	"github.com/bitcli/bitcli/internal/runtime"
	"github.com/bitcli/bitcli/internal/utils"
	"github.com/gin-gonic/gin"
)

func (s *Server) handleOpenAIChatCompletions(c *gin.Context) {
	var req OpenAIChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	m, err := s.ensureModel(c.Request.Context(), req.Model)
	if err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	opts := runtime.Options{
		Temperature:   s.cfg.Runtime.Temperature,
		TopP:          s.cfg.Runtime.TopP,
		TopK:          s.cfg.Runtime.TopK,
		Threads:       s.cfg.Runtime.Threads,
		GPULayers:     s.cfg.Runtime.GPULayers,
		ContextLength: s.cfg.Runtime.ContextLength,
		MaxTokens:     s.cfg.Runtime.MaxTokens,
	}
	if req.Temperature > 0 {
		opts.Temperature = req.Temperature
	}
	if req.TopP > 0 {
		opts.TopP = req.TopP
	}
	if req.MaxTokens > 0 {
		opts.MaxTokens = req.MaxTokens
	}
	runReq := runtime.ChatRequest{ModelID: m.UserID, Messages: req.Messages, Options: opts}
	events, errs := s.runtime.Chat(c.Request.Context(), runReq)
	if req.Stream {
		streamSSE(c, events, errs, func(ev runtime.TokenEvent) any {
			return openAIChunk(m.UserID, ev.Text, ev.Done)
		})
		return
	}
	text, err := collect(events, errs)
	if err != nil {
		writeError(c, http.StatusInternalServerError, utils.WrapError(utils.CodeInternal, "chat completion failed", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":      utils.NewID("chatcmpl"),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   m.UserID,
		"choices": []gin.H{{
			"index":         0,
			"finish_reason": "stop",
			"message":       gin.H{"role": "assistant", "content": text},
		}},
	})
}

func openAIChunk(modelID, text string, done bool) gin.H {
	choice := gin.H{
		"index": 0,
		"delta": gin.H{"content": text},
	}
	if done {
		choice["finish_reason"] = "stop"
	}
	return gin.H{
		"id":      utils.NewID("chatcmpl"),
		"object":  "chat.completion.chunk",
		"created": time.Now().Unix(),
		"model":   modelID,
		"choices": []gin.H{choice},
	}
}

