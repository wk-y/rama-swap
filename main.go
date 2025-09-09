package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/wk-y/rama-swap/ramalama"
	"github.com/wk-y/rama-swap/server"
)

func main() {
	l, err := net.Listen("tcp", "127.0.0.1:4917")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer l.Close()

	ramalamaCommand := []string{"ramalama"}
	if env := os.Getenv("RAMALAMA_COMMAND"); env != "" {
		ramalamaCommand = strings.Split(env, " ")
		if len(ramalamaCommand) == 0 {
			log.Fatalln("RAMALAMA_COMMAND environment variable should not be all whitespace")
		}
	}

	server := server.NewServer(ramalama.Ramalama{
		Command: ramalamaCommand,
	})

	server.ModelNameMangler = func(s string) string {
		return strings.ReplaceAll(s, "/", "_")
	}

	server.HandleHttp(http.DefaultServeMux)
	err = http.Serve(l, nil)
	log.Fatalf("Failed to serve: %v", err)
}
