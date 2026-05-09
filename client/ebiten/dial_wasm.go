//go:build js

package ebiten

import (
	"fmt"
	"sync"
	"syscall/js"
)

// dialWebSocket opens a WebSocket connection using the browser WebSocket API
// via syscall/js.  gorilla/websocket cannot be used in WASM because it
// attempts a raw TCP dial which is not available inside the browser sandbox.
//
// This function blocks until the connection is open (onopen fires) or fails
// (onerror fires before onopen).  It is safe to call from a goroutine; the
// Go WASM scheduler parks the goroutine while waiting, allowing the JS event
// loop to deliver WebSocket callbacks.
func dialWebSocket(url string) (wsConn, error) {
	ws := js.Global().Get("WebSocket").New(url)
	if ws.IsNull() || ws.IsUndefined() {
		return nil, fmt.Errorf("WebSocket API unavailable")
	}

	openCh := make(chan struct{}, 1)
	errCh := make(chan error, 1)

	conn := &jsWSConn{
		ws:   ws,
		msgs: make(chan []byte, 64),
		done: make(chan struct{}),
	}

	conn.openCb = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		select {
		case openCh <- struct{}{}:
		default:
		}
		return nil
	})

	conn.msgCb = js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		data := args[0].Get("data").String()
		select {
		case conn.msgs <- []byte(data):
		default:
			// Drop message if buffer is full rather than blocking the JS event loop.
		}
		return nil
	})

	conn.errCb = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		select {
		case errCh <- fmt.Errorf("WebSocket error"):
		default:
		}
		return nil
	})

	conn.closeCb = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		conn.closeOnce.Do(func() { close(conn.done) })
		return nil
	})

	ws.Set("onopen", conn.openCb)
	ws.Set("onmessage", conn.msgCb)
	ws.Set("onerror", conn.errCb)
	ws.Set("onclose", conn.closeCb)

	// Park until the browser WebSocket is open or an error fires.
	select {
	case <-openCh:
		return conn, nil
	case err := <-errCh:
		conn.release()
		return nil, err
	}
}

// jsWSConn wraps a browser WebSocket (js.Value) as a wsConn.
type jsWSConn struct {
	ws        js.Value
	msgs      chan []byte
	done      chan struct{}
	closeOnce sync.Once

	// js.Func values must be released when no longer needed to allow GC.
	openCb  js.Func
	msgCb   js.Func
	errCb   js.Func
	closeCb js.Func
}

// ReadMessage blocks until a text message arrives or the connection closes.
// messageType is always wsTextMessage (1); it is returned for interface
// compatibility with gorilla/websocket callers.
func (c *jsWSConn) ReadMessage() (int, []byte, error) {
	select {
	case data := <-c.msgs:
		return wsTextMessage, data, nil
	case <-c.done:
		return 0, nil, fmt.Errorf("WebSocket closed")
	}
}

// WriteMessage sends data as a text frame.  Calling send() on a closed
// browser WebSocket will panic in JS; we recover and return an error instead.
func (c *jsWSConn) WriteMessage(_ int, data []byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("WebSocket send panic: %v", r)
		}
	}()
	select {
	case <-c.done:
		return fmt.Errorf("WebSocket closed")
	default:
	}
	c.ws.Call("send", string(data))
	return nil
}

// Close closes the browser WebSocket and releases all JS callback functions.
func (c *jsWSConn) Close() error {
	c.closeOnce.Do(func() {
		c.ws.Call("close")
		close(c.done)
	})
	c.release()
	return nil
}

// release frees js.Func values to allow garbage collection.
func (c *jsWSConn) release() {
	c.openCb.Release()
	c.msgCb.Release()
	c.errCb.Release()
	c.closeCb.Release()
}
