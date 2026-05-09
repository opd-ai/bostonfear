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
	"log"
	"os"

	rootcmd "github.com/opd-ai/bostonfear/cmd"
)

func main() {
	if err := rootcmd.ExecuteWithDefaultSubcommand("desktop", os.Args[1:]); err != nil {
		log.Fatalf("RunGame: %v", err)
	}
}
