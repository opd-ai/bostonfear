package cmd

import (
	"fmt"
	"net"
	"os"
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

	handlers := transportws.RouteHandlers{
		WebSocket: transportws.NewWebSocketHandler(gameEngine),
		Health:    monitoring.HealthHandler(gameEngine),
		Metrics:   monitoring.MetricsHandler(gameEngine),
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
