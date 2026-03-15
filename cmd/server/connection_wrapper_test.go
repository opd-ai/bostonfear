package main

import (
	"net"
	"testing"
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
