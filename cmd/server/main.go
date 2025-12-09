package main

import (
	"flag"
	"log"
	"strings"

	"github.com/LemuriiL/MetricsAllerts/internal/server"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

func main() {
	var addr string
	flag.StringVar(&addr, "a", "localhost:8080", "HTTP server address")
	flag.Parse()

	if strings.Contains(addr, ":") {
		parts := strings.Split(addr, ":")
		addr = ":" + parts[len(parts)-1]
	}

	store := storage.NewMemStorage()
	srv := server.New(store)

	log.Printf("Starting server on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatal(err)
	}
}
