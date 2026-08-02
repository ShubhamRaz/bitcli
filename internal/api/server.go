// Package api exposes BitCLI REST API with OpenAI-compatible endpoints.
package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bitcli/bitcli/internal/cache"
	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/downloader"
	"github.com/bitcli/bitcli/internal/hardware"
	"github.com/bitcli/bitcli/internal/model"
	bitruntime "github.com/bitcli/bitcli/internal/runtime"
	"github.com/bitcli/bitcli/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server owns the Gin engine and application dependencies.
type Server struct {
	cfg        config.Config
	models     *model.Service
	downloads  *downloader.Service
	runtime    *bitruntime.Service
	cache      *cache.Service
	hardware   *hardware.Service
	version    string
	log        *zap.Logger
	engine     *gin.Engine
}

// Dependencies contains the services required by the HTTP server.
type Dependencies struct {
	Config    config.Config
	Models    *model.Service
	Downloads *downloader.Service
	Runtime   *bitruntime.Service
	Cache     *cache.Service
	Hardware  *hardware.Service
	Version   string
	Logger    *zap.Logger
}

// NewServer creates and wires a Gin HTTP server.
func NewServer(deps Dependencies) *Server {
	if deps.Logger == nil {
		deps.Logger = zap.NewNop()
	}
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		cfg:       deps.Config,
		models:    deps.Models,
		downloads: deps.Downloads,
		runtime:   deps.Runtime,
		cache:     deps.Cache,
		hardware:  deps.Hardware,
		version:   deps.Version,
		log:       deps.Logger,
	}
	s.engine = gin.New()
	s.engine.Use(gin.Recovery())
	s.engine.Use(s.cors())
	s.registerRoutes()
	return s
}

// Run starts the HTTP server on the configured host and port.
func (s *Server) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", s.cfg.API.Host, s.cfg.API.Port),
		Handler: s.engine,
	}
	errs := make(chan error, 1)
	go func() {
		s.log.Info("starting API server", zap.String("addr", server.Addr))
		errs <- server.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), httpShutdownTimeout)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errs:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// Handler returns the underlying http.Handler for use in tests.
func (s *Server) Handler() http.Handler {
	return s.engine
}

func (s *Server) cors() gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, origin := range s.cfg.API.AllowOrigins {
		allowed[origin] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" {
			if _, ok := allowed[origin]; ok {
				c.Header("Access-Control-Allow-Origin", origin)
				c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
				c.Header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			}
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func (s *Server) ensureModel(ctx context.Context, id string) (model.Model, error) {
	if strings.TrimSpace(id) == "" {
		id = s.cfg.DefaultModel
	}
	if m, err := s.models.Local(ctx, id); err == nil {
		return m, nil
	}
	artifact, err := s.models.Resolve(id)
	if err != nil {
		return model.Model{}, err
	}
	return s.downloads.PullModel(ctx, artifact, io.Discard)
}

func writeError(c *gin.Context, status int, err error) {
	c.JSON(status, ErrorResponse{
		Error: APIError{
			Code:    string(utils.ErrorCode(err)),
			Message: err.Error(),
		},
	})
}

