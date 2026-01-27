package main

import (
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/agent"
)

const (
	defaultAddr           = "localhost:8080"
	defaultReportInterval = 10
	defaultPollInterval   = 2
)

type stringFlag struct {
	val   string
	isSet bool
}

func (s *stringFlag) String() string { return s.val }
func (s *stringFlag) Set(v string) error {
	s.val = v
	s.isSet = true
	return nil
}

type intFlag struct {
	val   int
	isSet bool
}

func (i *intFlag) String() string { return strconv.Itoa(i.val) }
func (i *intFlag) Set(v string) error {
	n, err := strconv.Atoi(v)
	if err != nil {
		return err
	}
	i.val = n
	i.isSet = true
	return nil
}

func envString(key string) (string, bool) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	return v, true
}

func envInt(key string) (int, bool) {
	v, ok := envString(key)
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, false
	}
	return n, true
}

func main() {
	addr := defaultAddr
	reportInterval := defaultReportInterval
	pollInterval := defaultPollInterval

	aFlag := &stringFlag{val: defaultAddr}
	rFlag := &intFlag{val: defaultReportInterval}
	pFlag := &intFlag{val: defaultPollInterval}

	flag.Var(aFlag, "a", "Server address (host:port)")
	flag.Var(rFlag, "r", "Report interval in seconds")
	flag.Var(pFlag, "p", "Poll interval in seconds")

	flag.Parse()

	if v, ok := envString("ADDRESS"); ok {
		addr = v
	} else if aFlag.isSet {
		addr = aFlag.val
	}

	if v, ok := envInt("REPORT_INTERVAL"); ok {
		reportInterval = v
	} else if rFlag.isSet {
		reportInterval = rFlag.val
	}

	if v, ok := envInt("POLL_INTERVAL"); ok {
		pollInterval = v
	} else if pFlag.isSet {
		pollInterval = pFlag.val
	}

	httpAddr := addr
	if !strings.HasPrefix(httpAddr, "http://") && !strings.HasPrefix(httpAddr, "https://") {
		httpAddr = "http://" + httpAddr
	}

	a := agent.NewAgent(
		httpAddr,
		time.Duration(pollInterval)*time.Second,
		time.Duration(reportInterval)*time.Second,
	)

	log.Printf("Starting agent, poll=%ds, report=%ds, server=%s", pollInterval, reportInterval, addr)
	a.Run()
}
