package main

import (
	"database/sql"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"

	"github.com/LemuriiL/MetricsAllerts/internal/cli"
	"github.com/LemuriiL/MetricsAllerts/internal/server"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

const (
	defaultAddr          = "localhost:8080"
	defaultStoreInterval = 300
	defaultRestore       = true
	defaultDSN           = ""
	defaultFilePath      = ""
	defaultKey           = ""
)

type serverConfig struct {
	Addr          string
	StoreInterval int
	FilePath      string
	Restore       bool
	DSN           string
	Key           string
}

func main() {
	cfg := loadConfig()

	st, db, closeFn, err := initStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if closeFn != nil {
		defer closeFn()
	}

	srv := server.New(st, db, cfg.Key)

	logrus.Infof("Starting server on %s", cfg.Addr)
	if err := srv.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}

func loadConfig() serverConfig {
	aFlag := &cli.StringFlag{Val: defaultAddr}
	iFlag := &cli.IntFlag{Val: defaultStoreInterval}
	fFlag := &cli.StringFlag{Val: defaultFilePath}
	rFlag := &cli.BoolFlag{Val: defaultRestore}
	dFlag := &cli.StringFlag{Val: defaultDSN}
	kFlag := &cli.StringFlag{Val: defaultKey}

	flag.Var(aFlag, "a", "HTTP server address")
	flag.Var(iFlag, "i", "Store interval in seconds")
	flag.Var(fFlag, "f", "File storage path")
	flag.Var(rFlag, "r", "Restore from file on start")
	flag.Var(dFlag, "d", "Database DSN")
	flag.Var(kFlag, "k", "Signing key")
	flag.Parse()

	addr := cli.PickString("ADDRESS", aFlag.Val, aFlag.IsSet, defaultAddr)
	storeInterval := cli.PickInt("STORE_INTERVAL", iFlag.Val, iFlag.IsSet, defaultStoreInterval)
	filePath := cli.PickString("FILE_STORAGE_PATH", fFlag.Val, fFlag.IsSet, defaultFilePath)
	restore := cli.PickBool("RESTORE", rFlag.Val, rFlag.IsSet, defaultRestore)
	dsn := cli.PickString("DATABASE_DSN", dFlag.Val, dFlag.IsSet, defaultDSN)
	key := cli.PickString("KEY", kFlag.Val, kFlag.IsSet, defaultKey)

	key = cli.NormalizeKey(key)

	return serverConfig{
		Addr:          addr,
		StoreInterval: storeInterval,
		FilePath:      filePath,
		Restore:       restore,
		DSN:           dsn,
		Key:           key,
	}
}

func initStorage(cfg serverConfig) (storage.Storage, *sql.DB, func(), error) {
	if strings.TrimSpace(cfg.DSN) != "" {
		db, err := openDB(cfg.DSN)
		if err != nil {
			return nil, nil, nil, err
		}
		if err := applyMigrations(db); err != nil {
			_ = db.Close()
			return nil, nil, nil, err
		}
		st := storage.NewPostgresStorage(db)
		return st, db, func() { _ = db.Close() }, nil
	}

	if strings.TrimSpace(cfg.FilePath) != "" {
		fs := storage.NewFileStorage(cfg.FilePath, cfg.StoreInterval == 0)

		if cfg.Restore {
			if err := fs.Restore(); err != nil {
				return nil, nil, nil, err
			}
		}

		stop := startPeriodicSave(fs, cfg.StoreInterval)
		return fs, nil, stop, nil
	}

	return storage.NewMemStorage(), nil, nil, nil
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
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

func startPeriodicSave(fs *storage.FileStorage, interval int) func() {
	if interval <= 0 {
		return nil
	}

	ticker := timeNewTickerSeconds(interval)
	stopCh := make(chan struct{})

	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-stopCh:
				return
			case <-ticker.C:
				_ = fs.Save()
			}
		}
	}()

	return func() { close(stopCh) }
}

func timeNewTickerSeconds(sec int) *time.Ticker {
	return time.NewTicker(time.Duration(sec) * time.Second)
}
