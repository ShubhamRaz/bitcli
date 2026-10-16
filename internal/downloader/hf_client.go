// Package downloader handles Hugging Face Hub transfers and local verification.
package downloader

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/model"
)

// HFClient is a minimal Hugging Face Hub client for resolving and downloading model files.
type HFClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewHFClient creates a Hub client from BitCLI config.
func NewHFClient(cfg config.Config) *HFClient {
	base := strings.TrimRight(cfg.Download.Mirror, "/")
	token := ""
	if cfg.Download.TokenEnv != "" {
		token = os.Getenv(cfg.Download.TokenEnv)
	}
	return &HFClient{
		baseURL: base,
		token:   token,
		client:  &http.Client{Timeout: 0},
	}
}

// Resolve builds a file manifest and probes remote metadata with HEAD.
func (c *HFClient) Resolve(ctx context.Context, artifact model.Artifact) (Manifest, error) {
	fileURL, err := c.fileURL(artifact)
	if err != nil {
		return Manifest{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, fileURL, nil)
	if err != nil {
		return Manifest{}, err
	}
	c.authorize(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return Manifest{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return Manifest{}, fmt.Errorf("hugging face returned %s for %s", resp.Status, artifact.RepoID)
	}
	return Manifest{
		Artifact: artifact,
		URL:      fileURL,
		ETag:     strings.Trim(resp.Header.Get("ETag"), "\""),
		Size:     resp.ContentLength,
	}, nil
}

// NewDownloadRequest creates a GET request, optionally with a Range header.
func (c *HFClient) NewDownloadRequest(ctx context.Context, manifest Manifest, offset int64) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifest.URL, nil)
	if err != nil {
		return nil, err
	}
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
	}
	c.authorize(req)
	return req, nil
}

// Do runs an HTTP request with the client's retry-free primitive transport.
func (c *HFClient) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *HFClient) authorize(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	req.Header.Set("User-Agent", "bitcli/0.1")
}

func (c *HFClient) fileURL(artifact model.Artifact) (string, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", err
	}
	segments := []string{strings.Trim(artifact.RepoID, "/"), "resolve", artifact.Revision, artifact.Filename}
	escaped := make([]string, 0, len(segments)+1)
	for _, segment := range segments {
		if strings.Contains(segment, "/") {
			for _, part := range strings.Split(segment, "/") {
				escaped = append(escaped, url.PathEscape(part))
			}
			continue
		}
		escaped = append(escaped, url.PathEscape(segment))
	}
	base.Path = path.Join(append([]string{base.Path}, escaped...)...)
	return base.String(), nil
}

// RetryDelay returns a small exponential backoff bounded for CLI responsiveness.
func RetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := time.Duration(attempt*attempt) * time.Second
	if d > 15*time.Second {
		return 15 * time.Second
	}
	return d
}

