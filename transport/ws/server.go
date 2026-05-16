package ws

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/opd-ai/bostonfear/serverengine/common/logging"
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
	return SetupServerWithContext(context.Background(), listener, handlers)
}

// SetupServerWithContext registers routes and serves HTTP traffic until the
// listener fails or ctx cancellation triggers a graceful shutdown.
func SetupServerWithContext(ctx context.Context, listener net.Listener, handlers RouteHandlers) error {
	if ctx == nil {
		return errors.New("context is nil")
	}

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
	logging.Info("Arkham Horror server starting", "address", listener.Addr().String())
	logging.Info("Game client", "url", "http://"+addr+"/")
	logging.Info("WebSocket endpoint", "url", "ws://"+addr+"/ws")
	logging.Info("Health check", "url", "http://"+addr+"/health")
	logging.Info("Prometheus metrics", "url", "http://"+addr+"/metrics")

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logging.Error("HTTP server shutdown error", "error", err)
		}
	}()

	err := server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) && ctx.Err() != nil {
		return nil
	}
	return err
}
