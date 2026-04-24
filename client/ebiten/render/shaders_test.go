// Package render — shader compilation tests.
// Verifies that all three embedded Kage shaders (fog, glow, doom) compile
// without errors using NewShaderSet (ROADMAP §Priority 6: Ebitengine test coverage).
//
// These tests require an Ebitengine GPU context, which in turn requires a
// display connection (GLFW init). Build and run with:
//
//	DISPLAY=:99 xvfb-run -a go test -race -tags=requires_display ./client/ebiten/render/...

//go:build requires_display

package render

import "testing"

// TestShaderSet_Compiles verifies that NewShaderSet successfully compiles all
// three embedded Kage shaders and returns a fully populated *ShaderSet.
// This is a regression guard against shader source changes that break compilation.
func TestShaderSet_Compiles(t *testing.T) {
	ss, err := NewShaderSet()
	if err != nil {
		t.Fatalf("NewShaderSet() error: %v", err)
	}
	defer ss.Deallocate()

	if ss.Fog == nil {
		t.Error("ShaderSet.Fog is nil after NewShaderSet()")
	}
	if ss.Glow == nil {
		t.Error("ShaderSet.Glow is nil after NewShaderSet()")
	}
	if ss.Doom == nil {
		t.Error("ShaderSet.Doom is nil after NewShaderSet()")
	}
}

// TestShaderSet_Deallocate_NilSafe verifies that Deallocate does not panic
// when called on a ShaderSet with nil shader fields.
func TestShaderSet_Deallocate_NilSafe(t *testing.T) {
	ss := &ShaderSet{} // all fields nil
	ss.Deallocate()   // must not panic
}
