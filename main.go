package main

import (
	"log"
	"net"
	"net/http"

	"github.com/wk-y/rama-swap/ramalama"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:2005")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer l.Close()

	server := Server{
		ramalama: &ramalama.Ramalama{
			Command: []string{"uvx", "ramalama"},
		},
	}

	server.HandleHttp(http.DefaultServeMux)
	err = http.Serve(l, nil)
	log.Fatalf("Failed to serve: %v", err)
}
