package eldersign

import (
	"testing"
)

func TestModuleNewEngineReturnsElderSignEngine(t *testing.T) {
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

	// Verify engine is properly configured with Elder Sign-specific components
	esEngine, ok := engine.(*Engine)
	if !ok {
		t.Fatal("expected Engine type from NewEngine")
	}
	if esEngine.GameServer == nil {
		t.Fatal("expected GameServer to be initialized")
	}

	// Verify the engine can be configured with allowed origins
	engine.SetAllowedOrigins([]string{"localhost:3000"})

	// Note: We don't call Start() here as it would start a full game server
	// which requires port binding and connection handling. Integration tests
	// will verify end-to-end functionality.
}
