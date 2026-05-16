package ws

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/opd-ai/bostonfear/serverengine/common/logging"
)

// SessionEngine defines the transport-neutral session lifecycle surface expected
// by the WebSocket transport adapter.
//
// Implementations must be safe for concurrent calls because each upgraded
// WebSocket connection runs HandleConnection in its own goroutine.
//
// SessionEngine is a minimal interface (2 methods) designed to decouple the game engine
// from transport layer concerns. New transport adapters (TCP, in-process, gRPC) need
// only implement these two methods to integrate with any SessionEngine.
//
// This interface is a subset of serverengine/common/contracts.SessionHandler.
// It is transport-specific; for broader concerns (health checking, metrics),
// use the Engine interface from serverengine/common/contracts.
type SessionEngine interface {
	// HandleConnection manages a player session via the provided net.Conn.
	// conn must be non-nil and readable/writable. reconnectToken may be empty
	// (new player) or non-empty (restore disconnected player).
	// The method blocks until the connection closes or an error occurs.
	HandleConnection(conn net.Conn, reconnectToken string) error

	// AllowedOrigins returns the current list of permitted WebSocket upgrade origins.
	// An empty or nil list indicates permissive mode (any origin accepted).
	AllowedOrigins() []string
}

// NewWebSocketHandler returns an HTTP handler that upgrades to WebSocket and
// delegates connection lifecycle to the provided engine.
func NewWebSocketHandler(engine SessionEngine) http.Handler {
	upgrader := websocket.Upgrader{}
	upgrader.CheckOrigin = makeCheckOrigin(engine)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logging.Info("New WebSocket connection attempt", "remoteAddr", r.RemoteAddr)

		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logging.Error("WebSocket upgrade error", "error", err)
			return
		}

		remoteAddr := wsConn.RemoteAddr()
		localAddr := wsConn.NetConn().LocalAddr()
		displayName := r.URL.Query().Get("displayName")
		conn := newConnectionWrapper(wsConn, localAddr, remoteAddr, displayName)
		reconnectToken := r.URL.Query().Get("token")

		go func() {
			if err := engine.HandleConnection(conn, reconnectToken); err != nil {
				logging.Error("Connection handling error", "error", err)
			}
		}()
	})
}

// ValidateOrigin determines whether a request's origin should be accepted
// for WebSocket upgrade. It returns true if the origin is allowed or if the
// allowedHosts list is empty (permissive mode).
func ValidateOrigin(r *http.Request, allowedHosts []string) bool {
	if len(allowedHosts) == 0 {
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}

	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		logging.Warn("WebSocket upgrade rejected: malformed origin", "origin", origin)
		return false
	}

	switch strings.ToLower(u.Scheme) {
	case "http", "https", "ws", "wss":
	default:
		logging.Warn("WebSocket upgrade rejected: unsupported scheme in origin", "origin", origin)
		return false
	}

	hostLower := strings.ToLower(u.Host)
	for _, a := range allowedHosts {
		if strings.ToLower(a) == hostLower {
			return true
		}
	}

	logging.Warn("WebSocket upgrade rejected: origin not in allowed list", "origin", origin)
	return false
}

func makeCheckOrigin(engine SessionEngine) func(*http.Request) bool {
	return func(r *http.Request) bool {
		return ValidateOrigin(r, engine.AllowedOrigins())
	}
}
