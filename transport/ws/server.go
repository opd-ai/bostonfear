package ws

import (
	"log"
	"net"
	"net/http"
)

// RouteHandlers groups the HTTP handlers needed by the server transport layer.
type RouteHandlers struct {
	WebSocket  http.Handler
	Health     http.Handler
	Metrics    http.Handler
	Play       http.Handler
	WASMAssets http.Handler
}

// SetupServer registers the server routes on a dedicated ServeMux and serves HTTP
// traffic on the provided listener.
func SetupServer(listener net.Listener, handlers RouteHandlers) error {
	mux := http.NewServeMux()
	mux.Handle("/ws", handlers.WebSocket)
	mux.Handle("/health", handlers.Health)
	mux.Handle("/metrics", handlers.Metrics)
	if handlers.Play != nil {
		mux.Handle("/", handlers.Play)
	}
	if handlers.WASMAssets != nil {
		mux.Handle("/wasm/", handlers.WASMAssets)
	}

	server := &http.Server{Handler: mux}
	addr := listener.Addr().String()
	// Addr() on a TCP listener returns "[::]:port" for all-interfaces; normalise for display.
	displayHost := "localhost"
	if _, port, err := net.SplitHostPort(addr); err == nil {
		addr = net.JoinHostPort(displayHost, port)
	}
	log.Printf("Arkham Horror server starting on %s", listener.Addr().String())
	log.Printf("Game client: http://%s/", addr)
	log.Printf("WebSocket endpoint: ws://%s/ws", addr)
	log.Printf("Health check: http://%s/health", addr)
	log.Printf("Prometheus metrics: http://%s/metrics", addr)

	return server.Serve(listener)
}
