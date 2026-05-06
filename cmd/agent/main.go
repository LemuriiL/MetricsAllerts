package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/agent"
	"github.com/LemuriiL/MetricsAllerts/internal/cli"
)

const (
	defaultAddr           = "localhost:8080"
	defaultReportInterval = 10
	defaultPollInterval   = 2
	defaultKey            = ""
	defaultRateLimit      = 1
	defaultCryptoKey      = ""
)

func main() {
	printBuildInfo()

	aFlag := &cli.StringFlag{Val: defaultAddr}
	rFlag := &cli.IntFlag{Val: defaultReportInterval}
	pFlag := &cli.IntFlag{Val: defaultPollInterval}
	kFlag := &cli.StringFlag{Val: defaultKey}
	lFlag := &cli.IntFlag{Val: defaultRateLimit}
	ckFlag := &cli.StringFlag{Val: defaultCryptoKey}

	flag.Var(aFlag, "a", "Server address (host:port)")
	flag.Var(rFlag, "r", "Report interval in seconds")
	flag.Var(pFlag, "p", "Poll interval in seconds")
	flag.Var(kFlag, "k", "Signing key")
	flag.Var(lFlag, "l", "Rate limit (max concurrent outgoing requests)")
	flag.Var(ckFlag, "crypto-key", "Path to RSA public key")
	flag.Parse()

	addr := cli.PickString("ADDRESS", aFlag.Val, aFlag.IsSet, defaultAddr)
	reportInterval := cli.PickInt("REPORT_INTERVAL", rFlag.Val, rFlag.IsSet, defaultReportInterval)
	pollInterval := cli.PickInt("POLL_INTERVAL", pFlag.Val, pFlag.IsSet, defaultPollInterval)
	rateLimit := cli.PickInt("RATE_LIMIT", lFlag.Val, lFlag.IsSet, defaultRateLimit)
	key := cli.PickString("KEY", kFlag.Val, kFlag.IsSet, defaultKey)
	cryptoKeyPath := cli.PickString("CRYPTO_KEY", ckFlag.Val, ckFlag.IsSet, defaultCryptoKey)

	key = cli.NormalizeKey(key)
	rateLimit = cli.NormalizePositiveInt(rateLimit, 1)

	httpAddr := addr
	if !strings.HasPrefix(httpAddr, "http://") && !strings.HasPrefix(httpAddr, "https://") {
		httpAddr = "http://" + httpAddr
	}

	a := agent.NewAgentWithKeyAndLimitAndCrypto(
		httpAddr,
		time.Duration(pollInterval)*time.Second,
		time.Duration(reportInterval)*time.Second,
		key,
		rateLimit,
		cryptoKeyPath,
	)

	log.Printf("Starting agent, poll=%ds, report=%ds, server=%s, rateLimit=%d", pollInterval, reportInterval, addr, rateLimit)
	a.Run()
}
