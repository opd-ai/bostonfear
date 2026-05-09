package ws

import (
	"log"
	"net"
	"net/http"
)

// RouteHandlers groups the HTTP handlers needed by the server transport layer.
type RouteHandlers struct {
	WebSocket http.Handler
	Health    http.Handler
	Metrics   http.Handler
}

// SetupServer registers the server routes on a dedicated ServeMux and serves HTTP
// traffic on the provided listener.
func SetupServer(listener net.Listener, handlers RouteHandlers) error {
	mux := http.NewServeMux()
	mux.Handle("/ws", handlers.WebSocket)
	mux.Handle("/health", handlers.Health)
	mux.Handle("/metrics", handlers.Metrics)

	server := &http.Server{Handler: mux}
	log.Printf("Arkham Horror server starting on %s", listener.Addr().String())
	log.Printf("WebSocket endpoint: ws://localhost%s/ws", listener.Addr().String())
	log.Printf("Health check: http://localhost%s/health", listener.Addr().String())
	log.Printf("Prometheus metrics: http://localhost%s/metrics", listener.Addr().String())

	return server.Serve(listener)
}
