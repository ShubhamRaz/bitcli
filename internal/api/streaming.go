// Package api exposes BitCLI REST API with OpenAI-compatible endpoints.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bitcli/bitcli/internal/runtime"
	"github.com/gin-gonic/gin"
)

func streamJSONLines(c *gin.Context, events <-chan runtime.TokenEvent, errs <-chan error, build func(runtime.TokenEvent) any) {
	c.Header("Content-Type", "application/x-ndjson")
	c.Status(http.StatusOK)
	enc := json.NewEncoder(c.Writer)
	done := false
	for ev := range events {
		if ev.Done {
			done = true
		}
		if err := enc.Encode(build(ev)); err != nil {
			return
		}
		c.Writer.Flush()
	}
	if err := <-errs; err != nil {
		_ = enc.Encode(map[string]any{"error": err.Error(), "done": true})
		return
	}
	if !done {
		_ = enc.Encode(build(runtime.TokenEvent{Done: true, CreatedAt: time.Now().UTC()}))
	}
}

// sseBuilder converts a token event into an SSE payload. first is true for
// the first emitted chunk, last is true for the terminal (done) chunk.
type sseBuilder func(ev runtime.TokenEvent, first, last bool) any

func streamSSE(c *gin.Context, events <-chan runtime.TokenEvent, errs <-chan error, build sseBuilder) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(http.StatusOK)
	first := true
	done := false
	write := func(payload any) bool {
		data, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", data); err != nil {
			return false
		}
		c.Writer.Flush()
		return true
	}
	for ev := range events {
		last := ev.Done
		if ev.Done {
			done = true
		}
		if !write(build(ev, first, last)) {
			return
		}
		first = false
	}
	if err := <-errs; err != nil {
		payload, _ := json.Marshal(map[string]any{"error": err.Error()})
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		c.Writer.Flush()
		return
	}
	if !done {
		if !write(build(runtime.TokenEvent{Done: true, CreatedAt: time.Now().UTC()}, first, true)) {
			return
		}
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
	c.Writer.Flush()
}

func collect(events <-chan runtime.TokenEvent, errs <-chan error) (string, error) {
	var text string
	for ev := range events {
		text += ev.Text
	}
	if err := <-errs; err != nil {
		return "", err
	}
	return text, nil
}
