// Package api exposes BitCLI REST API with OpenAI-compatible endpoints.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bitcli/bitcli/internal/runtime"
	"github.com/bitcli/bitcli/internal/utils"
	"github.com/gin-gonic/gin"
)

// approxTokens estimates token counts (~4 chars per token) for OpenAI-style
// usage reporting. The bitnet.cpp backend does not expose tokenizers.
func approxTokens(text string) int {
	if text == "" {
		return 0
	}
	return len(text) / 4
}

func (s *Server) handleOpenAIChatCompletions(c *gin.Context) {
	var req OpenAIChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	m, err := s.ensureModel(c.Request.Context(), req.Model)
	if err != nil {
		writeError(c, statusForError(err), err)
		return
	}
	opts := runtime.OptionsFromConfig(s.cfg)
	if req.Temperature != nil {
		opts.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		opts.TopP = *req.TopP
	}
	if req.MaxTokens != nil {
		opts.MaxTokens = *req.MaxTokens
	}
	runReq := runtime.ChatRequest{ModelID: m.UserID, Messages: req.Messages, Options: opts}
	events, errs := s.runtime.Chat(c.Request.Context(), runReq)

	// One shared id per completion, per the OpenAI streaming contract.
	completionID := utils.NewID("chatcmpl")
	created := time.Now().Unix()
	promptTokens := 0
	for _, msg := range req.Messages {
		promptTokens += approxTokens(msg.Content)
	}

	if req.Stream {
		streamSSE(c, events, errs, func(ev runtime.TokenEvent, first bool, last bool) any {
			return openAIChatChunk(completionID, created, m.UserID, ev.Text, ev.Done, first)
		})
		return
	}
	text, err := collect(events, errs)
	if err != nil {
		writeError(c, statusForError(err), utils.WrapError(utils.CodeInternal, "chat completion failed", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":      completionID,
		"object":  "chat.completion",
		"created": created,
		"model":   m.UserID,
		"choices": []gin.H{{
			"index":         0,
			"finish_reason": "stop",
			"message":       gin.H{"role": "assistant", "content": text},
		}},
		"usage": gin.H{
			"prompt_tokens":     promptTokens,
			"completion_tokens": approxTokens(text),
			"total_tokens":      promptTokens + approxTokens(text),
		},
	})
}

// openAIChatChunk builds one SSE chunk. All chunks of a completion share the
// same id; the first carries the assistant role delta.
func openAIChatChunk(id string, created int64, modelID, text string, done, first bool) gin.H {
	delta := gin.H{}
	if first {
		delta["role"] = "assistant"
	}
	delta["content"] = text
	choice := gin.H{
		"index": 0,
		"delta": delta,
	}
	if done {
		choice["finish_reason"] = "stop"
	} else {
		choice["finish_reason"] = nil
	}
	return gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   modelID,
		"choices": []gin.H{choice},
	}
}

// handleOpenAIModels implements GET /v1/models (OpenAI model list format).
func (s *Server) handleOpenAIModels(c *gin.Context) {
	models, err := s.models.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	data := make([]gin.H, 0, len(models))
	for _, m := range models {
		data = append(data, gin.H{
			"id":       m.UserID,
			"object":   "model",
			"created":  m.CreatedAt.Unix(),
			"owned_by": "bitcli",
		})
	}
	c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
}

// handleOpenAICompletions implements POST /v1/completions (legacy text
// completions) on top of the generate pipeline.
func (s *Server) handleOpenAICompletions(c *gin.Context) {
	var req OpenAICompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	prompt := normalizeCompletionPrompt(req.Prompt)
	if strings.TrimSpace(prompt) == "" {
		writeError(c, http.StatusBadRequest, utils.NewError(utils.CodeInvalidInput, "prompt is required"))
		return
	}
	m, err := s.ensureModel(c.Request.Context(), req.Model)
	if err != nil {
		writeError(c, statusForError(err), err)
		return
	}
	opts := runtime.OptionsFromConfig(s.cfg)
	if req.Temperature != nil {
		opts.Temperature = *req.Temperature
	}
	if req.TopP != nil {
		opts.TopP = *req.TopP
	}
	if req.MaxTokens != nil {
		opts.MaxTokens = *req.MaxTokens
	}
	runReq := runtime.GenerateRequest{ModelID: m.UserID, Prompt: prompt, Options: opts}
	events, errs := s.runtime.Generate(c.Request.Context(), runReq)

	completionID := utils.NewID("cmpl")
	created := time.Now().Unix()
	promptTokens := approxTokens(prompt)

	if req.Stream {
		streamSSE(c, events, errs, func(ev runtime.TokenEvent, first bool, last bool) any {
			choice := gin.H{
				"index":         0,
				"text":          ev.Text,
				"finish_reason": nil,
			}
			if ev.Done {
				choice["finish_reason"] = "stop"
			}
			return gin.H{
				"id":      completionID,
				"object":  "text_completion.chunk",
				"created": created,
				"model":   m.UserID,
				"choices": []gin.H{choice},
			}
		})
		return
	}
	text, err := collect(events, errs)
	if err != nil {
		writeError(c, statusForError(err), utils.WrapError(utils.CodeInternal, "completion failed", err))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":      completionID,
		"object":  "text_completion",
		"created": created,
		"model":   m.UserID,
		"choices": []gin.H{{
			"index":         0,
			"text":          text,
			"finish_reason": "stop",
		}},
		"usage": gin.H{
			"prompt_tokens":     promptTokens,
			"completion_tokens": approxTokens(text),
			"total_tokens":      promptTokens + approxTokens(text),
		},
	})
}

// normalizeCompletionPrompt flattens the OpenAI prompt field which may be a
// string, a list of strings, or a list of token ints.
func normalizeCompletionPrompt(v any) string {
	switch p := v.(type) {
	case string:
		return p
	case []any:
		parts := make([]string, 0, len(p))
		for _, item := range p {
			switch t := item.(type) {
			case string:
				parts = append(parts, t)
			case float64:
				// Token id; render numerically as a fallback.
				parts = append(parts, strconv.Itoa(int(t)))
			}
		}
		return strings.Join(parts, "")
	case nil:
		return ""
	default:
		if b, err := json.Marshal(v); err == nil {
			return string(b)
		}
		return ""
	}
}
