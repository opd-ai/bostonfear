package runtime

import (
	"reflect"
	"testing"

	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

type mockModule struct {
	key  string
	desc string
}

func (m *mockModule) Key() string                          { return m.key }
func (m *mockModule) Description() string                  { return m.desc }
func (m *mockModule) NewEngine() (contracts.Engine, error) { return nil, nil }

func TestRegistryRegisterAndGet(t *testing.T) {
	reg := NewRegistry()
	mod := &mockModule{key: "testgame", desc: "Test Game"}

	if err := reg.Register(mod); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	retrieved, ok := reg.Get("testgame")
	if !ok {
		t.Fatal("Get() returned false for registered module")
	}
	if retrieved != mod {
		t.Errorf("Get() returned different module than registered")
	}
}

func TestRegistryRegisterNormalizesKeys(t *testing.T) {
	reg := NewRegistry()
	mod := &mockModule{key: " TestGame ", desc: "Test"}

	if err := reg.Register(mod); err != nil {
		t.Fatalf("Register() failed: %v", err)
	}

	// Should retrieve with normalized key
	if _, ok := reg.Get("testgame"); !ok {
		t.Error("Get() failed to retrieve module with normalized key")
	}
	if _, ok := reg.Get(" TESTGAME "); !ok {
		t.Error("Get() failed to normalize lookup key")
	}
}

func TestRegistryRegisterNilModule(t *testing.T) {
	reg := NewRegistry()
	err := reg.Register(nil)
	if err == nil {
		t.Fatal("Register(nil) should return error")
	}
	if err.Error() != "module cannot be nil" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegistryRegisterEmptyKey(t *testing.T) {
	reg := NewRegistry()
	mod := &mockModule{key: "", desc: "Test"}
	err := reg.Register(mod)
	if err == nil {
		t.Fatal("Register() with empty key should return error")
	}
}

func TestRegistryRegisterDuplicate(t *testing.T) {
	reg := NewRegistry()
	mod1 := &mockModule{key: "game1", desc: "First"}
	mod2 := &mockModule{key: "game1", desc: "Second"}

	if err := reg.Register(mod1); err != nil {
		t.Fatalf("first Register() failed: %v", err)
	}

	err := reg.Register(mod2)
	if err == nil {
		t.Fatal("Register() should fail for duplicate key")
	}
	if err.Error() != `module "game1" already registered` {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRegistryMustRegister(t *testing.T) {
	reg := NewRegistry()
	mod := &mockModule{key: "test", desc: "Test"}

	// Should not panic for valid module
	reg.MustRegister(mod)

	// Verify it was registered
	if _, ok := reg.Get("test"); !ok {
		t.Error("MustRegister() did not register module")
	}
}

func TestRegistryMustRegisterPanicsOnError(t *testing.T) {
	reg := NewRegistry()
	mod := &mockModule{key: "test", desc: "Test"}
	reg.MustRegister(mod)

	defer func() {
		if r := recover(); r == nil {
			t.Error("MustRegister() should panic on duplicate key")
		}
	}()

	// Should panic on duplicate
	reg.MustRegister(mod)
}

func TestRegistryGetNonExistent(t *testing.T) {
	reg := NewRegistry()
	_, ok := reg.Get("nonexistent")
	if ok {
		t.Error("Get() returned true for non-existent module")
	}
}

func TestRegistryKeys(t *testing.T) {
	reg := NewRegistry()
	mod1 := &mockModule{key: "zebra", desc: "Last"}
	mod2 := &mockModule{key: "alpha", desc: "First"}
	mod3 := &mockModule{key: "middle", desc: "Middle"}

	reg.MustRegister(mod1)
	reg.MustRegister(mod2)
	reg.MustRegister(mod3)

	keys := reg.Keys()
	expected := []string{"alpha", "middle", "zebra"}
	if !reflect.DeepEqual(keys, expected) {
		t.Errorf("Keys() = %v, want %v (sorted)", keys, expected)
	}
}

func TestRegistryKeysEmpty(t *testing.T) {
	reg := NewRegistry()
	keys := reg.Keys()
	if len(keys) != 0 {
		t.Errorf("Keys() for empty registry = %v, want []", keys)
	}
}
