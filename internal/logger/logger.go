package logger

import (
	"io"
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

// Sets up the default logger
func Initialize(level slog.Level, output io.Writer) {
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(output, opts)
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// Logs a debug message
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Logs an info message
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Logs a warning message
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Logs an error message
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Returns a logger with additional attributes
func With(args ...any) *slog.Logger {
	return defaultLogger.With(args...)
}

func init() {
	// Default initialization with info level
	Initialize(slog.LevelInfo, nil)
}
