package main

import (
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/wk-y/rama-swap/ramalama"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:4917")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer l.Close()

	server := NewServer(ramalama.Ramalama{
		Command: []string{"uvx", "ramalama"},
	})
	server.ModelNameMangler = func(s string) string {
		return strings.ReplaceAll(s, "/", "_")
	}

	server.HandleHttp(http.DefaultServeMux)
	err = http.Serve(l, nil)
	log.Fatalf("Failed to serve: %v", err)
}
