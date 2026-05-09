package main

import (
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/opd-ai/bostonfear/monitoring"
	"github.com/opd-ai/bostonfear/serverengine"
	transportws "github.com/opd-ai/bostonfear/transport/ws"
)

// clientDir is the path to the client assets directory, relative to the repository root.
const clientDir = "./client"

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

	handlers := transportws.RouteHandlers{
		WebSocket: transportws.NewWebSocketHandler(gameServer),
		Health:    monitoring.HealthHandler(gameServer),
		Metrics:   monitoring.MetricsHandler(gameServer),
		Dashboard: monitoring.DashboardHandler(clientDir),
		Static:    http.FileServer(http.Dir(clientDir + "/")),
	}

	// Setup and start server with proper interface usage
	if err := transportws.SetupServer(listener, handlers); err != nil {
		return fmt.Errorf("setup server: %w", err)
	}

	return nil
}
