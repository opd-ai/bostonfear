package main

import (
	"log"
	"os"

	rootcmd "github.com/opd-ai/bostonfear/cmd"
)

// Main server setup via Cobra command wrapper.
func main() {
	if err := rootcmd.ExecuteWithDefaultSubcommand("server", os.Args[1:]); err != nil {
		log.Fatalf("Server startup error: %v", err)
	}
}
