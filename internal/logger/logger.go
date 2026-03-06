package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// Logger defines the minimal logging contract
type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// SlogLogger is adapter over slog.Logger
type SlogLogger struct {
	logger *slog.Logger
}

func (l *SlogLogger) Debug(msg string, args ...any) {
	l.logger.Debug(msg, args...)
}

func (l *SlogLogger) Info(msg string, args ...any) {
	l.logger.Info(msg, args...)
}

func (l *SlogLogger) Warn(msg string, args ...any) {
	l.logger.Warn(msg, args...)
}

func (l *SlogLogger) Error(msg string, args ...any) {
	l.logger.Error(msg, args...)
}

type Options struct {
	Level  string
	Format string
}

// New creates app logger based on provided options
func New(opts Options) (Logger, error) {
	level, err := parseLevel(opts.Level)
	if err != nil {
		return nil, err
	}

	handler, err := newHandler(opts.Format, level)
	if err != nil {
		return nil, err
	}
	return &SlogLogger{
		logger: slog.New(handler),
	}, nil
}

func parseLevel(level string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid level: %s", level)
	}
}

func newHandler(format string, level slog.Level) (slog.Handler, error) {
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "text":
		return slog.NewTextHandler(os.Stdout, opts), nil
	case "json":
		return slog.NewJSONHandler(os.Stdout, opts), nil
	default:
		return nil, fmt.Errorf("invalid format: %s", format)
	}
}
