package sentry

// Example how to enable sentry
// logger := log.New(log.WithSentry(cfg.Sentry))

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	DSN         string `json:"SENTRY_DSN"`
	Environment string `json:"SENTRY_ENVIRONMENT" default:"development"`
	Enabled     bool   `json:"SENTRY_ENABLED" default:"true"`
}

// AddSentryToZap init sentry for logger
func AddSentryToZap(logger *zap.Logger, cfg Config, release string) (sentryLog *zap.Logger, err error) {
	err = sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.DSN,
		Release:     release,
		Environment: cfg.Environment,
	})
	if err != nil {
		return nil, err
	}
	logger = logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		if entry.Level >= zapcore.ErrorLevel {
			defer sentry.Flush(2 * time.Second)
			sentry.CaptureMessage(fmt.Sprintf("%s, Line No: %d :: %s\n\nstack:\n%s", entry.Caller.File, entry.Caller.Line, entry.Message, entry.Stack))
		}

		return nil
	}))

	return logger, nil
}
