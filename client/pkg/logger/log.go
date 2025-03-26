package logger

import (
	"log/slog"
	"os"
)

var GlobalLogger *Logger

type LoggerInterface interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	With(args ...any) LoggerInterface
}

type Logger struct {
	LoggerInterface
}

type LoggerOptions struct {
	Level     string
	LogFormat string
}

func SetupGlobalLogger(opts LoggerOptions) *Logger {
	return NewLogger(&SlogWrapper{
		logger: NewSlogLogger(SlogOptions{
			Handler: slogHandler(opts.LogFormat, slogLevel(opts.Level)),
		}),
	})
}

func NewLogger(l LoggerInterface) *Logger {
	return &Logger{LoggerInterface: l}
}

type SlogOptions struct {
	Handler slog.Handler
}

func NewSlogLogger(opts SlogOptions) *slog.Logger {
	return slog.New(opts.Handler)
}

type SlogWrapper struct {
	logger *slog.Logger
}

func (s *SlogWrapper) Debug(msg string, args ...any) {
	s.logger.Debug(msg, args...)
}

func (s *SlogWrapper) Error(msg string, args ...any) {
	s.logger.Error(msg, args...)
}

func (s *SlogWrapper) Info(msg string, args ...any) {
	s.logger.Info(msg, args...)
}

func (s *SlogWrapper) Warn(msg string, args ...any) {
	s.logger.Warn(msg, args...)
}

func (s *SlogWrapper) With(args ...any) LoggerInterface {
	return &SlogWrapper{
		logger: s.logger.With(args...),
	}
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{LoggerInterface: l.LoggerInterface.With(args...)}
}

func slogLevel(l string) slog.Level {
	switch l {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func slogHandler(format string, level slog.Level) slog.Handler {
	options := &slog.HandlerOptions{
		Level: level,
	}
	switch format {
	case "json":
		return slog.NewJSONHandler(os.Stdout, options)
	default:
		return slog.NewTextHandler(os.Stdout, options)
	}
}
