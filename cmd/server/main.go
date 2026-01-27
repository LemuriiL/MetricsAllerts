package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"

	"github.com/LemuriiL/MetricsAllerts/internal/server"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

const (
	defaultAddr          = "localhost:8080"
	defaultStoreInterval = 300
	defaultFilePath      = "metrics-db.json"
	defaultRestore       = true
	defaultDSN           = ""
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

func main() {
	addr := defaultAddr
	storeInterval := defaultStoreInterval
	filePath := defaultFilePath
	restore := defaultRestore
	dsn := defaultDSN

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
		addr = v
	} else if aFlag.isSet {
		addr = aFlag.val
	}

	if v, ok := envInt("STORE_INTERVAL"); ok {
		storeInterval = v
	} else if iFlag.isSet {
		storeInterval = iFlag.val
	}

	if v, ok := envString("FILE_STORAGE_PATH"); ok {
		filePath = v
	} else if fFlag.isSet {
		filePath = fFlag.val
	}

	if v, ok := envBool("RESTORE"); ok {
		restore = v
	} else if rFlag.isSet {
		restore = rFlag.val
	}

	if v, ok := envString("DATABASE_DSN"); ok {
		dsn = v
	} else if dFlag.isSet {
		dsn = dFlag.val
	}

	store := storage.NewFileStorage(filePath, storeInterval == 0)

	if restore {
		if err := store.Restore(); err != nil {
			log.Fatal(err)
		}
	}

	if storeInterval > 0 {
		ticker := time.NewTicker(time.Duration(storeInterval) * time.Second)
		go func() {
			for range ticker.C {
				store.Save()
			}
		}()
	}

	var db *sql.DB
	if strings.TrimSpace(dsn) != "" {
		var err error
		logrus.Infof("DSN = %s", dsn)
		db, err = sql.Open("pgx", dsn)
		if err != nil {
			log.Fatal(err)
		}
		if err := db.Ping(); err != nil {
			log.Fatal(err)
		}
		defer db.Close()
	}

	srv := server.New(store, db)

	logrus.Infof("Starting server on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatal(err)
	}
}
