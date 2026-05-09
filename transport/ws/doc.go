// Package ws implements the HTTP and WebSocket transport layer for the Arkham Horror game server.
//
// This package owns all network-level abstractions: WebSocket upgrade, connection wrapping,
// HTTP route registration, and origin validation. It decouples the game engine from
// transport-specific concerns by treating all connections as net.Conn interfaces.
//
// Architecture:
//
// - connectionWrapper: Adapts Gorilla WebSocket to net.Conn, hiding concrete transport types
// - websocket_handler: HTTP handler for WebSocket upgrade with origin validation
// - server: HTTP server setup and route registration on a net.Listener
// - SessionEngine: Transport-neutral interface that the engine must implement
//
// The package follows idiomatic Go by using net.Conn, net.Listener, and net.Addr interfaces instead
// of concrete types (e.g., net.TCPConn), enabling testing with mocks and supporting multiple
// future protocols without engine changes.
//
// Example usage:
//
//	gameEngine := serverengine.NewGameServer()
//	listener, _ := net.Listen("tcp", ":8080")
//	handlers := RouteHandlers{
//		WebSocket: NewWebSocketHandler(gameEngine),
//		Health:    healthHandler,
//		Metrics:   metricsHandler,
//		Play:      playHandler,
//		WASMAssets: wasmAssetsHandler,
//	}
//	SetupServer(listener, handlers) // blocks until shutdown
package ws
