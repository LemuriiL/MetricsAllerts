package config

import (
	"flag"
	"os"
	"strconv"
	"strings"
)

type ServerConfig struct {
	Address       string
	StoreInterval int
	FilePath      string
	Restore       bool
	DSN           string
}

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

type boolFlag struct {
	val   bool
	isSet bool
}

func (b *boolFlag) String() string {
	if b.val {
		return "true"
	}
	return "false"
}
func (b *boolFlag) Set(v string) error {
	x, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	b.val = x
	b.isSet = true
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

func envBool(key string) (bool, bool) {
	v, ok := envString(key)
	if !ok {
		return false, false
	}
	x, err := strconv.ParseBool(v)
	if err != nil {
		return false, false
	}
	return x, true
}

func LoadServer() ServerConfig {
	const (
		defaultAddr          = "localhost:8080"
		defaultStoreInterval = 300
		defaultRestore       = true
		defaultDSN           = ""
		defaultFilePath      = ""
	)

	cfg := ServerConfig{
		Address:       defaultAddr,
		StoreInterval: defaultStoreInterval,
		FilePath:      defaultFilePath,
		Restore:       defaultRestore,
		DSN:           defaultDSN,
	}

	aFlag := &stringFlag{val: defaultAddr}
	iFlag := &intFlag{val: defaultStoreInterval}
	fFlag := &stringFlag{val: defaultFilePath}
	rFlag := &boolFlag{val: defaultRestore}
	dFlag := &stringFlag{val: defaultDSN}

	flag.Var(aFlag, "a", "HTTP server address")
	flag.Var(iFlag, "i", "Store interval in seconds")
	flag.Var(fFlag, "f", "File storage path")
	flag.Var(rFlag, "r", "Restore from file on start")
	flag.Var(dFlag, "d", "Database DSN")

	flag.Parse()

	if v, ok := envString("ADDRESS"); ok {
		cfg.Address = v
	} else if aFlag.isSet {
		cfg.Address = aFlag.val
	}

	if v, ok := envInt("STORE_INTERVAL"); ok {
		cfg.StoreInterval = v
	} else if iFlag.isSet {
		cfg.StoreInterval = iFlag.val
	}

	if v, ok := envString("FILE_STORAGE_PATH"); ok {
		cfg.FilePath = v
	} else if fFlag.isSet {
		cfg.FilePath = fFlag.val
	}

	if v, ok := envBool("RESTORE"); ok {
		cfg.Restore = v
	} else if rFlag.isSet {
		cfg.Restore = rFlag.val
	}

	if v, ok := envString("DATABASE_DSN"); ok {
		cfg.DSN = v
	} else if dFlag.isSet {
		cfg.DSN = dFlag.val
	}

	return cfg
}
