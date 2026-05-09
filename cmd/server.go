package cmd

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/opd-ai/bostonfear/monitoring"
	"github.com/opd-ai/bostonfear/serverengine"
	"github.com/opd-ai/bostonfear/serverengine/arkhamhorror"
	arkhamcontent "github.com/opd-ai/bostonfear/serverengine/arkhamhorror/content"
	commonruntime "github.com/opd-ai/bostonfear/serverengine/common/runtime"
	"github.com/opd-ai/bostonfear/serverengine/eldersign"
	"github.com/opd-ai/bostonfear/serverengine/eldritchhorror"
	"github.com/opd-ai/bostonfear/serverengine/finalhour"
	transportws "github.com/opd-ai/bostonfear/transport/ws"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var wasmBuildMu sync.Mutex

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
	if err := prepareGameStartup(gameID, "."); err != nil {
		return err
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
	warnIfWASMBuildUnavailable(wasmDir)
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
				serveGameWASM(w, r, wasmDir)
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

func prepareGameStartup(gameID, repoRoot string) error {
	if gameID != "arkhamhorror" {
		serverengine.SetStartupScenarioDefaultID("")
		return fmt.Errorf("game module %q is registered but not runnable yet; choose arkhamhorror", gameID)
	}

	if err := arkhamcontent.EnsureNightglassContentInstalled(repoRoot); err != nil {
		return fmt.Errorf("install embedded arkham content: %w", err)
	}

	rawScenarioID := strings.TrimSpace(viper.GetString("scenario.default_id"))
	selectedScenarioID, err := arkhamcontent.ResolveNightglassScenarioID(repoRoot, rawScenarioID)
	if err != nil {
		return fmt.Errorf("resolve default scenario: %w", err)
	}
	serverengine.SetStartupScenarioDefaultID(selectedScenarioID)
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
	updated := bytes.Replace(
		data,
		[]byte(`<script src="wasm_exec.js">`),
		append([]byte(injection), []byte(`<script src="wasm_exec.js">`)...),
		1,
	)
	if bytes.Equal(updated, data) {
		// Fallback when markup changes and the exact script tag isn't present.
		updated = append([]byte(injection), data...)
	}
	data = updated

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

// serveGameWASM serves game.wasm and builds it on-demand if the binary is
// missing. This removes the mandatory manual GOOS=js GOARCH=wasm build step
// during local development.
func serveGameWASM(w http.ResponseWriter, r *http.Request, wasmDir string) {
	wasmPath := filepath.Join(wasmDir, "game.wasm")
	if _, err := os.Stat(wasmPath); os.IsNotExist(err) {
		if err := buildGameWASM(wasmPath); err != nil {
			http.Error(w, fmt.Sprintf("failed to build game.wasm: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/wasm")
	http.ServeFile(w, r, wasmPath)
}

// buildGameWASM compiles ./cmd/web into the target wasmPath. Calls are
// serialized so concurrent /game.wasm requests do not race the output file.
func buildGameWASM(wasmPath string) error {
	wasmBuildMu.Lock()
	defer wasmBuildMu.Unlock()

	// Re-check inside lock in case another request built it first.
	if _, err := os.Stat(wasmPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(wasmPath), 0o755); err != nil {
		return fmt.Errorf("create wasm output dir: %w", err)
	}

	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("go toolchain not found on host; prebuild and ship %s (e.g. GOOS=js GOARCH=wasm go build -o %s ./cmd/web/)", wasmPath, wasmPath)
	}

	cmd := exec.Command("go", "build", "-o", wasmPath, "./cmd/web/")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go build failed: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	log.Printf("Built game.wasm at runtime: %s", wasmPath)
	return nil
}

// warnIfWASMBuildUnavailable logs a startup warning when no prebuilt game.wasm
// is present and runtime auto-build cannot run due to missing Go toolchain.
func warnIfWASMBuildUnavailable(wasmDir string) {
	wasmPath := filepath.Join(wasmDir, "game.wasm")
	if _, err := os.Stat(wasmPath); err == nil {
		return
	}
	if _, err := exec.LookPath("go"); err != nil {
		log.Printf("Warning: %s missing and Go toolchain unavailable; /game.wasm requests will fail. Ship a prebuilt game.wasm artifact.", wasmPath)
	}
}
