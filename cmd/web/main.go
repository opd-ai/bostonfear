//go:build js && wasm

// Command web is the WASM build target for the Arkham Horror Ebitengine client.
// Compile with:
//
//	GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web
//
// Serve the client/wasm/ directory over HTTP and open client/wasm/index.html.
// The server URL is read from the JavaScript global __serverURL if defined,
// falling back to the WebSocket equivalent of the current page origin.
package main

import (
	"fmt"
	"log"
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"
	ebapp "github.com/opd-ai/bostonfear/client/ebiten/app"
)

func main() {
	serverURL := resolveServerURL()
	log.Printf("web: connecting to %s", serverURL)

	game := ebapp.NewGame(serverURL)

	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Arkham Horror")

	if err := ebiten.RunGame(game); err != nil {
		log.Fatalf("RunGame: %v", err)
	}
}

// resolveServerURL reads the server URL from JavaScript or derives it from
// the page location so the WASM binary works without a hardcoded address.
func resolveServerURL() string {
	// Try the optional page-level override first.
	global := js.Global()
	if v := global.Get("__serverURL"); v.Type() == js.TypeString {
		return v.String()
	}

	// Derive ws:// or wss:// from window.location.
	loc := global.Get("window").Get("location")
	proto := "ws"
	if loc.Get("protocol").String() == "https:" {
		proto = "wss"
	}
	host := loc.Get("host").String()
	return fmt.Sprintf("%s://%s/ws", proto, host)
}
