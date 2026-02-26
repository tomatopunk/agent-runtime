package main

import (
	"os"

	"github.com/tomatopunk/agent-runtime/internal/logger"
	"go.uber.org/zap"
)

func main() {
	log := logger.New()
	_ = zap.ReplaceGlobals(log)
	if err := rootCmd.Execute(); err != nil {
		log.Error("command failed", zap.Error(err))
		os.Exit(1)
	}
}
