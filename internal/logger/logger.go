package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
)

var _logger *slog.Logger

func init() {
	_logger = NewSlogger()
}

func handler() *slog.TextHandler {
	var opts = &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	return slog.NewTextHandler(os.Stdout, opts)
}

func handle(level slog.Level, msg string, args ...any) {
	_, f, l, _ := runtime.Caller(2)
	source := fmt.Sprintf("%s:%d", f, l)
	args = append(args, slog.String("source", source))

	_logger.Log(context.Background(), level, msg, args...)
}

func Debug(msg string, args ...any) {
	handle(slog.LevelDebug, msg, args...)
}

func Info(msg string, args ...any) {
	handle(slog.LevelInfo, msg, args...)
}

func Warn(msg string, args ...any) {
	handle(slog.LevelWarn, msg, args...)
}

func Error(msg string, args ...any) {
	handle(slog.LevelError, msg, args...)
}

func NewSlogger() *slog.Logger {
	return slog.New(handler())
}
