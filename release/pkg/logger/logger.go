package logger

import (
	"context"
	"log/slog"
	"os"
)

type Logger struct {
	logger *slog.Logger
}

func NewLogger(level slog.Level) *Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return &Logger{logger: slog.New(handler)}
}

func ParseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l *Logger) Debug(taskID, msg string) {
	l.log(context.Background(), slog.LevelDebug, taskID, msg)
}

func (l *Logger) Info(taskID, msg string) {
	l.log(context.Background(), slog.LevelInfo, taskID, msg)
}

func (l *Logger) Warn(taskID, msg string) {
	l.log(context.Background(), slog.LevelWarn, taskID, msg)
}

func (l *Logger) Error(taskID, msg string) {
	l.log(context.Background(), slog.LevelError, taskID, msg)
}

func (l *Logger) log(ctx context.Context, level slog.Level, taskID, msg string) {
	if taskID == "" {
		l.logger.Log(ctx, level, msg)
	} else {
		l.logger.Log(ctx, level, "task_id", taskID, msg)
	}
}
