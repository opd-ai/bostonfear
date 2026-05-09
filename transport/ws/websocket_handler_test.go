// Package ws — tests for WebSocket origin validation.
// Verifies that ValidateOrigin enforces the allowedOrigins list and falls back to
// permissive mode when the list is empty.
package ws

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

// TestValidateOrigin_Permissive verifies that an empty allowedHosts list accepts
// any origin, preserving the default local-development behaviour.
func TestValidateOrigin_Permissive(t *testing.T) {
	// No allowed hosts configured — every origin must be accepted.
	for _, origin := range []string{
		"http://localhost:8080",
		"http://evil.example.com",
		"https://anything.invalid",
		"",
	} {
		r := makeRequest(origin)
		if !ValidateOrigin(r, []string{}) {
			t.Errorf("ValidateOrigin(%q) = false; want true (permissive mode)", origin)
		}
	}
}

// TestValidateOrigin_NoOriginHeader verifies that requests without an Origin header
// are always accepted, even when allowedHosts is non-empty.
func TestValidateOrigin_NoOriginHeader(t *testing.T) {
	r := makeRequest("") // no Origin header
	allowedHosts := []string{"localhost:8080"}
	if !ValidateOrigin(r, allowedHosts) {
		t.Error("ValidateOrigin with no Origin header = false; want true")
	}
}

// TestValidateOrigin_AllowedOrigin verifies that a matching origin is accepted.
func TestValidateOrigin_AllowedOrigin(t *testing.T) {
	allowedHosts := []string{"localhost:8080", "mygame.example.com"}

	allowed := []string{
		"http://localhost:8080",
		"https://localhost:8080",
		"http://mygame.example.com",
		"https://mygame.example.com",
	}
	for _, origin := range allowed {
		r := makeRequest(origin)
		if !ValidateOrigin(r, allowedHosts) {
			t.Errorf("ValidateOrigin(%q) = false; want true", origin)
		}
	}
}

// TestValidateOrigin_RejectedOrigin verifies that an origin not in allowedHosts
// is rejected when the list is non-empty.
func TestValidateOrigin_RejectedOrigin(t *testing.T) {
	allowedHosts := []string{"localhost:8080"}

	rejected := []string{
		"http://evil.example.com",
		"http://localhost:9090",
		"https://attacker.invalid",
	}
	for _, origin := range rejected {
		r := makeRequest(origin)
		if ValidateOrigin(r, allowedHosts) {
			t.Errorf("ValidateOrigin(%q) = true; want false (origin not allowed)", origin)
		}
	}
}

// TestValidateOrigin_CaseInsensitive verifies that host comparison ignores case.
func TestValidateOrigin_CaseInsensitive(t *testing.T) {
	allowedHosts := []string{"Localhost:8080"}
	r := makeRequest("http://localhost:8080")
	if !ValidateOrigin(r, allowedHosts) {
		t.Error("ValidateOrigin with case-differing origin = false; want true")
	}
}

// TestValidateOrigin_MalformedOrigin verifies that a malformed Origin header is
// rejected (not panicked upon) when an allowedHosts list is configured.
func TestValidateOrigin_MalformedOrigin(t *testing.T) {
	allowedHosts := []string{"localhost:8080"}
	malformed := []string{
		"ht!tp://malformed",
		"://no-scheme",
		"no-slashes",
	}
	for _, origin := range malformed {
		r := makeRequest(origin)
		if ValidateOrigin(r, allowedHosts) {
			t.Errorf("ValidateOrigin(%q) = true; want false (malformed origin)", origin)
		}
	}
}

// TestValidateOrigin_UnsupportedScheme verifies that origins with non-HTTP(S)/WS(S)
// schemes are rejected when an allowedHosts list is configured.
func TestValidateOrigin_UnsupportedScheme(t *testing.T) {
	allowedHosts := []string{"localhost:8080"}
	unsupported := []string{
		"file://localhost:8080",
		"gopher://localhost:8080",
		"ftp://localhost:8080",
	}
	for _, origin := range unsupported {
		r := makeRequest(origin)
		if ValidateOrigin(r, allowedHosts) {
			t.Errorf("ValidateOrigin(%q) = true; want false (unsupported scheme)", origin)
		}
	}
}

// TestValidateOrigin_WebSocketSchemes verifies that ws:// and wss:// schemes are
// accepted in addition to http and https.
func TestValidateOrigin_WebSocketSchemes(t *testing.T) {
	allowedHosts := []string{"localhost:8080"}
	wsOrigins := []string{
		"ws://localhost:8080",
		"wss://localhost:8080",
	}
	for _, origin := range wsOrigins {
		r := makeRequest(origin)
		if !ValidateOrigin(r, allowedHosts) {
			t.Errorf("ValidateOrigin(%q) = false; want true (ws scheme)", origin)
		}
	}
}
