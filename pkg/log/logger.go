package log

import (
	"os"

	"github.com/Harardin/rate-limit/pkg/sentry"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogLevel string

func (l LogLevel) String() string {
	return string(l)
}

const (
	DEBUG   LogLevel = "debug"
	INFO    LogLevel = "info"
	WARNING LogLevel = "warning"
	ERROR   LogLevel = "error"
	FATAL   LogLevel = "fatal"
	PANIC   LogLevel = "panic"
)

// GetAllLevels return all log levels. Used in validation.
func GetAllLevels() []interface{} {
	return []interface{}{
		DEBUG.String(), INFO.String(), WARNING.String(), ERROR.String(), FATAL.String(), PANIC.String(),
	}
}

type LogFormat string

const (
	FORMAT_CONSOLE LogFormat = "console"
	FORMAT_JSON    LogFormat = "json"
)

// Logger common interface
type Logger interface {
	Debug(...interface{})
	Debugf(string, ...interface{})
	Info(...interface{})
	Infof(string, ...interface{})
	Warn(...interface{})
	Warnf(string, ...interface{})
	Error(...interface{})
	Errorf(string, ...interface{})
	Fatal(...interface{})
	Fatalf(string, ...interface{})
	Panic(...interface{})
	Panicf(string, ...interface{})
	With(...interface{}) *zap.SugaredLogger
	Sync() error
}

func initLogger(level LogLevel, format LogFormat, consoleColored bool, timeKey string) *zap.Logger {
	atom := zap.NewAtomicLevel()

	encoderCfg := zap.NewProductionEncoderConfig()

	encoderCfg.TimeKey = "ts"
	if timeKey != "" {
		encoderCfg.TimeKey = timeKey
	}

	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	// Default JSON encoder
	encoder := zapcore.NewJSONEncoder(encoderCfg)
	switch format {
	case FORMAT_CONSOLE:
		if consoleColored {
			encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	}

	logger := zap.New(zapcore.NewCore(
		encoder,
		zapcore.Lock(os.Stdout),
		atom),
		zap.AddCaller(),
	)

	switch level {
	case DEBUG:
		atom.SetLevel(zap.DebugLevel)
	case INFO:
		atom.SetLevel(zap.InfoLevel)
	case WARNING:
		atom.SetLevel(zap.WarnLevel)
	case ERROR:
		atom.SetLevel(zap.ErrorLevel)
	case FATAL:
		atom.SetLevel(zap.FatalLevel)
	case PANIC:
		atom.SetLevel(zap.PanicLevel)
	default:
		atom.SetLevel(zap.InfoLevel)
	}

	return logger
}

// New - init new logger with options
func New(opts ...Option) Logger {

	options := Options{}
	for _, opt := range opts {
		opt(&options)
	}

	if options.LogLevel == "" {
		options.LogLevel = DEBUG
	}

	logger := initLogger(
		options.LogLevel,
		options.LogFormat,
		options.ConsoleColored,
		options.TimeKey,
	)

	if options.AppName != "" {
		logger = logger.With(
			zap.String("app", options.AppName),
		)
	}

	if options.AppVersion != "" {
		logger = logger.With(
			zap.String("version", options.AppVersion),
		)
	}

	if options.SentryConfig != nil && options.SentryConfig.Enabled {
		var err error
		logger, err = sentry.AddSentryToZap(logger, *options.SentryConfig, options.AppVersion)
		if err != nil {
			logger.Fatal("logger init sentry error: ", zap.Error(err))
		}
	}

	return logger.Sugar()
}
