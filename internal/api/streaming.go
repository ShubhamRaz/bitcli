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

func streamSSE(c *gin.Context, events <-chan runtime.TokenEvent, errs <-chan error, build func(runtime.TokenEvent) any) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)
	done := false
	for ev := range events {
		if ev.Done {
			done = true
		}
		payload, err := json.Marshal(build(ev))
		if err != nil {
			return
		}
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		c.Writer.Flush()
	}
	if err := <-errs; err != nil {
		payload, _ := json.Marshal(map[string]any{"error": err.Error()})
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
		return
	}
	if !done {
		payload, _ := json.Marshal(build(runtime.TokenEvent{Done: true, CreatedAt: time.Now().UTC()}))
		fmt.Fprintf(c.Writer, "data: %s\n\n", payload)
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
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
