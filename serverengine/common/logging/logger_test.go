package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

func TestSetLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		wantFunc func(*slog.Logger) bool
	}{
		{
			name:  "Debug level enables debug logs",
			level: LevelDebug,
			wantFunc: func(l *slog.Logger) bool {
				return l.Enabled(nil, slog.LevelDebug)
			},
		},
		{
			name:  "Info level disables debug logs",
			level: LevelInfo,
			wantFunc: func(l *slog.Logger) bool {
				return !l.Enabled(nil, slog.LevelDebug) && l.Enabled(nil, slog.LevelInfo)
			},
		},
		{
			name:  "Warn level disables info logs",
			level: LevelWarn,
			wantFunc: func(l *slog.Logger) bool {
				return !l.Enabled(nil, slog.LevelInfo) && l.Enabled(nil, slog.LevelWarn)
			},
		},
		{
			name:  "Error level disables warn logs",
			level: LevelError,
			wantFunc: func(l *slog.Logger) bool {
				return !l.Enabled(nil, slog.LevelWarn) && l.Enabled(nil, slog.LevelError)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetLevel(tt.level)
			if !tt.wantFunc(Logger) {
				t.Errorf("SetLevel(%v) did not configure logger as expected", tt.level)
			}
		})
	}
}

func TestStructuredLogging(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	Logger = slog.New(slog.NewJSONHandler(&buf, opts))

	tests := []struct {
		name    string
		logFunc func()
		wantMsg string
		wantKey string
		wantVal interface{}
		wantLvl string
	}{
		{
			name: "Debug with structured fields",
			logFunc: func() {
				Debug("test debug message", "playerID", "player1", "action", "move")
			},
			wantMsg: "test debug message",
			wantKey: "playerID",
			wantVal: "player1",
			wantLvl: "DEBUG",
		},
		{
			name: "Info with structured fields",
			logFunc: func() {
				Info("test info message", "doom", 5)
			},
			wantMsg: "test info message",
			wantKey: "doom",
			wantVal: float64(5), // JSON numbers are float64
			wantLvl: "INFO",
		},
		{
			name: "Warn with structured fields",
			logFunc: func() {
				Warn("test warn message", "error", "timeout")
			},
			wantMsg: "test warn message",
			wantKey: "error",
			wantVal: "timeout",
			wantLvl: "WARN",
		},
		{
			name: "Error with structured fields",
			logFunc: func() {
				Error("test error message", "code", 500)
			},
			wantMsg: "test error message",
			wantKey: "code",
			wantVal: float64(500), // JSON numbers are float64
			wantLvl: "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc()

			output := buf.String()
			if !strings.Contains(output, tt.wantMsg) {
				t.Errorf("Log output missing message: got %q, want to contain %q", output, tt.wantMsg)
			}

			// Parse JSON to verify structured fields
			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse JSON log output: %v", err)
			}

			if val, ok := logEntry[tt.wantKey]; !ok {
				t.Errorf("Log output missing key %q", tt.wantKey)
			} else if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", tt.wantVal) {
				t.Errorf("Log field %q = %v, want %v", tt.wantKey, val, tt.wantVal)
			}

			if level, ok := logEntry["level"]; !ok {
				t.Errorf("Log output missing level field")
			} else if levelStr := strings.ToUpper(level.(string)); levelStr != tt.wantLvl {
				t.Errorf("Log level = %q, want %q", levelStr, tt.wantLvl)
			}
		})
	}
}
