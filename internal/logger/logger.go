// Package logger configures structured logging for BitCLI.
package logger

import (
	"os"
	"path/filepath"

	"github.com/bitcli/bitcli/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a Zap logger with optional file output.
func New(cfg config.Config, paths config.Paths) (*zap.Logger, error) {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	level := zap.NewAtomicLevelAt(ParseLevel(cfg.Logging.Level))
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderCfg),
		zapcore.Lock(os.Stderr),
		level,
	)

	cores := []zapcore.Core{consoleCore}
	if cfg.Logging.File {
		if err := os.MkdirAll(paths.LogDir, 0o755); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(filepath.Join(paths.LogDir, "bitcli.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		cores = append(cores, zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderCfg),
			zapcore.AddSync(file),
			level,
		))
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller()), nil
}

