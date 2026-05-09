package cmd

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/opd-ai/bostonfear/monitoring"
	"github.com/opd-ai/bostonfear/serverengine/arkhamhorror"
	commonruntime "github.com/opd-ai/bostonfear/serverengine/common/runtime"
	"github.com/opd-ai/bostonfear/serverengine/eldersign"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror"
	"github.com/opd-ai/bostonfear/serverengine/finalhour"
	transportws "github.com/opd-ai/bostonfear/transport/ws"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewServerCommand wraps the existing server startup flow in a Cobra command.
func NewServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the multiplayer game server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServer(cmd)
		},
	}

	cmd.Flags().String("game", "", "Game module key (overrides BOSTONFEAR_GAME)")
	cmd.Flags().String("listen", "", "TCP listen address (overrides --host/--port when set)")
	cmd.Flags().String("host", "", "TCP listen host (default all interfaces)")
	cmd.Flags().Int("port", 8080, "TCP listen port")
	cmd.Flags().StringSlice("allowed-origins", nil, "Allowed WebSocket upgrade origins")

	_ = viper.BindPFlag("server.game", cmd.Flags().Lookup("game"))
	_ = viper.BindPFlag("server.listen", cmd.Flags().Lookup("listen"))
	_ = viper.BindPFlag("server.host", cmd.Flags().Lookup("host"))
	_ = viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))
	_ = viper.BindPFlag("network.allowed-origins", cmd.Flags().Lookup("allowed-origins"))

	return cmd
}

func runServer(cmd *cobra.Command) error {
	registry := commonruntime.NewRegistry()
	registry.MustRegister(arkhamhorror.NewModule())
	registry.MustRegister(eldersign.NewModule())
	registry.MustRegister(eldritchhorror.NewModule())
	registry.MustRegister(finalhour.NewModule())

	gameID := strings.ToLower(strings.TrimSpace(viper.GetString("server.game")))
	if gameID == "" {
		gameID = strings.ToLower(strings.TrimSpace(os.Getenv("BOSTONFEAR_GAME")))
	}
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

	allowedOrigins := viper.GetStringSlice("network.allowed-origins")
	if len(allowedOrigins) > 0 {
		gameEngine.SetAllowedOrigins(allowedOrigins)
	}

	if err := gameEngine.Start(); err != nil {
		return fmt.Errorf("failed to start game server: %w", err)
	}

	listenAddr := resolveListenAddress(cmd)
	if listenAddr == "" {
		return fmt.Errorf("invalid listen configuration")
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	defer listener.Close()

	wasmDir := "./client/wasm"
	handlers := transportws.RouteHandlers{
		WebSocket: transportws.NewWebSocketHandler(gameEngine),
		Health:    monitoring.HealthHandler(gameEngine),
		Metrics:   monitoring.MetricsHandler(gameEngine),
		Play: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/":
				serveIndexWithServerURL(w, r, wasmDir)
			case "/favicon.ico":
				// Avoid noisy browser 404s when no favicon is bundled.
				w.WriteHeader(http.StatusNoContent)
			case "/wasm_exec.js":
				serveWASMExecJS(w, r, wasmDir)
			case "/game.wasm":
				w.Header().Set("Content-Type", "application/wasm")
				http.ServeFile(w, r, wasmDir+"/game.wasm")
			default:
				http.NotFound(w, r)
			}
		}),
		WASMAssets: http.StripPrefix("/wasm/", http.FileServer(http.Dir(wasmDir))),
	}

	if err := transportws.SetupServer(listener, handlers); err != nil {
		return fmt.Errorf("setup server: %w", err)
	}

	return nil
}

func resolveListenAddress(cmd *cobra.Command) string {
	listenAddr := strings.TrimSpace(viper.GetString("server.listen"))
	host := strings.TrimSpace(viper.GetString("server.host"))
	port := viper.GetInt("server.port")
	if port == 0 {
		port = 8080
	}

	listenChanged := false
	hostOrPortChanged := false
	if cmd != nil {
		listenChanged = cmd.Flags().Changed("listen")
		hostOrPortChanged = cmd.Flags().Changed("host") || cmd.Flags().Changed("port")
	}

	// CLI precedence:
	// 1. --listen
	// 2. --host/--port
	// 3. config/env server.listen
	// 4. config/env server.host + server.port
	if listenChanged {
		return listenAddr
	}
	if !hostOrPortChanged && listenAddr != "" {
		return listenAddr
	}

	// net.JoinHostPort with empty host produces ":port" for all interfaces.

	if port <= 0 || port > 65535 {
		return ""
	}

	return net.JoinHostPort(host, fmt.Sprintf("%d", port))
}

// serveIndexWithServerURL reads index.html and injects a window.__serverURL global
// so the WASM client always dials the WebSocket on the same host:port it was served
// from, preventing the port-mismatch that occurs when a separate file server is used.
func serveIndexWithServerURL(w http.ResponseWriter, r *http.Request, wasmDir string) {
	data, err := os.ReadFile(wasmDir + "/index.html")
	if err != nil {
		http.Error(w, "index.html not found", http.StatusNotFound)
		return
	}

	proto := "ws"
	if r.TLS != nil {
		proto = "wss"
	}
	// sanitizeHost ensures only valid host:port characters are embedded in the script.
	host := sanitizeHost(r.Host)
	injection := fmt.Sprintf(`<script>window.__serverURL="%s://%s/ws";</script>`+"\n  ", proto, host)

	// Inject before the wasm_exec.js <script> tag so __serverURL is defined first.
	data = bytes.Replace(
		data,
		[]byte(`<script src="wasm_exec.js">`),
		append([]byte(injection), []byte(`<script src="wasm_exec.js">`)...),
		1,
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

// sanitizeHost strips characters that are not valid in a host:port value
// to prevent injection into the inline script block.
func sanitizeHost(host string) string {
	var b strings.Builder
	for _, ch := range host {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
			(ch >= '0' && ch <= '9') || ch == '.' || ch == ':' || ch == '-' || ch == '[' || ch == ']' {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

func serveWASMExecJS(w http.ResponseWriter, r *http.Request, wasmDir string) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")

	localPath := filepath.Join(wasmDir, "wasm_exec.js")
	if _, err := os.Stat(localPath); err == nil {
		http.ServeFile(w, r, localPath)
		return
	}

	// Go 1.22+ ships wasm_exec.js in GOROOT/lib/wasm, older versions used misc/wasm.
	goRoot := runtime.GOROOT()
	fallbacks := []string{
		filepath.Join(goRoot, "lib", "wasm", "wasm_exec.js"),
		filepath.Join(goRoot, "misc", "wasm", "wasm_exec.js"),
	}

	for _, path := range fallbacks {
		if _, err := os.Stat(path); err == nil {
			http.ServeFile(w, r, path)
			return
		}
	}

	http.NotFound(w, r)
}
