package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/agent"
	"github.com/LemuriiL/MetricsAllerts/internal/cli"
	"github.com/LemuriiL/MetricsAllerts/internal/config"
)

const (
	defaultAddr           = "localhost:8080"
	defaultReportInterval = 10
	defaultPollInterval   = 2
	defaultKey            = ""
	defaultRateLimit      = 1
	defaultCryptoKey      = ""
	defaultConfigPath     = ""
)

type agentConfig struct {
	Addr           string
	ReportInterval int
	PollInterval   int
	Key            string
	RateLimit      int
	CryptoKey      string
}

func main() {
	printBuildInfo()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	httpAddr := cfg.Addr
	if !strings.HasPrefix(httpAddr, "http://") && !strings.HasPrefix(httpAddr, "https://") {
		httpAddr = "http://" + httpAddr
	}

	a := agent.NewAgentWithKeyAndLimitAndCrypto(
		httpAddr,
		time.Duration(cfg.PollInterval)*time.Second,
		time.Duration(cfg.ReportInterval)*time.Second,
		cfg.Key,
		cfg.RateLimit,
		cfg.CryptoKey,
	)

	log.Printf(
		"Starting agent, poll=%ds, report=%ds, server=%s, rateLimit=%d",
		cfg.PollInterval,
		cfg.ReportInterval,
		cfg.Addr,
		cfg.RateLimit,
	)

	a.Run()
}

func loadConfig() (agentConfig, error) {
	aFlag := &cli.StringFlag{Val: defaultAddr}
	rFlag := &cli.IntFlag{Val: defaultReportInterval}
	pFlag := &cli.IntFlag{Val: defaultPollInterval}
	kFlag := &cli.StringFlag{Val: defaultKey}
	lFlag := &cli.IntFlag{Val: defaultRateLimit}
	ckFlag := &cli.StringFlag{Val: defaultCryptoKey}
	cFlag := &cli.StringFlag{Val: defaultConfigPath}

	flag.Var(aFlag, "a", "Server address (host:port)")
	flag.Var(rFlag, "r", "Report interval in seconds")
	flag.Var(pFlag, "p", "Poll interval in seconds")
	flag.Var(kFlag, "k", "Signing key")
	flag.Var(lFlag, "l", "Rate limit (max concurrent outgoing requests)")
	flag.Var(ckFlag, "crypto-key", "Path to RSA public key")
	flag.Var(cFlag, "c", "Path to JSON config")
	flag.Var(cFlag, "config", "Path to JSON config")
	flag.Parse()

	configPath := pickString(defaultConfigPath, cFlag.Val, cFlag.IsSet, "CONFIG")

	fileCfg := config.AgentFileConfig{}
	if strings.TrimSpace(configPath) != "" {
		loaded, err := config.LoadAgentFileConfig(configPath)
		if err != nil {
			return agentConfig{}, err
		}
		fileCfg = loaded
	}

	fileReportInterval := defaultReportInterval
	if strings.TrimSpace(fileCfg.ReportInterval) != "" {
		seconds, err := parseDurationSeconds(fileCfg.ReportInterval)
		if err != nil {
			return agentConfig{}, err
		}
		fileReportInterval = seconds
	}

	filePollInterval := defaultPollInterval
	if strings.TrimSpace(fileCfg.PollInterval) != "" {
		seconds, err := parseDurationSeconds(fileCfg.PollInterval)
		if err != nil {
			return agentConfig{}, err
		}
		filePollInterval = seconds
	}

	fileRateLimit := defaultRateLimit
	if fileCfg.RateLimit != nil {
		fileRateLimit = *fileCfg.RateLimit
	}

	cfg := agentConfig{
		Addr:           pickString(firstNonEmpty(fileCfg.Address, defaultAddr), aFlag.Val, aFlag.IsSet, "ADDRESS"),
		ReportInterval: pickInt(fileReportInterval, rFlag.Val, rFlag.IsSet, "REPORT_INTERVAL"),
		PollInterval:   pickInt(filePollInterval, pFlag.Val, pFlag.IsSet, "POLL_INTERVAL"),
		Key:            cli.NormalizeKey(pickString(firstNonEmpty(fileCfg.Key, defaultKey), kFlag.Val, kFlag.IsSet, "KEY")),
		RateLimit:      cli.NormalizePositiveInt(pickInt(fileRateLimit, lFlag.Val, lFlag.IsSet, "RATE_LIMIT"), 1),
		CryptoKey:      pickString(firstNonEmpty(fileCfg.CryptoKey, defaultCryptoKey), ckFlag.Val, ckFlag.IsSet, "CRYPTO_KEY"),
	}

	return cfg, nil
}

func pickString(defaultValue, flagValue string, flagSet bool, envName string) string {
	if value, ok := os.LookupEnv(envName); ok {
		return value
	}

	if flagSet {
		return flagValue
	}

	return defaultValue
}

func pickInt(defaultValue, flagValue int, flagSet bool, envName string) int {
	if value, ok := os.LookupEnv(envName); ok {
		parsed, err := strconv.Atoi(value)
		if err == nil {
			return parsed
		}
	}

	if flagSet {
		return flagValue
	}

	return defaultValue
}

func parseDurationSeconds(value string) (int, error) {
	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, err
	}

	return int(d / time.Second), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}

	return ""
}
