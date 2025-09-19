package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/wk-y/rama-swap/ramalama"
	"github.com/wk-y/rama-swap/server"
)

const defaultPort = 4917
const defaultHost = "127.0.0.1"

const EX_USAGE = 64

func main() {
	args, rest, err := parseArgs(os.Args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", os.Args[0], err)
		os.Exit(EX_USAGE)
	}

	if len(rest) > 0 {
		fmt.Fprintf(os.Stderr, "%s: unexpected positional argument %v\n", os.Args[0], rest[0])
		os.Exit(EX_USAGE)
	}

	// set default values for unspecified flags
	if args.Host == nil {
		host := "127.0.0.1"
		args.Host = &host
	}

	if args.Port == nil {
		port := 4917
		args.Port = &port
	}

	if args.IdleTimeout == nil {
		timeout := time.Duration(0)
		args.IdleTimeout = &timeout
	}

	if args.Ramalama == nil {
		if env := os.Getenv("RAMALAMA_COMMAND"); env != "" {
			args.Ramalama = strings.Split(env, " ")
			if len(args.Ramalama) == 0 {
				log.Fatalln("RAMALAMA_COMMAND environment variable should not be all whitespace")
			}
		} else {
			args.Ramalama = []string{"ramalama"}
		}
	}

	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *args.Host, *args.Port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	defer l.Close()

	ramalama := ramalama.Ramalama{
		Command: args.Ramalama,
	}
	scheduler := server.NewFcfsScheduler(ramalama, 49170, *args.IdleTimeout)
	server := server.NewServer(ramalama, scheduler)

	server.ModelNameMangler = func(s string) string {
		return strings.ReplaceAll(s, "/", "_")
	}

	server.HandleHttp(http.DefaultServeMux)

	log.Printf("Listening on http://%s:%d\n", *args.Host, *args.Port)
	err = http.Serve(l, nil)

	log.Fatalf("Failed to serve: %v", err)
}
