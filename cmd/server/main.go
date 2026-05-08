// filepath: /workspaces/bostonfear/cmd/server/main.go
package main

import (
	"fmt"
	"log"
	"net"

	"github.com/opd-ai/bostonfear/serverengine"
)

// Main server setup using net.Listener interface
// Moved from: main.go (original location)
func main() {
	if err := run(); err != nil {
		log.Fatalf("Server startup error: %v", err)
	}
}

func run() error {
	// Initialize random number generator (Go 1.20+ compatible)
	// No need to call rand.Seed in modern Go versions

	// Create game server
	gameServer := serverengine.NewGameServer()
	if err := gameServer.Start(); err != nil {
		return fmt.Errorf("failed to start game server: %w", err)
	}

	// Create listener using net.Listener interface
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	defer listener.Close()

	// Setup and start server with proper interface usage
	if err := serverengine.SetupServer(listener, gameServer); err != nil {
		return fmt.Errorf("setup server: %w", err)
	}

	return nil
}
