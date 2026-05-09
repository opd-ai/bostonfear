package cmd

import (
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	ebapp "github.com/opd-ai/bostonfear/client/ebiten/app"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const defaultDesktopServerURL = "ws://localhost:8080/ws"

// NewDesktopCommand wraps the desktop Ebitengine startup logic.
func NewDesktopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "desktop",
		Short: "Run the desktop Ebitengine client",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runDesktop()
		},
	}

	cmd.Flags().String("server", defaultDesktopServerURL, "WebSocket server URL (e.g. ws://host:port/ws)")
	_ = viper.BindPFlag("desktop.server", cmd.Flags().Lookup("server"))

	return cmd
}

func runDesktop() error {
	serverURL := strings.TrimSpace(viper.GetString("desktop.server"))
	if serverURL == "" {
		serverURL = defaultDesktopServerURL
	}

	game := ebapp.NewGame(serverURL)
	defer game.Close()

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Arkham Horror — Ebitengine Client")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		return err
	}

	return nil
}
