// Package downloader handles Hugging Face Hub transfers and local verification.
package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bitcli/bitcli/internal/cache"
	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/database"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/utils"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// Service downloads, resumes, verifies, and registers model artifacts.
type Service struct {
	cfg       config.Config
	cache     *cache.Service
	models    *model.Service
	downloads *database.DownloadRepository
	hf        *HFClient
}

// NewService creates a downloader service from infrastructure dependencies.
func NewService(cfg config.Config, cacheSvc *cache.Service, modelSvc *model.Service, downloads *database.DownloadRepository) *Service {
	return &Service{
		cfg:       cfg,
		cache:     cacheSvc,
		models:    modelSvc,
		downloads: downloads,
		hf:        NewHFClient(cfg),
	}
}

// PullModel downloads a model artifact if needed and saves local metadata.
func (s *Service) PullModel(ctx context.Context, artifact model.Artifact, progressOut io.Writer) (model.Model, error) {
	if err := s.cache.Ensure(); err != nil {
		return model.Model{}, err
	}
	target := s.cache.Layout().ModelFile(artifact)
	if _, err := os.Stat(target); err == nil {
		return s.register(ctx, artifact, target, "", artifact.SizeBytes)
	}

	manifest, err := s.hf.Resolve(ctx, artifact)
	if err != nil {
		return model.Model{}, err
	}
	if manifest.Size > 0 {
		artifact.SizeBytes = manifest.Size
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return model.Model{}, err
	}

	partial := s.cache.Layout().PartialFile(artifact)
	rec := database.DownloadRecord{
		ID:          utils.NewID("dl"),
		RepoID:      artifact.RepoID,
		Revision:    artifact.Revision,
		Filename:    artifact.Filename,
		TargetPath:  target,
		PartialPath: partial,
		BytesTotal:  manifest.Size,
		ETag:        manifest.ETag,
		State:       string(DownloadStateRunning),
	}
	if old, ok, err := s.downloads.ByArtifact(ctx, artifact.RepoID, artifact.Revision, artifact.Filename); err == nil && ok {
		rec.ID = old.ID
		rec.CreatedAt = old.CreatedAt
	}

	var lastErr error
	for attempt := 1; attempt <= max(1, s.cfg.Download.Retries); attempt++ {
		if err := s.downloadOnce(ctx, manifest, partial, target, progressOut, &rec); err == nil {
			sum, err := VerifyFile(target, artifact)
			if err != nil {
				return model.Model{}, err
			}
			rec.State = string(DownloadStateReady)
			rec.BytesDone = rec.BytesTotal
			rec.Error = ""
			_ = s.downloads.Upsert(ctx, rec)
			return s.register(ctx, artifact, target, sum, manifest.Size)
		} else {
			lastErr = err
			rec.State = string(DownloadStateFailed)
			rec.Error = err.Error()
			_ = s.downloads.Upsert(ctx, rec)
			select {
			case <-ctx.Done():
				return model.Model{}, utils.WrapError(utils.CodeDownloadInterrupted, "download interrupted", ctx.Err())
			case <-time.After(RetryDelay(attempt)):
			}
		}
	}
	return model.Model{}, utils.WrapError(utils.CodeDownloadInterrupted, "download failed", lastErr)
}

func (s *Service) downloadOnce(ctx context.Context, manifest Manifest, partial, target string, progressOut io.Writer, rec *database.DownloadRecord) error {
	offset := ExistingPartialSize(partial)
	req, err := s.hf.NewDownloadRequest(ctx, manifest, offset)
	if err != nil {
		return err
	}
	resp, err := s.hf.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if offset > 0 && resp.StatusCode != http.StatusPartialContent {
		offset = 0
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("download returned %s", resp.Status)
	}

	flags := os.O_CREATE | os.O_WRONLY
	if offset > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	if err := os.MkdirAll(filepath.Dir(partial), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(partial, flags, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	if progressOut == nil {
		progressOut = io.Discard
	}
	total := manifest.Size
	if total <= 0 && resp.ContentLength > 0 {
		total = offset + resp.ContentLength
	}
	progress := mpb.New(mpb.WithOutput(progressOut))
	bar := progress.AddBar(total,
		mpb.PrependDecorators(decor.Name(manifest.Artifact.Filename+" "), decor.CountersKibiByte("% .2f / % .2f")),
		mpb.AppendDecorators(decor.Percentage()),
	)
	if offset > 0 {
		bar.SetCurrent(offset)
	}

	var written int64
	buf := make([]byte, 64*1024)
	for {
		nr, readErr := resp.Body.Read(buf)
		if nr > 0 {
			nw, writeErr := f.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
				bar.IncrBy(nw)
			}
			if writeErr != nil {
				bar.SetTotal(total, true)
				progress.Wait()
				return writeErr
			}
			if nw != nr {
				bar.SetTotal(total, true)
				progress.Wait()
				return io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			bar.SetTotal(total, true)
			progress.Wait()
			return readErr
		}
	}
	rec.BytesDone = offset + written
	rec.BytesTotal = total
	rec.State = string(DownloadStateRunning)
	_ = s.downloads.Upsert(ctx, *rec)
	bar.SetTotal(total, true)
	progress.Wait()
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(partial, target)
}

func (s *Service) register(ctx context.Context, artifact model.Artifact, path, sha string, size int64) (model.Model, error) {
	now := time.Now().UTC()
	id := utils.SanitizeModelPathSegment(artifact.CanonicalID + "-" + artifact.Revision)
	m := model.Model{
		ID:            id,
		UserID:        artifact.UserID,
		CanonicalID:   artifact.CanonicalID,
		Backend:       artifact.Backend,
		RepoID:        artifact.RepoID,
		Revision:      artifact.Revision,
		Quantization:  artifact.Quantization,
		Family:        artifact.Family,
		Parameters:    artifact.Parameters,
		ContextLength: artifact.ContextLength,
		Path:          path,
		State:         model.StateReady,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	file := model.File{
		ID:        utils.NewID("file"),
		ModelID:   m.ID,
		Path:      path,
		Filename:  artifact.Filename,
		SizeBytes: size,
		SHA256:    sha,
		State:     model.StateReady,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.models.Save(ctx, m, []model.File{file}); err != nil {
		return model.Model{}, err
	}
	return m, nil
}
