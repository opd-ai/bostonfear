package ws

import (
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

// connectionWrapper adapts a Gorilla WebSocket to net.Conn for engine APIs.
// It is package-local so websocket types do not leak across package boundaries.
type connectionWrapper struct {
	ws          *websocket.Conn
	localAddr   net.Addr
	remoteAddr  net.Addr
	displayName string
}

func newConnectionWrapper(wsConn *websocket.Conn, localAddr, remoteAddr net.Addr, displayName string) net.Conn {
	return &connectionWrapper{
		ws:          wsConn,
		localAddr:   localAddr,
		remoteAddr:  remoteAddr,
		displayName: displayName,
	}
}

func (c *connectionWrapper) Read(b []byte) (n int, err error) {
	_, data, err := c.ws.ReadMessage()
	if err != nil {
		return 0, err
	}
	n = copy(b, data)
	return n, nil
}

func (c *connectionWrapper) Write(b []byte) (n int, err error) {
	if err := c.ws.WriteMessage(websocket.TextMessage, b); err != nil {
		return 0, err
	}
	return len(b), nil
}

func (c *connectionWrapper) Close() error {
	return c.ws.Close()
}

func (c *connectionWrapper) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *connectionWrapper) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *connectionWrapper) DisplayName() string {
	return c.displayName
}

func (c *connectionWrapper) SetDeadline(t time.Time) error {
	if err := c.ws.SetReadDeadline(t); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}
	return c.ws.SetWriteDeadline(t)
}

func (c *connectionWrapper) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

func (c *connectionWrapper) SetWriteDeadline(t time.Time) error {
	return c.ws.SetWriteDeadline(t)
}
