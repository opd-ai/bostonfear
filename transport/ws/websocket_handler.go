package ws

import (
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/websocket"
)

// SessionEngine defines the transport-neutral session lifecycle surface expected
// by the WebSocket transport adapter.
// Implementations must be safe for concurrent calls because each upgraded
// session is handled in its own goroutine.
type SessionEngine interface {
	HandleConnection(conn net.Conn, reconnectToken string) error
	AllowedOrigins() []string
}

// NewWebSocketHandler returns an HTTP handler that upgrades to WebSocket and
// delegates connection lifecycle to the provided engine.
func NewWebSocketHandler(engine SessionEngine) http.Handler {
	upgrader := websocket.Upgrader{}
	upgrader.CheckOrigin = makeCheckOrigin(engine)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("New WebSocket connection attempt from %s", r.RemoteAddr)

		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("WebSocket upgrade error: %v", err)
			return
		}

		remoteAddr := wsConn.RemoteAddr()
		localAddr := wsConn.NetConn().LocalAddr()
		conn := newConnectionWrapper(wsConn, localAddr, remoteAddr)
		reconnectToken := r.URL.Query().Get("token")

		go func() {
			if err := engine.HandleConnection(conn, reconnectToken); err != nil {
				log.Printf("Connection handling error: %v", err)
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
		log.Printf("WebSocket upgrade rejected: malformed origin %q", origin)
		return false
	}

	switch strings.ToLower(u.Scheme) {
	case "http", "https", "ws", "wss":
	default:
		log.Printf("WebSocket upgrade rejected: unsupported scheme in origin %q", origin)
		return false
	}

	hostLower := strings.ToLower(u.Host)
	for _, a := range allowedHosts {
		if strings.ToLower(a) == hostLower {
			return true
		}
	}

	log.Printf("WebSocket upgrade rejected: origin %q not in allowed list", origin)
	return false
}

func makeCheckOrigin(engine SessionEngine) func(*http.Request) bool {
	return func(r *http.Request) bool {
		return ValidateOrigin(r, engine.AllowedOrigins())
	}
}
