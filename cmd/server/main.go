package main

import (
	"log"

	"github.com/LemuriiL/MetricsAllerts/internal/server"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

func main() {
	store := storage.NewMemStorage()
	srv := server.New(store)

	log.Println("Starting server on :8080")
	if err := srv.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
