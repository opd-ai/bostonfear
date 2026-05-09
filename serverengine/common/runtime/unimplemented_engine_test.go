package runtime

import (
	"reflect"
	"testing"
)

func TestUnimplementedEngineAllowedOrigins(t *testing.T) {
	engine := NewUnimplementedEngine("test").(*UnimplementedEngine)

	if got := engine.AllowedOrigins(); got != nil {
		t.Fatalf("expected nil origins by default, got %v", got)
	}

	engine.SetAllowedOrigins([]string{" Example.com ", "", "LOCALHOST:8080"})
	got := engine.AllowedOrigins()
	want := []string{"example.com", "localhost:8080"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("AllowedOrigins() = %v, want %v", got, want)
	}

	got[0] = "mutated"
	if again := engine.AllowedOrigins(); !reflect.DeepEqual(again, want) {
		t.Fatalf("AllowedOrigins() should return a copy, got %v", again)
	}
}
