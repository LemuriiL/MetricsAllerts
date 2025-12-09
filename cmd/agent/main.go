package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/agent"
)

func main() {
	var (
		serverAddr     string
		reportInterval int
		pollInterval   int
	)

	flag.StringVar(&serverAddr, "a", "localhost:8080", "Server address (host:port)")
	flag.IntVar(&reportInterval, "r", 10, "Report interval in seconds")
	flag.IntVar(&pollInterval, "p", 2, "Poll interval in seconds")

	flag.Parse()

	httpAddr := serverAddr
	if !strings.HasPrefix(httpAddr, "http://") && !strings.HasPrefix(httpAddr, "https://") {
		httpAddr = "http://" + httpAddr
	}

	a := agent.NewAgent(
		httpAddr,
		time.Duration(pollInterval)*time.Second,
		time.Duration(reportInterval)*time.Second,
	)

	log.Printf("Starting agent, poll=%ds, report=%ds, server=%s", pollInterval, reportInterval, serverAddr)
	a.Run()
}
