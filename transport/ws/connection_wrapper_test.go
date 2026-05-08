package ws

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/websocket"
)

type mockAddr struct{ addr string }

func (m *mockAddr) Network() string { return "tcp" }
func (m *mockAddr) String() string  { return m.addr }

func TestConnectionWrapper_LocalRemoteAddrDistinct(t *testing.T) {
	local := &mockAddr{addr: "127.0.0.1:8080"}
	remote := &mockAddr{addr: "192.168.1.1:54321"}

	cw := &connectionWrapper{
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

func TestConnectionWrapper_AddrInterface(t *testing.T) {
	local := &mockAddr{addr: "0.0.0.0:9000"}
	remote := &mockAddr{addr: "10.0.0.1:12345"}
	cw := &connectionWrapper{localAddr: local, remoteAddr: remote}

	var _ net.Addr = cw.LocalAddr()
	var _ net.Addr = cw.RemoteAddr()
}

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

func TestConnectionWrapper_ReadExact(t *testing.T) {
	serverWS, clientWS, cleanup := newTestWSPair(t)
	defer cleanup()

	msg := []byte("hello world")
	if err := clientWS.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, 64)
	cw := &connectionWrapper{ws: serverWS, localAddr: &mockAddr{"s"}, remoteAddr: &mockAddr{"c"}}
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

func TestConnectionWrapper_ReadTruncation(t *testing.T) {
	serverWS, clientWS, cleanup := newTestWSPair(t)
	defer cleanup()

	msg := []byte("hello world")
	if err := clientWS.WriteMessage(websocket.TextMessage, msg); err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, 5)
	cw := &connectionWrapper{ws: serverWS, localAddr: &mockAddr{"s"}, remoteAddr: &mockAddr{"c"}}
	n, err := cw.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n > len(buf) {
		t.Errorf("Read returned n=%d which exceeds buffer size %d", n, len(buf))
	}
	want := 5
	if n != want {
		t.Errorf("Read returned n=%d, want %d", n, want)
	}
	if string(buf[:n]) != string(msg[:n]) {
		t.Errorf("buf[:n] = %q, want %q", buf[:n], msg[:n])
	}
}
