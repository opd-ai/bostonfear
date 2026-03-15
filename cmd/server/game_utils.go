package main

import (
	"log"
	"net"
	"net/http"
)

func setupServer(listener net.Listener, gameServer *GameServer) error {
	// Set up HTTP handlers
	http.HandleFunc("/ws", gameServer.handleWebSocket)
	http.HandleFunc("/health", gameServer.handleHealthCheck)
	http.HandleFunc("/metrics", gameServer.handleMetrics)
	http.HandleFunc("/dashboard", gameServer.handleDashboard)
	http.Handle("/", http.FileServer(http.Dir(clientDir+"/")))

	// Start HTTP server with the provided listener
	server := &http.Server{}
	log.Printf("Arkham Horror server starting on %s", listener.Addr().String())
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", listener.Addr().String())
	log.Printf("Health check: http://localhost%s/health", listener.Addr().String())
	log.Printf("Prometheus metrics: http://localhost%s/metrics", listener.Addr().String())
	log.Printf("Performance Dashboard: http://localhost%s/dashboard", listener.Addr().String())
	log.Printf("Client available at: http://localhost%s", listener.Addr().String())

	return server.Serve(listener)
}
