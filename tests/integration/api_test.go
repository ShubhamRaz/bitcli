// Package integration tests the BitCLI HTTP API endpoints.
package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bitcli/bitcli/internal/api"
	"github.com/bitcli/bitcli/internal/cache"
	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/database"
	"github.com/bitcli/bitcli/internal/downloader"
	"github.com/bitcli/bitcli/internal/hardware"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/process"
	bitruntime "github.com/bitcli/bitcli/internal/runtime"
	"github.com/bitcli/bitcli/internal/runtime/backend"
	"github.com/bitcli/bitcli/internal/runtime/backend/bitnet"
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// openIntegrationDB creates an in-memory SQLite DB for integration tests.
func openIntegrationDB(t *testing.T) *database.DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.ExecContext(context.Background(),
		"PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	db := &database.DB{SQL: sqlDB}
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func buildTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	cfg := config.DefaultConfig()
	paths, err := config.DefaultPaths()
	if err != nil {
		t.Fatalf("default paths: %v", err)
	}

	db := openIntegrationDB(t)
	modelRepo := database.NewModelRepository(db)
	modelSvc := model.NewService(modelRepo, model.NewResolver(model.DefaultCatalog()))
	cacheSvc := cache.NewService(paths)
	downloadRepo := database.NewDownloadRepository(db)
	downloadSvc := downloader.NewService(cfg, cacheSvc, modelSvc, downloadRepo)

	runner := process.NewRunner()
	reg := backend.NewRegistry()
	reg.Register(bitnet.New(cfg, paths, runner, zap.NewNop()))

	runtimeSvc := bitruntime.NewService(modelSvc, reg)

	deps := api.Dependencies{
		Config:    cfg,
		Models:    modelSvc,
		Downloads: downloadSvc,
		Runtime:   runtimeSvc,
		Cache:     cacheSvc,
		Hardware:  hardware.NewService(),
		Version:   "0.1.0-test",
		Logger:    zap.NewNop(),
	}
	srv := api.NewServer(deps)
	return httptest.NewServer(srv.Handler())
}

func TestAPIIntegration_GetVersion(t *testing.T) {
	srv := buildTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/version")
	if err != nil {
		t.Fatalf("GET /api/version: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["version"] != "0.1.0-test" {
		t.Fatalf("unexpected version: %v", body["version"])
	}
}

func TestAPIIntegration_GetModels_Empty(t *testing.T) {
	srv := buildTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/models")
	if err != nil {
		t.Fatalf("GET /api/models: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	models, ok := body["models"]
	if !ok {
		t.Fatal("response should contain 'models' key")
	}
	if models == nil {
		t.Fatal("'models' should not be nil")
	}
}

func TestAPIIntegration_GetHardware(t *testing.T) {
	srv := buildTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/hardware")
	if err != nil {
		t.Fatalf("GET /api/hardware: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestAPIIntegration_PostGenerate_UnknownModel_Returns4xx(t *testing.T) {
	srv := buildTestServer(t)
	defer srv.Close()

	payload, _ := json.Marshal(map[string]any{
		"model":  "unknown/model",
		"prompt": "Hello",
		"stream": false,
	})
	resp, err := http.Post(srv.URL+"/api/generate", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /api/generate: %v", err)
	}
	defer resp.Body.Close()

	// ollama returns 404 for unknown models.
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown model, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error envelope, got %v", body)
	}
	if errObj["type"] == "" {
		t.Fatal("OpenAI-style error envelope should include a type field")
	}
}

func TestAPIIntegration_OllamaCompatibilityEndpoints(t *testing.T) {
	srv := buildTestServer(t)
	defer srv.Close()

	// GET / (liveness, like ollama's "Ollama is running").
	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / expected 200, got %d", resp.StatusCode)
	}

	// GET /health
	resp, err = http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /health expected 200, got %d", resp.StatusCode)
	}

	// GET /api/tags (ollama's model list endpoint).
	resp, err = http.Get(srv.URL + "/api/tags")
	if err != nil {
		t.Fatalf("GET /api/tags: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/tags expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode /api/tags: %v", err)
	}
	if _, ok := body["models"]; !ok {
		t.Fatal("/api/tags should contain 'models' key")
	}
}

func TestAPIIntegration_OpenAIModelsEndpoint(t *testing.T) {
	srv := buildTestServer(t)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/models")
	if err != nil {
		t.Fatalf("GET /v1/models: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /v1/models expected 200, got %d", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode /v1/models: %v", err)
	}
	if body["object"] != "list" {
		t.Fatalf("/v1/models should be an OpenAI list object, got %v", body["object"])
	}
}

func TestAPIIntegration_ShowUnknownModelReturns404(t *testing.T) {
	srv := buildTestServer(t)
	defer srv.Close()

	payload, _ := json.Marshal(map[string]any{"model": "nope/none"})
	resp, err := http.Post(srv.URL+"/api/show", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("POST /api/show: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown model, got %d", resp.StatusCode)
	}
}

func TestAPIIntegration_OptionsPreflightAllowed(t *testing.T) {
	srv := buildTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/api/generate", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("OPTIONS request: %v", err)
	}
	defer resp.Body.Close()
	// CORS preflight should return 204 No Content
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204 for OPTIONS preflight, got %d", resp.StatusCode)
	}
}
