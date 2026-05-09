package cmd

import (
	"fmt"
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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewServerCommand wraps the existing server startup flow in a Cobra command.
func NewServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the multiplayer game server",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runServer()
		},
	}

	cmd.Flags().String("game", "", "Game module key (overrides BOSTONFEAR_GAME)")
	cmd.Flags().String("listen", ":8080", "TCP listen address")
	cmd.Flags().String("client-dir", "./client", "Path to browser client assets")
	cmd.Flags().StringSlice("allowed-origins", nil, "Allowed WebSocket upgrade origins")

	_ = viper.BindPFlag("server.game", cmd.Flags().Lookup("game"))
	_ = viper.BindPFlag("server.listen", cmd.Flags().Lookup("listen"))
	_ = viper.BindPFlag("server.client-dir", cmd.Flags().Lookup("client-dir"))
	_ = viper.BindPFlag("network.allowed-origins", cmd.Flags().Lookup("allowed-origins"))

	return cmd
}

func runServer() error {
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

	listenAddr := strings.TrimSpace(viper.GetString("server.listen"))
	if listenAddr == "" {
		listenAddr = ":8080"
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	defer listener.Close()

	clientDir := strings.TrimSpace(viper.GetString("server.client-dir"))
	if clientDir == "" {
		clientDir = "./client"
	}

	handlers := transportws.RouteHandlers{
		WebSocket: transportws.NewWebSocketHandler(gameEngine),
		Health:    monitoring.HealthHandler(gameEngine),
		Metrics:   monitoring.MetricsHandler(gameEngine),
		Dashboard: monitoring.DashboardHandler(clientDir),
		Static:    http.FileServer(http.Dir(clientDir + "/")),
	}

	if err := transportws.SetupServer(listener, handlers); err != nil {
		return fmt.Errorf("setup server: %w", err)
	}

	return nil
}
