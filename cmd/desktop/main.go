// Command desktop is the native desktop build target for the Arkham Horror
// Ebitengine client. It parses a -server flag, opens an 800×600 game window,
// and connects to the WebSocket server at the given URL.
//
// Usage:
//
//	desktop [-server ws://host:port/ws]
//
// The default server address is ws://localhost:8080/ws.
package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	ebapp "github.com/opd-ai/bostonfear/client/ebiten/app"
)

func main() {
	serverURL := flag.String("server", "ws://localhost:8080/ws",
		"WebSocket server URL (e.g. ws://host:port/ws)")
	flag.Parse()

	game := ebapp.NewGame(*serverURL)

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Arkham Horror — Ebitengine Client")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("RunGame: %v", err)
	}
}
