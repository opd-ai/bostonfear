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
	"log"
	"os"

	rootcmd "github.com/opd-ai/bostonfear/cmd"
)

func main() {
	if err := rootcmd.ExecuteWithDefaultSubcommand("web", os.Args[1:]); err != nil {
		log.Fatalf("RunGame: %v", err)
	}
}
