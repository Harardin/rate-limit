package log

import (
	"io"
)

type writerLoggerAdapter struct {
	logger Logger
}

func NewWriterLoggerAdapter(logger Logger) io.Writer {
	return &writerLoggerAdapter{
		logger: logger,
	}
}

func (w *writerLoggerAdapter) Write(p []byte) (n int, err error) {
	w.logger.Debug(string(p))
	return len(p), nil
}
