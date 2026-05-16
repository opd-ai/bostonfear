// Package logging provides structured logging with levels for the BostonFear server.
package logging

import (
	"log/slog"
	"os"
)

// Logger is the default structured logger instance
var Logger *slog.Logger

// LogLevel defines the logging level
type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

func init() {
	// Initialize default logger with JSON handler
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo, // Default to INFO level
	}
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// SetLevel configures the logging level
func SetLevel(level LogLevel) {
	var slogLevel slog.Level
	switch level {
	case LevelDebug:
		slogLevel = slog.LevelDebug
	case LevelInfo:
		slogLevel = slog.LevelInfo
	case LevelWarn:
		slogLevel = slog.LevelWarn
	case LevelError:
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: slogLevel,
	}
	Logger = slog.New(slog.NewJSONHandler(os.Stdout, opts))
}

// Debug logs a debug-level message with structured fields
func Debug(msg string, args ...any) {
	Logger.Debug(msg, args...)
}

// Info logs an info-level message with structured fields
func Info(msg string, args ...any) {
	Logger.Info(msg, args...)
}

// Warn logs a warning-level message with structured fields
func Warn(msg string, args ...any) {
	Logger.Warn(msg, args...)
}

// Error logs an error-level message with structured fields
func Error(msg string, args ...any) {
	Logger.Error(msg, args...)
}
