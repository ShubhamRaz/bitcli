// Package logger configures structured logging for BitCLI.
package logger

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

// ParseLevel converts a config string into a Zap level.
func ParseLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

