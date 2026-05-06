package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

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
	defaultCryptoKey     = ""
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
	CryptoKey     string
}

func main() {
	runPPROF()
	printBuildInfo()

	cfg := loadConfig()

	st, db, closeFn, err := initStorage(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}

	if closeFn != nil {
		defer closeFn()
	}

	srv := server.New(st, db, cfg.Key, cfg.CryptoKey)

	auditor := initAuditor(cfg)
	if auditor != nil {
		srv.SetAuditor(auditor)
	}

	slog.Info("starting server", "addr", cfg.Addr)

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
	ckFlag := &cli.StringFlag{Val: defaultCryptoKey}

	flag.Var(aFlag, "a", "HTTP server address")
	flag.Var(iFlag, "i", "Store interval in seconds")
	flag.Var(fFlag, "f", "File storage path")
	flag.Var(rFlag, "r", "Restore from file on start")
	flag.Var(dFlag, "d", "Database DSN")
	flag.Var(kFlag, "k", "Signing key")
	flag.Var(afFlag, "audit-file", "Audit log file path")
	flag.Var(auFlag, "audit-url", "Audit receiver URL")
	flag.Var(ckFlag, "crypto-key", "Path to RSA private key")
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
		CryptoKey:     cli.PickString("CRYPTO_KEY", ckFlag.Val, ckFlag.IsSet, defaultCryptoKey),
	}
}

func initStorage(ctx context.Context, cfg serverConfig) (storage.Storage, *sql.DB, func(), error) {
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
		fileStorage := storage.NewFileStorage(cfg.FilePath, cfg.StoreInterval == 0)

		if cfg.Restore {
			if err := fileStorage.Restore(ctx); err != nil {
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
						if err := fileStorage.Save(context.Background()); err != nil {
							slog.Error("save metrics failed", "error", err)
						}
					}
				}
			}()

			stop = func() { close(stopCh) }
		}

		return fileStorage, nil, stop, nil
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

	broadcaster := audit.NewBroadcaster()

	if strings.TrimSpace(cfg.AuditFile) != "" {
		broadcaster.Subscribe(audit.NewFileObserver(cfg.AuditFile))
	}

	if strings.TrimSpace(cfg.AuditURL) != "" {
		broadcaster.Subscribe(audit.NewHTTPObserver(cfg.AuditURL))
	}

	return broadcaster
}
