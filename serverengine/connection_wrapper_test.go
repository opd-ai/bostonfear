package serverengine

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

// mockAddr is a simple net.Addr for testing.
type mockAddr struct{ addr string }

func (m *mockAddr) Network() string { return "tcp" }
func (m *mockAddr) String() string  { return m.addr }

// TestConnectionWrapper_LocalRemoteAddrDistinct asserts that LocalAddr and RemoteAddr
// return different addresses, verifying the net.Conn contract is satisfied.
func TestConnectionWrapper_LocalRemoteAddrDistinct(t *testing.T) {
	local := &mockAddr{addr: "127.0.0.1:8080"}
	remote := &mockAddr{addr: "192.168.1.1:54321"}

	cw := &ConnectionWrapper{
		ws:         nil,
		localAddr:  local,
		remoteAddr: remote,
	}

	if cw.LocalAddr().String() == cw.RemoteAddr().String() {
		t.Errorf("LocalAddr() == RemoteAddr() = %q; want distinct addresses", cw.LocalAddr().String())
	}
	if cw.LocalAddr().String() != local.addr {
		t.Errorf("LocalAddr() = %q; want %q", cw.LocalAddr().String(), local.addr)
	}
	if cw.RemoteAddr().String() != remote.addr {
		t.Errorf("RemoteAddr() = %q; want %q", cw.RemoteAddr().String(), remote.addr)
	}
}

// TestConnectionWrapper_AddrInterface verifies both addresses implement net.Addr.
func TestConnectionWrapper_AddrInterface(t *testing.T) {
	local := &mockAddr{addr: "0.0.0.0:9000"}
	remote := &mockAddr{addr: "10.0.0.1:12345"}
	cw := &ConnectionWrapper{localAddr: local, remoteAddr: remote}

	var _ net.Addr = cw.LocalAddr()
	var _ net.Addr = cw.RemoteAddr()
}

// newTestWSPair creates an in-process WebSocket server and connected client.
// It returns the server-side and client-side *websocket.Conn values and a cleanup func.
func newTestWSPair(t *testing.T) (serverConn, clientConn *websocket.Conn, cleanup func()) {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	serverCh := make(chan *websocket.Conn, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		serverCh <- conn
	}))

	wsURL := "ws" + srv.URL[len("http"):]
	clientWS, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}
	serverWS := <-serverCh
	cleanup = func() {
		clientWS.Close()
		serverWS.Close()
		srv.Close()
	}
	return serverWS, clientWS, cleanup
}

// TestConnectionWrapper_ReadExact verifies that Read returns n == len(data) when
// the caller's buffer is large enough to hold the full message.
func TestConnectionWrapper_ReadExact(t *testing.T) {
	serverWS, clientWS, cleanup := newTestWSPair(t)
	defer cleanup()

	msg := []byte("hello world")
	if err := clientWS.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, 64) // larger than message
	cw := &ConnectionWrapper{ws: serverWS, localAddr: &mockAddr{"s"}, remoteAddr: &mockAddr{"c"}}
	n, err := cw.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != len(msg) {
		t.Errorf("Read returned n=%d, want %d", n, len(msg))
	}
	if string(buf[:n]) != string(msg) {
		t.Errorf("buf[:n] = %q, want %q", buf[:n], msg)
	}
}

// TestConnectionWrapper_ReadTruncation verifies that Read returns n == len(b) (not
// len(data)) when the caller's buffer is smaller than the incoming message, honouring
// the net.Conn contract that n never exceeds len(b).
func TestConnectionWrapper_ReadTruncation(t *testing.T) {
	serverWS, clientWS, cleanup := newTestWSPair(t)
	defer cleanup()

	msg := []byte("hello world") // 11 bytes
	if err := clientWS.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, 5) // smaller than message
	cw := &ConnectionWrapper{ws: serverWS, localAddr: &mockAddr{"s"}, remoteAddr: &mockAddr{"c"}}
	n, err := cw.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	// n must not exceed len(b)
	if n > len(buf) {
		t.Errorf("Read returned n=%d which exceeds buffer size %d", n, len(buf))
	}
	// n must equal min(len(b), len(msg)) = 5
	want := 5
	if n != want {
		t.Errorf("Read returned n=%d, want %d", n, want)
	}
	if string(buf[:n]) != string(msg[:n]) {
		t.Errorf("buf[:n] = %q, want %q", buf[:n], msg[:n])
	}
}
