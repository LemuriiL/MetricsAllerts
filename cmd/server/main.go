package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/LemuriiL/MetricsAllerts/internal/audit"
	"github.com/LemuriiL/MetricsAllerts/internal/cli"
	"github.com/LemuriiL/MetricsAllerts/internal/config"
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
	defaultConfigPath    = ""
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

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

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

func loadConfig() (serverConfig, error) {
	aFlag := &cli.StringFlag{Val: defaultAddr}
	iFlag := &cli.IntFlag{Val: defaultStoreInterval}
	fFlag := &cli.StringFlag{Val: defaultFilePath}
	rFlag := &cli.BoolFlag{Val: defaultRestore}
	dFlag := &cli.StringFlag{Val: defaultDSN}
	kFlag := &cli.StringFlag{Val: defaultKey}
	afFlag := &cli.StringFlag{Val: defaultAuditFile}
	auFlag := &cli.StringFlag{Val: defaultAuditURL}
	ckFlag := &cli.StringFlag{Val: defaultCryptoKey}
	cFlag := &cli.StringFlag{Val: defaultConfigPath}

	flag.Var(aFlag, "a", "HTTP server address")
	flag.Var(iFlag, "i", "Store interval in seconds")
	flag.Var(fFlag, "f", "File storage path")
	flag.Var(rFlag, "r", "Restore from file on start")
	flag.Var(dFlag, "d", "Database DSN")
	flag.Var(kFlag, "k", "Signing key")
	flag.Var(afFlag, "audit-file", "Audit log file path")
	flag.Var(auFlag, "audit-url", "Audit receiver URL")
	flag.Var(ckFlag, "crypto-key", "Path to RSA private key")
	flag.Var(cFlag, "c", "Path to JSON config")
	flag.Var(cFlag, "config", "Path to JSON config")
	flag.Parse()

	configPath := pickString(defaultConfigPath, cFlag.Val, cFlag.IsSet, "CONFIG")

	fileCfg := config.ServerFileConfig{}
	if strings.TrimSpace(configPath) != "" {
		loaded, err := config.LoadServerFileConfig(configPath)
		if err != nil {
			return serverConfig{}, err
		}
		fileCfg = loaded
	}

	fileStoreInterval := defaultStoreInterval
	if strings.TrimSpace(fileCfg.StoreInterval) != "" {
		seconds, err := parseDurationSeconds(fileCfg.StoreInterval)
		if err != nil {
			return serverConfig{}, err
		}
		fileStoreInterval = seconds
	}

	fileRestore := defaultRestore
	if fileCfg.Restore != nil {
		fileRestore = *fileCfg.Restore
	}

	fileStorePath := firstNonEmpty(fileCfg.StoreFile, defaultFilePath)

	cfg := serverConfig{
		Addr:          pickString(firstNonEmpty(fileCfg.Address, defaultAddr), aFlag.Val, aFlag.IsSet, "ADDRESS"),
		StoreInterval: pickInt(fileStoreInterval, iFlag.Val, iFlag.IsSet, "STORE_INTERVAL"),
		FilePath:      pickStringFromEnvs(fileStorePath, fFlag.Val, fFlag.IsSet, "STORE_FILE", "FILE_STORAGE_PATH"),
		Restore:       pickBool(fileRestore, rFlag.Val, rFlag.IsSet, "RESTORE"),
		DSN:           pickString(firstNonEmpty(fileCfg.DatabaseDSN, defaultDSN), dFlag.Val, dFlag.IsSet, "DATABASE_DSN"),
		Key:           cli.NormalizeKey(pickString(firstNonEmpty(fileCfg.Key, defaultKey), kFlag.Val, kFlag.IsSet, "KEY")),
		AuditFile:     pickString(firstNonEmpty(fileCfg.AuditFile, defaultAuditFile), afFlag.Val, afFlag.IsSet, "AUDIT_FILE"),
		AuditURL:      pickString(firstNonEmpty(fileCfg.AuditURL, defaultAuditURL), auFlag.Val, auFlag.IsSet, "AUDIT_URL"),
		CryptoKey:     pickString(firstNonEmpty(fileCfg.CryptoKey, defaultCryptoKey), ckFlag.Val, ckFlag.IsSet, "CRYPTO_KEY"),
	}

	return cfg, nil
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

func pickString(defaultValue, flagValue string, flagSet bool, envName string) string {
	return pickStringFromEnvs(defaultValue, flagValue, flagSet, envName)
}

func pickStringFromEnvs(defaultValue, flagValue string, flagSet bool, envNames ...string) string {
	for _, envName := range envNames {
		if value, ok := os.LookupEnv(envName); ok {
			return value
		}
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

func pickBool(defaultValue, flagValue bool, flagSet bool, envName string) bool {
	if value, ok := os.LookupEnv(envName); ok {
		parsed, err := strconv.ParseBool(value)
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
