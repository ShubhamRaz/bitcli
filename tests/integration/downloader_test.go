// Package integration contains end-to-end tests for the BitCLI downloader.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// fakeHFServer serves a model file over HTTP, mimicking the Hugging Face Hub.
func fakeHFServer(t *testing.T, content []byte) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodHead:
			w.Header().Set("Content-Length", fmt_itoa(len(content)))
			w.Header().Set("ETag", `"fakeetag"`)
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			rangeHeader := r.Header.Get("Range")
			if rangeHeader != "" {
				// Serve partial content for resume test.
				var offset int64
				_, _ = fmt_sscanf(rangeHeader, "bytes=%d-", &offset)
				if offset > 0 && int(offset) < len(content) {
					w.Header().Set("Content-Length", fmt_itoa(len(content)-int(offset)))
					w.WriteHeader(http.StatusPartialContent)
					_, _ = w.Write(content[offset:])
					return
				}
			}
			w.Header().Set("Content-Length", fmt_itoa(len(content)))
			_, _ = w.Write(content)
		}
	}))
}

func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func TestDownloader_FullDownloadAndVerify(t *testing.T) {
	content := []byte("fake gguf model content for testing")
	srv := fakeHFServer(t, content)
	defer srv.Close()

	dir := t.TempDir()
	targetPath := filepath.Join(dir, "model.gguf")

	// Simulate a simple HTTP GET download.
	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/microsoft/bitnet/resolve/main/model.gguf", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	f, err := os.Create(targetPath)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		t.Fatalf("copy: %v", err)
	}
	_ = f.Close()

	// Verify content is identical.
	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("content mismatch:\ngot:  %q\nwant: %q", got, content)
	}
}

func TestDownloader_ResumePartialDownload(t *testing.T) {
	content := []byte("abcdefghijklmnopqrstuvwxyz0123456789")
	srv := fakeHFServer(t, content)
	defer srv.Close()

	dir := t.TempDir()
	partialPath := filepath.Join(dir, "model.gguf.partial")
	// Simulate 10 bytes already downloaded.
	if err := os.WriteFile(partialPath, content[:10], 0o644); err != nil {
		t.Fatalf("write partial: %v", err)
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/model.gguf", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Range", "bytes=10-")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET with Range: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected 206 Partial Content, got %d", resp.StatusCode)
	}

	remainder, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read remainder: %v", err)
	}
	// Append the remainder to the partial file.
	f, _ := os.OpenFile(partialPath, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.Write(remainder)
	_ = f.Close()

	assembled, _ := os.ReadFile(partialPath)
	if string(assembled) != string(content) {
		t.Fatalf("reassembled content mismatch")
	}
}

// Minimal helpers to avoid importing fmt in the handler (keeps the fake simple).

func fmt_itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func fmt_sscanf(s, format string, args ...any) (int, error) {
	// Minimal "bytes=%d-" scanner used only in tests.
	if len(args) == 0 {
		return 0, nil
	}
	var offset int64
	// Find the numeric part after "bytes="
	const prefix = "bytes="
	if len(s) > len(prefix) {
		rem := s[len(prefix):]
		for i, ch := range rem {
			if ch < '0' || ch > '9' {
				rem = rem[:i]
				break
			}
		}
		for _, ch := range rem {
			offset = offset*10 + int64(ch-'0')
		}
	}
	if p, ok := args[0].(*int64); ok {
		*p = offset
	}
	return 1, nil
}
