package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/agent"
)

func main() {

	if os.Getenv("DISABLE_AGENT") == "true" {
		log.Println("Agent disabled by environment")
		return
	}

	var (
		serverAddr     string
		reportInterval int
		pollInterval   int
	)

	flag.StringVar(&serverAddr, "a", "http://localhost:8080", "Server address")
	flag.IntVar(&reportInterval, "r", 10, "Report interval in seconds")
	flag.IntVar(&pollInterval, "p", 2, "Poll interval in seconds")

	flag.Parse()

	agent := agent.NewAgent(
		serverAddr,
		time.Duration(pollInterval)*time.Second,
		time.Duration(reportInterval)*time.Second,
	)

	log.Printf("Starting agent, poll=%ds, report=%ds, server=%s", pollInterval, reportInterval, serverAddr)
	agent.Run()
}
