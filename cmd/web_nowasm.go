//go:build !js || !wasm

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewWebCommand returns a stub on non-WASM targets.
func NewWebCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "web",
		Short: "Run the WASM Ebitengine client",
		Long: `The web command is only available for js/wasm builds.

To run the WASM client, compile with:
  GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web

This command intentionally errors on non-WASM targets (Linux/macOS/Windows).`,
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return fmt.Errorf("web command is only available for js/wasm builds")
		},
	}
}
