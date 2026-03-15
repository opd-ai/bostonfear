// Package mobile provides the Ebitengine mobile binding entry point for the
// Arkham Horror game client, targeting Android and iOS via ebitenmobile.
//
// Build for Android:
//
//	ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile
//
// Build for iOS:
//
//	ebitenmobile bind -target ios -o dist/BostonFear.xcframework ./cmd/mobile
//
// The server URL defaults to ws://10.0.2.2:8080/ws (Android emulator host)
// and can be overridden by calling SetServerURL before the game starts.
package mobile

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/mobile"
	ebapp "github.com/opd-ai/bostonfear/client/ebiten/app"
)

// defaultServerURL is the WebSocket server URL used when none is set explicitly.
// On Android emulators 10.0.2.2 routes to the host machine's localhost.
const defaultServerURL = "ws://10.0.2.2:8080/ws"

var serverURL = defaultServerURL

// SetServerURL overrides the default server URL before starting the game.
// Call this from native Android/iOS code before the first frame is drawn.
func SetServerURL(url string) {
	serverURL = url
}

// init registers the game with Ebitengine's mobile runner.
// ebitenmobile calls init() on the binding package after loading.
func init() {
	game := ebapp.NewGame(serverURL)
	mobile.SetGame(game)
}

// Dummy is required by ebitenmobile's binding generator to export a symbol from
// this package. The binding tool wraps exported identifiers; without at least one
// the package is elided from the generated AAR/xcframework.
func Dummy() {}

// Ensure the ebiten package is referenced so its init() runs.
var _ = ebiten.ActualFPS
