// Command bitcli wires Cobra commands for the BitCLI CLI.
package main

import (
	"context"
	"io"
	"os"

	"github.com/bitcli/bitcli/internal/cache"
	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/database"
	"github.com/bitcli/bitcli/internal/downloader"
	"github.com/bitcli/bitcli/internal/hardware"
	"github.com/bitcli/bitcli/internal/logger"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/process"
	bitruntime "github.com/bitcli/bitcli/internal/runtime"
	"github.com/bitcli/bitcli/internal/runtime/backend"
	"github.com/bitcli/bitcli/internal/runtime/backend/bitnet"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type app struct {
	cfg       config.Config
	paths     config.Paths
	log       *zap.Logger
	db        *database.DB
	models    *model.Service
	cache     *cache.Service
	downloads *downloader.Service
	runtime   *bitruntime.Service
	hardware  *hardware.Service
	chats     *database.ChatRepository
	backends  *backend.Registry
	runner    *process.Runner
}

func newApp(cmd *cobra.Command, opts *rootOptions) (*app, func(), error) {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	cfgSvc, err := config.NewService()
	if err != nil {
		return nil, nil, err
	}
	cfg, paths, err := cfgSvc.Load(opts.configPath)
	if err != nil {
		return nil, nil, err
	}
	if opts.verbose {
		cfg.Logging.Level = "debug"
	}
	log, err := logger.New(cfg, paths)
	if err != nil {
		return nil, nil, err
	}
	db, err := database.Open(ctx, paths.DatabaseFile)
	if err != nil {
		_ = log.Sync()
		return nil, nil, err
	}

	modelRepo := database.NewModelRepository(db)
	modelSvc := model.NewService(modelRepo, model.NewResolver(model.DefaultCatalog()))
	cacheSvc := cache.NewService(paths)
	downloadRepo := database.NewDownloadRepository(db)
	downloadSvc := downloader.NewService(cfg, cacheSvc, modelSvc, downloadRepo)
	runner := process.NewRunner()
	registry := backend.NewRegistry()
	registry.Register(bitnet.New(cfg, paths, runner, log))

	a := &app{
		cfg:       cfg,
		paths:     paths,
		log:       log,
		db:        db,
		models:    modelSvc,
		cache:     cacheSvc,
		downloads: downloadSvc,
		runtime:   bitruntime.NewService(modelSvc, registry),
		hardware:  hardware.NewService(),
		chats:     database.NewChatRepository(db),
		backends:  registry,
		runner:    runner,
	}
	cleanup := func() {
		_ = db.Close()
		_ = log.Sync()
	}
	return a, cleanup, nil
}

func (a *app) ensureModel(ctx context.Context, id string, progress io.Writer) (model.Model, error) {
	if id == "" {
		id = a.cfg.DefaultModel
	}
	if m, err := a.models.Local(ctx, id); err == nil {
		return m, nil
	}
	artifact, err := a.models.Resolve(id)
	if err != nil {
		return model.Model{}, err
	}
	if progress == nil {
		progress = os.Stdout
	}
	return a.downloads.PullModel(ctx, artifact, progress)
}

func runtimeOptions(cfg config.Config) bitruntime.Options {
	return bitruntime.Options{
		Temperature:   cfg.Runtime.Temperature,
		TopP:          cfg.Runtime.TopP,
		TopK:          cfg.Runtime.TopK,
		Threads:       cfg.Runtime.Threads,
		GPULayers:     cfg.Runtime.GPULayers,
		ContextLength: cfg.Runtime.ContextLength,
		MaxTokens:     cfg.Runtime.MaxTokens,
	}
}

