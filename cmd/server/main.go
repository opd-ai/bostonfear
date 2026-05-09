package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/opd-ai/bostonfear/monitoring"
	"github.com/opd-ai/bostonfear/serverengine/arkhamhorror"
	commonruntime "github.com/opd-ai/bostonfear/serverengine/common/runtime"
	"github.com/opd-ai/bostonfear/serverengine/eldersign"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror"
	"github.com/opd-ai/bostonfear/serverengine/finalhour"
	transportws "github.com/opd-ai/bostonfear/transport/ws"
)

// clientDir is the path to the client assets directory, relative to the repository root.
const clientDir = "./client"

// Main server setup using net.Listener interface.
func main() {
	if err := run(); err != nil {
		log.Fatalf("Server startup error: %v", err)
	}
}

func run() error {
	registry := commonruntime.NewRegistry()
	registry.MustRegister(arkhamhorror.NewModule())
	registry.MustRegister(eldersign.NewModule())
	registry.MustRegister(eldritchhorror.NewModule())
	registry.MustRegister(finalhour.NewModule())

	gameID := strings.ToLower(strings.TrimSpace(os.Getenv("BOSTONFEAR_GAME")))
	if gameID == "" {
		gameID = "arkhamhorror"
	}

	module, ok := registry.Get(gameID)
	if !ok {
		return fmt.Errorf("unknown game module %q (available: %v)", gameID, registry.Keys())
	}

	gameEngine, err := module.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to initialize %s engine: %w", module.Key(), err)
	}

	log.Printf("Loaded game module: %s (%s)", module.Key(), module.Description())
	if err := gameEngine.Start(); err != nil {
		return fmt.Errorf("failed to start game server: %w", err)
	}

	// Create listener using net.Listener interface
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	defer listener.Close()

	handlers := transportws.RouteHandlers{
		WebSocket: transportws.NewWebSocketHandler(gameEngine),
		Health:    monitoring.HealthHandler(gameEngine),
		Metrics:   monitoring.MetricsHandler(gameEngine),
		Dashboard: monitoring.DashboardHandler(clientDir),
		Static:    http.FileServer(http.Dir(clientDir + "/")),
	}

	// Setup and start server with proper interface usage
	if err := transportws.SetupServer(listener, handlers); err != nil {
		return fmt.Errorf("setup server: %w", err)
	}

	return nil
}
