package ws_test

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"

	transportws "github.com/opd-ai/bostonfear/transport/ws"
)

func ExampleSetupServer() {
	orig := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(orig)

	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	_ = listener.Close()

	handlers := transportws.RouteHandlers{
		WebSocket: http.NotFoundHandler(),
		Health:    http.NotFoundHandler(),
		Metrics:   http.NotFoundHandler(),
	}

	err := transportws.SetupServer(listener, handlers)
	fmt.Println(err != nil)
	// Output: true
}
