package main

import (
	"log"

	"github.com/sirupsen/logrus"

	"github.com/LemuriiL/MetricsAllerts/internal/app"
	"github.com/LemuriiL/MetricsAllerts/internal/config"
)

func main() {
	cfg := config.LoadServer()

	closer, err := app.RunServer(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer closer()

	logrus.Infof("Starting server on %s", cfg.Address)
	select {}
}
