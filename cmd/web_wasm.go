//go:build js && wasm

package cmd

import (
	"fmt"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	ebapp "github.com/opd-ai/bostonfear/client/ebiten/app"
	"github.com/spf13/cobra"
)

// NewWebCommand wraps the WASM Ebitengine startup logic.
func NewWebCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Run the WASM Ebitengine client",
		RunE: func(_ *cobra.Command, _ []string) error {
			return runWeb()
		},
	}

	return cmd
}

func runWeb() error {
	serverURL := resolveWebServerURL()

	game := ebapp.NewGame(serverURL)
	defer game.Close()

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Arkham Horror")

	if err := ebiten.RunGame(game); err != nil {
		return err
	}

	return nil
}

func resolveWebServerURL() string {
	global := js.Global()
	if v := global.Get("__serverURL"); v.Type() == js.TypeString {
		return v.String()
	}

	loc := global.Get("window").Get("location")
	proto := "ws"
	if loc.Get("protocol").String() == "https:" {
		proto = "wss"
	}
	host := loc.Get("host").String()
	return fmt.Sprintf("%s://%s/ws", proto, host)
}
