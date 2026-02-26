package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New returns a zap logger (JSON encoder to stderr, info level).
func New() *zap.Logger {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{"stderr"}
	cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	log, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	return log
}

// NewNop returns a no-op logger.
func NewNop() *zap.Logger {
	return zap.NewNop()
}
