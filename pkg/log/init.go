package log

import (
	"log"

	"go.uber.org/zap"
)

// InitLogger Init logger
func InitLogger() *zap.Logger {
	var err error
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to init logger: %s", err.Error())
	}

	return logger
}

func SyncLogger(logger *zap.Logger) {
	err := logger.Sync()
	if err != nil {
		log.Printf("Failed to syncing log: %s", err.Error())
	}
}
