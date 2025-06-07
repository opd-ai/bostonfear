// filepath: /workspaces/bostonfear/cmd/server/main.go
package main

import (
	"log"
	"net"
)

// Main server setup using net.Listener interface
// Moved from: main.go (original location)
func main() {
	// Initialize random number generator (Go 1.20+ compatible)
	// No need to call rand.Seed in modern Go versions

	// Create game server
	gameServer := NewGameServer()
	if err := gameServer.Start(); err != nil {
		log.Fatalf("Failed to start game server: %v", err)
	}

	// Create listener using net.Listener interface
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Setup and start server with proper interface usage
	if err := setupServer(listener, gameServer); err != nil {
		log.Fatalf("Server startup error: %v", err)
	}
}
