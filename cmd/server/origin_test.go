// Package main — tests for WebSocket origin validation (ROADMAP Priority 5).
// Verifies that checkOrigin enforces the allowedOrigins list and falls back to
// permissive mode when the list is empty.
package main

import (
	"net/http"
	"testing"
)

// makeRequest builds a minimal *http.Request with the given Origin header.
func makeRequest(origin string) *http.Request {
	r, _ := http.NewRequest(http.MethodGet, "/ws", nil)
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	return r
}

// TestCheckOrigin_Permissive verifies that an empty allowedOrigins list accepts
// any origin, preserving the default local-development behaviour.
func TestCheckOrigin_Permissive(t *testing.T) {
	gs := NewGameServer()
	// No allowed origins configured — every origin must be accepted.
	for _, origin := range []string{
		"http://localhost:8080",
		"http://evil.example.com",
		"https://anything.invalid",
		"",
	} {
		r := makeRequest(origin)
		if !gs.checkOrigin(r) {
			t.Errorf("checkOrigin(%q) = false; want true (permissive mode)", origin)
		}
	}
}

// TestCheckOrigin_NoOriginHeader verifies that requests without an Origin header
// are always accepted, even when allowedOrigins is non-empty.
func TestCheckOrigin_NoOriginHeader(t *testing.T) {
	gs := NewGameServer()
	gs.SetAllowedOrigins([]string{"localhost:8080"})
	r := makeRequest("") // no Origin header
	if !gs.checkOrigin(r) {
		t.Error("checkOrigin with no Origin header = false; want true")
	}
}

// TestCheckOrigin_AllowedOrigin verifies that a matching origin is accepted.
func TestCheckOrigin_AllowedOrigin(t *testing.T) {
	gs := NewGameServer()
	gs.SetAllowedOrigins([]string{"localhost:8080", "mygame.example.com"})

	allowed := []string{
		"http://localhost:8080",
		"https://localhost:8080",
		"http://mygame.example.com",
		"https://mygame.example.com",
	}
	for _, origin := range allowed {
		r := makeRequest(origin)
		if !gs.checkOrigin(r) {
			t.Errorf("checkOrigin(%q) = false; want true", origin)
		}
	}
}

// TestCheckOrigin_RejectedOrigin verifies that an origin not in allowedOrigins
// is rejected when the list is non-empty.
func TestCheckOrigin_RejectedOrigin(t *testing.T) {
	gs := NewGameServer()
	gs.SetAllowedOrigins([]string{"localhost:8080"})

	rejected := []string{
		"http://evil.example.com",
		"http://localhost:9090",
		"https://attacker.invalid",
	}
	for _, origin := range rejected {
		r := makeRequest(origin)
		if gs.checkOrigin(r) {
			t.Errorf("checkOrigin(%q) = true; want false (origin not allowed)", origin)
		}
	}
}

// TestCheckOrigin_CaseInsensitive verifies that host comparison ignores case.
func TestCheckOrigin_CaseInsensitive(t *testing.T) {
	gs := NewGameServer()
	gs.SetAllowedOrigins([]string{"Localhost:8080"})
	r := makeRequest("http://localhost:8080")
	if !gs.checkOrigin(r) {
		t.Error("checkOrigin with case-differing origin = false; want true")
	}
}
