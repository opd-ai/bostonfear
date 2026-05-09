package serverengine_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/opd-ai/bostonfear/serverengine"
)

func ExampleNewGameServer() {
	gs := serverengine.NewGameServer()
	fmt.Println(gs != nil)
	// Output: true
}

func ExampleGameServer_HandleConnection() {
	orig := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(orig)

	gs := serverengine.NewGameServer()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, peer := net.Pipe()
	_ = peer.Close()

	err := gs.HandleConnectionWithContext(ctx, conn, "")
	fmt.Println(err == nil)
	// Output: true
}
