package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/wk-y/rama-swap/ramalama"
	"github.com/wk-y/rama-swap/server"
)

const defaultPort = 4917
const defaultHost = "127.0.0.1"

func main() {
	host := flag.String("host", defaultHost, "ip address to listen on")
	port := flag.Int("port", defaultPort, "port to listen on")
	flag.Parse()

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *host, *port))
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

	log.Printf("Listening on http://%s:%d\n", *host, *port)
	err = http.Serve(l, nil)

	log.Fatalf("Failed to serve: %v", err)
}
