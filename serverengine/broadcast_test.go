package serverengine

import (
	"net"
	"testing"
)

func TestSnapshotConnections_ReturnsStableCopy(t *testing.T) {
	gs := NewGameServer()

	addr1 := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50101}
	addr2 := &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 50102}

	c1 := &stubConn{local: addr1, remote: addr1}
	c2 := &stubConn{local: addr2, remote: addr2}

	gs.mutex.Lock()
	gs.connections[addr1.String()] = c1
	gs.connections[addr2.String()] = c2
	gs.mutex.Unlock()

	snapshot := gs.snapshotConnections()
	if len(snapshot) != 2 {
		t.Fatalf("expected 2 snapshot entries, got %d", len(snapshot))
	}

	gs.mutex.Lock()
	delete(gs.connections, addr1.String())
	gs.mutex.Unlock()

	if len(snapshot) != 2 {
		t.Fatalf("snapshot should remain stable after map mutation, got %d entries", len(snapshot))
	}
}
