package main

import (
	"flag"
	"log"
	"net"

	"github.com/LemuriiL/MetricsAllerts/internal/server"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

func main() {
	var addr string
	flag.StringVar(&addr, "a", ":8080", "HTTP server address")

	flag.Parse()

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		log.Fatalf("invalid address: %v", err)
	}

	if host == "localhost" || host == "127.0.0.1" {
		addr = net.JoinHostPort("", port)
	}

	store := storage.NewMemStorage()
	srv := server.New(store)

	log.Printf("Starting server on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatal(err)
	}
}
