// Package api exposes BitCLI REST API with OpenAI-compatible endpoints.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/runtime"
	"github.com/bitcli/bitcli/internal/utils"
	"github.com/gin-gonic/gin"
)

func (s *Server) handleRoot(c *gin.Context) {
	c.String(http.StatusOK, "BitCLI is running")
}

func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) handleGenerate(c *gin.Context) {
	var req GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	m, err := s.ensureModel(c.Request.Context(), req.Model)
	if err != nil {
		writeError(c, statusForError(err), err)
		return
	}
	start := time.Now()
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
		writeError(c, statusForError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"model":          m.UserID,
		"created_at":     time.Now().UTC().Format(time.RFC3339Nano),
		"response":       text,
		"done":           true,
		"done_reason":    "stop",
		"total_duration": time.Since(start).Nanoseconds(),
	})
}

func (s *Server) handleChat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	m, err := s.ensureModel(c.Request.Context(), req.Model)
	if err != nil {
		writeError(c, statusForError(err), err)
		return
	}
	start := time.Now()
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
		writeError(c, statusForError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"model":          m.UserID,
		"created_at":     time.Now().UTC().Format(time.RFC3339Nano),
		"message":        gin.H{"role": "assistant", "content": text},
		"done":           true,
		"done_reason":    "stop",
		"total_duration": time.Since(start).Nanoseconds(),
	})
}

func (s *Server) handleModels(c *gin.Context) {
	models, err := s.models.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, err)
		return
	}
	items := make([]gin.H, 0, len(models))
	for _, m := range models {
		items = append(items, s.modelSummary(c.Request.Context(), m))
	}
	c.JSON(http.StatusOK, gin.H{"models": items})
}

func (s *Server) handleDeleteModel(c *gin.Context) {
	id := strings.TrimPrefix(c.Param("id"), "/")
	if err := s.removeModel(c, id); err != nil {
		writeError(c, statusForError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// handleOllamaDelete implements ollama's POST /api/delete ({"model": "..."}).
func (s *Server) handleOllamaDelete(c *gin.Context) {
	var req DeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		writeError(c, http.StatusBadRequest, utils.NewError(utils.CodeInvalidInput, "model is required"))
		return
	}
	if err := s.removeModel(c, req.Model); err != nil {
		writeError(c, statusForError(err), err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}

// removeModel deletes a model's cache files (when not shared with a sibling
// variant) and its metadata, then prunes its chat sessions.
func (s *Server) removeModel(c *gin.Context, id string) error {
	m, err := s.models.Local(c.Request.Context(), id)
	if err != nil {
		return err
	}
	siblings, err := s.models.List(c.Request.Context())
	if err != nil {
		return err
	}
	if err := s.cache.RemoveModel(m, siblings); err != nil {
		return err
	}
	if _, err := s.models.Remove(c.Request.Context(), id); err != nil {
		return err
	}
	return nil
}

// handlePull implements ollama's POST /api/pull: streams JSON-line status
// progress while downloading a model.
func (s *Server) handlePull(c *gin.Context) {
	var req PullRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		writeError(c, http.StatusBadRequest, utils.NewError(utils.CodeInvalidInput, "model is required"))
		return
	}
	artifact, err := s.models.Resolve(req.Model)
	if err != nil {
		writeError(c, statusForError(err), err)
		return
	}

	stream := shouldStream(req.Stream)
	if !stream {
		m, err := s.downloads.PullModel(c.Request.Context(), artifact, discardingWriter{})
		if err != nil {
			writeError(c, statusForError(err), err)
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "success", "path": m.Path})
		return
	}

	c.Header("Content-Type", "application/x-ndjson")
	c.Status(http.StatusOK)
	enc := json.NewEncoder(c.Writer)
	emit := func(v any) bool {
		if err := enc.Encode(v); err != nil {
			return false
		}
		c.Writer.Flush()
		return true
	}
	if !emit(gin.H{"status": "pulling manifest"}) {
		return
	}

	// Throttle progress updates to ~2 per second like ollama does.
	var mu sync.Mutex
	lastEmit := time.Time{}
	stopped := false
	onProgress := func(done, total int64) {
		mu.Lock()
		defer mu.Unlock()
		if stopped || time.Since(lastEmit) < 500*time.Millisecond {
			return
		}
		lastEmit = time.Now()
		stopped = !emit(gin.H{"status": "downloading", "completed": done, "total": total})
	}
	m, err := s.downloads.PullModelWithProgress(c.Request.Context(), artifact, discardingWriter{}, onProgress)
	if err != nil {
		mu.Lock()
		stopped = true
		mu.Unlock()
		_ = enc.Encode(gin.H{"error": err.Error()})
		c.Writer.Flush()
		return
	}
	mu.Lock()
	defer mu.Unlock()
	_ = enc.Encode(gin.H{"status": "verifying sha256 digest"})
	_ = enc.Encode(gin.H{"status": "success", "path": m.Path})
	c.Writer.Flush()
}

// handleShow implements ollama's /api/show: model details for an installed model.
func (s *Server) handleShow(c *gin.Context) {
	var req ShowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Model) == "" {
		writeError(c, http.StatusBadRequest, utils.NewError(utils.CodeInvalidInput, "model is required"))
		return
	}
	m, err := s.models.Local(c.Request.Context(), req.Model)
	if err != nil {
		writeError(c, statusForError(err), err)
		return
	}
	files, _ := s.models.Files(c.Request.Context(), m.ID)
	size := int64(0)
	if fi, statErr := os.Stat(m.Path); statErr == nil {
		size = fi.Size()
	}
	digest := ""
	if len(files) > 0 && files[0].SHA256 != "" {
		digest = "sha256:" + files[0].SHA256
	}
	c.JSON(http.StatusOK, gin.H{
		"license":   "mit",
		"modelfile": fmt.Sprintf("# model: %s\n# repo: %s@%s\n", m.UserID, m.RepoID, m.Revision),
		"parameters": fmt.Sprintf(
			"temperature %.2f\ntop_p %.2f\ntop_k %d\nmin_p %.2f\ncontext_length %d\n",
			s.cfg.Runtime.Temperature, s.cfg.Runtime.TopP, s.cfg.Runtime.TopK,
			s.cfg.Runtime.MinP, s.cfg.Runtime.ContextLength),
		"modified_at": m.UpdatedAt.Format(time.RFC3339Nano),
		"details": gin.H{
			"family":             m.Family,
			"families":           []string{m.Family},
			"parameter_size":     m.Parameters,
			"quantization_level": m.Quantization,
		},
		"model_info": gin.H{
			"general.architecture":    "bitnet",
			"general.parameter_count": m.Parameters,
			"bitnet.context_length":   m.ContextLength,
			"general.quantization":    m.Quantization,
		},
		"size":      size,
		"digest":    digest,
		"repo":      m.RepoID,
		"revision":  m.Revision,
		"file_name": m.Path,
	})
}

func (s *Server) handleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"version": s.version})
}

func (s *Server) handleHardware(c *gin.Context) {
	c.JSON(http.StatusOK, s.hardware.Report(c.Request.Context()))
}

func (s *Server) optionsFromMap(values map[string]any) runtime.Options {
	// Start from config so runtime.device (gpu offload) and runtime.min_p
	// apply to API requests exactly like they do for the CLI.
	opts := runtime.OptionsFromConfig(s.cfg)
	for key, value := range values {
		switch key {
		case "temperature":
			opts.Temperature = number(value, opts.Temperature)
		case "top_p":
			opts.TopP = number(value, opts.TopP)
		case "top_k":
			opts.TopK = int(number(value, float64(opts.TopK)))
		case "min_p":
			opts.MinP = number(value, opts.MinP)
		case "num_thread", "threads":
			opts.Threads = int(number(value, float64(opts.Threads)))
		case "num_gpu", "gpu_layers":
			opts.GPULayers = int(number(value, float64(opts.GPULayers)))
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

type discardingWriter struct{}

func (discardingWriter) Write(p []byte) (int, error) { return len(p), nil }

// modelSummary builds an ollama-shaped model list entry with real size and digest.
func (s *Server) modelSummary(ctx context.Context, m model.Model) gin.H {
	size := int64(0)
	if fi, err := os.Stat(m.Path); err == nil {
		size = fi.Size()
	}
	digest := ""
	if files, err := s.models.Files(ctx, m.ID); err == nil && len(files) > 0 && files[0].SHA256 != "" {
		digest = "sha256:" + files[0].SHA256
	}
	return gin.H{
		"name":        m.UserID,
		"model":       m.UserID,
		"modified_at": m.UpdatedAt.Format(time.RFC3339Nano),
		"size":        size,
		"digest":      digest,
		"details": gin.H{
			"family":             m.Family,
			"parameter_size":     m.Parameters,
			"quantization_level": m.Quantization,
		},
	}
}
