package finalhour

import (
	"testing"
)

func TestModuleNewEngineReturnsFinalHourEngine(t *testing.T) {
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

	// Verify engine can be started
	startErr := engine.Start()
	if startErr != nil {
		t.Fatalf("Engine.Start() failed: %v", startErr)
	}
}

func TestModuleKey(t *testing.T) {
	mod := NewModule()
	if mod.Key() != "finalhour" {
		t.Errorf("expected key 'finalhour', got '%s'", mod.Key())
	}
}

func TestModuleDescription(t *testing.T) {
	mod := NewModule()
	desc := mod.Description()
	if desc == "" {
		t.Error("expected non-empty description")
	}
}
