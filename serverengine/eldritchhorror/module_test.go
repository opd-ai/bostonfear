package eldritchhorror

import (
	"testing"
)

func TestModuleNewEngineReturnsEldritchHorrorEngine(t *testing.T) {
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

	// Verify module metadata
	if module.Key() != "eldritchhorror" {
		t.Fatalf("expected key 'eldritchhorror', got %s", module.Key())
	}
	if module.Description() == "" {
		t.Fatal("expected non-empty description")
	}

	// Verify engine is properly typed as Engine wrapper
	if _, ok := engine.(*Engine); !ok {
		t.Fatalf("expected *Engine type, got %T", engine)
	}
}
