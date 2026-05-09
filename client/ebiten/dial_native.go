//go:build !js

package ebiten

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// dialWebSocket opens a WebSocket connection using gorilla/websocket.
// On native (non-WASM) targets, gorilla performs a standard TCP dial and
// HTTP Upgrade handshake.  The returned wsConn satisfies the interface used
// by reconnectLoop; *websocket.Conn already implements ReadMessage /
// WriteMessage / Close so no wrapper is needed.
func dialWebSocket(url string) (wsConn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{})
	if err != nil {
		return nil, err
	}
	return conn, nil
}
