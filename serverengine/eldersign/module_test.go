package eldersign

import "testing"

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

	if _, ok := engine.(*Engine); !ok {
		t.Fatalf("expected *Engine, got %T", engine)
	}
}
