package finalhour

import (
	"strings"
	"testing"
)

func TestModuleNewEngineReturnsFinalHourPlaceholder(t *testing.T) {
	module := NewModule()
	if module == nil {
		t.Fatal("expected module instance")
	}

	engine, err := module.NewEngine()
	if err != nil {
		t.Fatalf("NewEngine returned error: %v", err)
	}
	if engine == nil {
		t.Fatal("expected non-nil engine")
	}

	startErr := engine.Start()
	if startErr == nil {
		t.Fatal("expected Start to return not-implemented error")
	}
	if !strings.Contains(startErr.Error(), "not implemented") {
		t.Fatalf("expected not-implemented error, got %v", startErr)
	}
}
