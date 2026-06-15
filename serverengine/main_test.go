package serverengine

import (
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/opd-ai/bostonfear/serverengine/common/logging"
)

func TestMain(m *testing.M) {
	orig := logging.Logger
	logging.Logger = slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelError}))

	code := m.Run()

	logging.Logger = orig
	os.Exit(code)
}
