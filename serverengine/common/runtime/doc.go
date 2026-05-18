// Package runtime provides the module registry and UnimplementedEngine placeholder.
//
// UnimplementedEngine satisfies the Engine interface but returns "not implemented"
// errors for Start() and HandleConnection(). Other methods (SetAllowedOrigins,
// health checks, metrics collectors) succeed silently to allow CLI initialization
// and monitoring endpoint registration, but do not enable gameplay.
//
// This design allows server startup to proceed with scaffolded game modules
// (eldersign, eldritchhorror, finalhour) without crashing, while clearly signaling
// via Start() failure that the selected module is not yet playable.
package runtime
