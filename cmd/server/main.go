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

	"github.com/LemuriiL/MetricsAllerts/internal/audit"
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
	defaultAuditFile     = ""
	defaultAuditURL      = ""
)

type serverConfig struct {
	Addr          string
	StoreInterval int
	FilePath      string
	Restore       bool
	DSN           string
	Key           string
	AuditFile     string
	AuditURL      string
}

func main() {
	runPPROF()
	cfg := loadConfig()

	st, db, closeFn, err := initStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}
	if closeFn != nil {
		defer closeFn()
	}

	srv := server.New(st, db, cfg.Key)

	auditor := initAuditor(cfg)
	if auditor != nil {
		srv.SetAuditor(auditor)
	}

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
	afFlag := &cli.StringFlag{Val: defaultAuditFile}
	auFlag := &cli.StringFlag{Val: defaultAuditURL}

	flag.Var(aFlag, "a", "HTTP server address")
	flag.Var(iFlag, "i", "Store interval in seconds")
	flag.Var(fFlag, "f", "File storage path")
	flag.Var(rFlag, "r", "Restore from file on start")
	flag.Var(dFlag, "d", "Database DSN")
	flag.Var(kFlag, "k", "Signing key")
	flag.Var(afFlag, "audit-file", "Audit log file path")
	flag.Var(auFlag, "audit-url", "Audit receiver URL")
	flag.Parse()

	return serverConfig{
		Addr:          cli.PickString("ADDRESS", aFlag.Val, aFlag.IsSet, defaultAddr),
		StoreInterval: cli.PickInt("STORE_INTERVAL", iFlag.Val, iFlag.IsSet, defaultStoreInterval),
		FilePath:      cli.PickString("FILE_STORAGE_PATH", fFlag.Val, fFlag.IsSet, defaultFilePath),
		Restore:       cli.PickBool("RESTORE", rFlag.Val, rFlag.IsSet, defaultRestore),
		DSN:           cli.PickString("DATABASE_DSN", dFlag.Val, dFlag.IsSet, defaultDSN),
		Key:           cli.NormalizeKey(cli.PickString("KEY", kFlag.Val, kFlag.IsSet, defaultKey)),
		AuditFile:     cli.PickString("AUDIT_FILE", afFlag.Val, afFlag.IsSet, defaultAuditFile),
		AuditURL:      cli.PickString("AUDIT_URL", auFlag.Val, auFlag.IsSet, defaultAuditURL),
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
		return storage.NewPostgresStorage(db), db, func() { _ = db.Close() }, nil
	}

	if strings.TrimSpace(cfg.FilePath) != "" {
		fs := storage.NewFileStorage(cfg.FilePath, cfg.StoreInterval == 0)

		if cfg.Restore {
			if err := fs.Restore(); err != nil {
				return nil, nil, nil, err
			}
		}

		var stop func()
		if cfg.StoreInterval > 0 {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
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

			stop = func() {
				close(stopCh)
			}
		}

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

func initAuditor(cfg serverConfig) *audit.Broadcaster {
	if strings.TrimSpace(cfg.AuditFile) == "" && strings.TrimSpace(cfg.AuditURL) == "" {
		return nil
	}

	b := audit.NewBroadcaster()

	if strings.TrimSpace(cfg.AuditFile) != "" {
		b.Subscribe(audit.NewFileObserver(cfg.AuditFile))
	}

	if strings.TrimSpace(cfg.AuditURL) != "" {
		b.Subscribe(audit.NewHTTPObserver(cfg.AuditURL))
	}

	return b
}
