package logger

import (
	"io"
	"log/slog"
	"os"
)

// Type that defines log levels
type LogLevel string

// Predefined log levels
const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
)

var defaultLogger *slog.Logger

// Convert string to log level
func ParseLogLevel(level LogLevel) slog.Level {
	switch level {
	case DebugLevel:
		return slog.LevelDebug
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	case InfoLevel:
		fallthrough
	default:
		return slog.LevelInfo
	}
}

// Sets up the default logger
func Initialize(level LogLevel, output io.Writer) {
	if output == nil {
		output = os.Stdout
	}

	logLevel := ParseLogLevel(level)
	opts := &slog.HandlerOptions{Level: logLevel}
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
	Initialize(InfoLevel, nil)
}
