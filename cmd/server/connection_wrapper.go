package main

import (
	"fmt"
	"net"
	"time"

	"github.com/gorilla/websocket"
)

// ConnectionWrapper implements net.Conn interface around a WebSocket connection,
// enabling interface-based testing and abstraction over the underlying transport.
type ConnectionWrapper struct {
	ws   *websocket.Conn
	addr net.Addr
}

// NewConnectionWrapper creates a new connection wrapper instance
// Moved from: main.go
func NewConnectionWrapper(ws *websocket.Conn, addr net.Addr) *ConnectionWrapper {
	return &ConnectionWrapper{
		ws:   ws,
		addr: addr,
	}
}

// Read implements net.Conn Read method for WebSocket message reading
// Moved from: main.go
func (c *ConnectionWrapper) Read(b []byte) (n int, err error) {
	_, data, err := c.ws.ReadMessage()
	if err != nil {
		return 0, err
	}
	copy(b, data)
	return len(data), nil
}

// Write implements net.Conn Write method for WebSocket message writing
// Moved from: main.go
func (c *ConnectionWrapper) Write(b []byte) (n int, err error) {
	err = c.ws.WriteMessage(websocket.TextMessage, b)
	return len(b), err
}

// Close implements net.Conn Close method for WebSocket connection closure
// Moved from: main.go
func (c *ConnectionWrapper) Close() error {
	return c.ws.Close()
}

// LocalAddr implements net.Conn LocalAddr method
// Moved from: main.go
func (c *ConnectionWrapper) LocalAddr() net.Addr {
	return c.addr
}

// RemoteAddr implements net.Conn RemoteAddr method
// Moved from: main.go
func (c *ConnectionWrapper) RemoteAddr() net.Addr {
	return c.addr
}

// SetDeadline implements net.Conn SetDeadline by delegating to the underlying
// WebSocket connection, setting both read and write deadlines simultaneously.
func (c *ConnectionWrapper) SetDeadline(t time.Time) error {
	if err := c.ws.SetReadDeadline(t); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}
	return c.ws.SetWriteDeadline(t)
}

// SetReadDeadline implements net.Conn SetReadDeadline by delegating to the
// underlying WebSocket connection, enabling the 30-second inactivity timeout.
func (c *ConnectionWrapper) SetReadDeadline(t time.Time) error {
	return c.ws.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn SetWriteDeadline by delegating to the
// underlying WebSocket connection.
func (c *ConnectionWrapper) SetWriteDeadline(t time.Time) error {
	return c.ws.SetWriteDeadline(t)
}
