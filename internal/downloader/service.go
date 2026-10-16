// Package downloader handles Hugging Face Hub transfers and local verification.
package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/bitcli/bitcli/internal/cache"
	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/database"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/utils"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// stallTimeout is how long a transfer may make zero progress before the
// watchdog cancels it. It is deliberately generous: a stall means no bytes at
// all are arriving, which even on a very slow link should never happen for
// this long while a server keeps sending.
const stallTimeout = 120 * time.Second

// ProgressFunc reports incremental download progress. It is optional.
type ProgressFunc func(bytesDone, bytesTotal int64)

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
	return s.PullModelWithProgress(ctx, artifact, progressOut, nil)
}

// PullModelWithProgress is PullModel with an optional progress callback that
// receives cumulative byte counts as the transfer advances.
func (s *Service) PullModelWithProgress(ctx context.Context, artifact model.Artifact, progressOut io.Writer, onProgress ProgressFunc) (model.Model, error) {
	if err := s.cache.Ensure(); err != nil {
		return model.Model{}, err
	}
	target := s.cache.Layout().ModelFile(artifact)
	if _, err := os.Stat(target); err == nil {
		size := artifact.SizeBytes
		if fi, statErr := os.Stat(target); statErr == nil {
			size = fi.Size()
		}
		return s.register(ctx, artifact, target, "", size)
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

	retries := max(1, s.cfg.Download.Retries)
	var lastErr error
	for attempt := 1; attempt <= retries; attempt++ {
		if err := s.downloadOnce(ctx, manifest, partial, target, progressOut, &rec, onProgress); err == nil {
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
			// Do not sleep after the final attempt: surface the
			// error to the caller immediately.
			if attempt == retries {
				break
			}
			select {
			case <-ctx.Done():
				return model.Model{}, utils.WrapError(utils.CodeDownloadInterrupted, "download interrupted", ctx.Err())
			case <-time.After(RetryDelay(attempt)):
			}
		}
	}
	return model.Model{}, utils.WrapError(utils.CodeDownloadInterrupted, "download failed", lastErr)
}

func (s *Service) downloadOnce(ctx context.Context, manifest Manifest, partial, target string, progressOut io.Writer, rec *database.DownloadRecord, onProgress ProgressFunc) error {
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

	// A 416 means our resume offset is past the end of the remote file
	// (e.g. a corrupt or stale .partial). Restart the transfer from zero.
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
		_ = resp.Body.Close()
		if rmErr := os.Remove(partial); rmErr != nil && !os.IsNotExist(rmErr) {
			return rmErr
		}
		req, err = s.hf.NewDownloadRequest(ctx, manifest, 0)
		if err != nil {
			return err
		}
		resp, err = s.hf.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		offset = 0
	}
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

	// No-progress watchdog: cancels the request context when the transfer
	// stalls so a silent network hang cannot block forever.
	stallCtx, cancelStall := context.WithCancel(ctx)
	defer cancelStall()
	req = req.WithContext(stallCtx)
	var lastProgressUnix atomic.Int64
	lastProgressUnix.Store(time.Now().Unix())
	watchdogDone := make(chan struct{})
	defer close(watchdogDone)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-watchdogDone:
				return
			case <-ticker.C:
				if time.Since(time.Unix(lastProgressUnix.Load(), 0)) > stallTimeout {
					cancelStall()
					return
				}
			}
		}
	}()
	touch := func() { lastProgressUnix.Store(time.Now().Unix()) }

	var written int64
	buf := make([]byte, 64*1024)
	for {
		nr, readErr := resp.Body.Read(buf)
		if nr > 0 {
			touch()
			nw, writeErr := f.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
				bar.IncrBy(nw)
				if onProgress != nil {
					onProgress(offset+written, total)
				}
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
		ChatMode:      artifact.ChatMode,
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
