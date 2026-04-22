package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func runPPROF() {
	go func() {
		if err := http.ListenAndServe("localhost:6060", nil); err != nil {
			log.Printf("pprof server error: %v", err)
		}
	}()
}
