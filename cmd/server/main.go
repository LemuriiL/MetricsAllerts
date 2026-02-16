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

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/LemuriiL/MetricsAllerts/internal/server"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

const (
	defaultAddr          = "localhost:8080"
	defaultStoreInterval = 300
	defaultRestore       = true
	defaultDSN           = ""
	defaultFilePath      = ""
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

func applyMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
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

	var (
		st  storage.Storage
		db  *sql.DB
		err error
	)

	if strings.TrimSpace(dsn) != "" {
		db, err = sql.Open("pgx", dsn)
		if err != nil {
			log.Fatal(err)
		}
		if err := db.Ping(); err != nil {
			log.Fatal(err)
		}
		if err := applyMigrations(db); err != nil {
			log.Fatal(err)
		}
		defer db.Close()
		st = storage.NewPostgresStorage(db)
	} else if strings.TrimSpace(filePath) != "" {
		fs := storage.NewFileStorage(filePath, storeInterval == 0)

		if restore {
			if err := fs.Restore(); err != nil {
				log.Fatal(err)
			}
		}

		if storeInterval > 0 {
			ticker := time.NewTicker(time.Duration(storeInterval) * time.Second)
			go func() {
				for range ticker.C {
					fs.Save()
				}
			}()
		}

		st = fs
	} else {
		st = storage.NewMemStorage()
	}

	srv := server.New(st, db)

	logrus.Infof("Starting server on %s", addr)
	if err := srv.Run(addr); err != nil {
		log.Fatal(err)
	}
}
