package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/LemuriiL/MetricsAllerts/internal/audit"
	"github.com/LemuriiL/MetricsAllerts/internal/cli"
	"github.com/LemuriiL/MetricsAllerts/internal/config"
	"github.com/LemuriiL/MetricsAllerts/internal/grpcserver"
	pb "github.com/LemuriiL/MetricsAllerts/internal/proto"
	"github.com/LemuriiL/MetricsAllerts/internal/server"
	"github.com/LemuriiL/MetricsAllerts/internal/service"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
	"google.golang.org/grpc"
)

const (
	defaultAddr          = "localhost:8080"
	defaultGRPCAddr      = ""
	defaultStoreInterval = 300
	defaultRestore       = true
	defaultDSN           = ""
	defaultFilePath      = ""
	defaultKey           = ""
	defaultAuditFile     = ""
	defaultAuditURL      = ""
	defaultCryptoKey     = ""
	defaultTrustedSubnet = ""
	defaultConfigPath    = ""
)

type serverConfig struct {
	Addr          string
	GRPCAddr      string
	StoreInterval int
	FilePath      string
	Restore       bool
	DSN           string
	Key           string
	AuditFile     string
	AuditURL      string
	CryptoKey     string
	TrustedSubnet string
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

	httpSrv := server.New(st, db, cfg.Key, cfg.CryptoKey, cfg.TrustedSubnet)

	auditor := initAuditor(cfg)
	if auditor != nil {
		httpSrv.SetAuditor(auditor)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer stop()

	httpErrCh := make(chan error, 1)
	grpcErrCh := make(chan error, 1)

	go func() {
		slog.Info("starting http server", "addr", cfg.Addr)
		httpErrCh <- httpSrv.Run(cfg.Addr)
	}()

	var grpcSrv *grpc.Server
	var grpcListener net.Listener

	if strings.TrimSpace(cfg.GRPCAddr) != "" {
		grpcListener, err = net.Listen("tcp", cfg.GRPCAddr)
		if err != nil {
			log.Fatal(err)
		}

		grpcSrv = grpc.NewServer(
			grpc.UnaryInterceptor(grpcserver.TrustedSubnetInterceptor(cfg.TrustedSubnet)),
		)

		pb.RegisterMetricsServer(
			grpcSrv,
			grpcserver.New(service.NewMetricsService(st)),
		)

		go func() {
			slog.Info("starting grpc server", "addr", cfg.GRPCAddr)

			err := grpcSrv.Serve(grpcListener)
			if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				grpcErrCh <- err
				return
			}

			grpcErrCh <- nil
		}()
	}

	select {
	case err := <-httpErrCh:
		if err != nil {
			log.Fatal(err)
		}
	case err := <-grpcErrCh:
		if err != nil {
			log.Fatal(err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if grpcSrv != nil {
			done := make(chan struct{})

			go func() {
				grpcSrv.GracefulStop()
				close(done)
			}()

			select {
			case <-done:
			case <-shutdownCtx.Done():
				grpcSrv.Stop()
			}
		}

		if err := httpSrv.Shutdown(shutdownCtx); err != nil {
			log.Fatal(err)
		}

		if closeFn != nil {
			closeFn()
		}

		httpErr := <-httpErrCh
		if httpErr != nil {
			log.Fatal(httpErr)
		}

		if grpcSrv != nil {
			grpcErr := <-grpcErrCh
			if grpcErr != nil {
				log.Fatal(grpcErr)
			}
		}

		slog.Info("server stopped gracefully")
	}
}

func loadConfig() (serverConfig, error) {
	aFlag := &cli.StringFlag{Val: defaultAddr}
	gFlag := &cli.StringFlag{Val: defaultGRPCAddr}
	iFlag := &cli.IntFlag{Val: defaultStoreInterval}
	fFlag := &cli.StringFlag{Val: defaultFilePath}
	rFlag := &cli.BoolFlag{Val: defaultRestore}
	dFlag := &cli.StringFlag{Val: defaultDSN}
	kFlag := &cli.StringFlag{Val: defaultKey}
	afFlag := &cli.StringFlag{Val: defaultAuditFile}
	auFlag := &cli.StringFlag{Val: defaultAuditURL}
	ckFlag := &cli.StringFlag{Val: defaultCryptoKey}
	tFlag := &cli.StringFlag{Val: defaultTrustedSubnet}
	cFlag := &cli.StringFlag{Val: defaultConfigPath}

	flag.Var(aFlag, "a", "HTTP server address")
	flag.Var(gFlag, "grpc-address", "gRPC server address")
	flag.Var(iFlag, "i", "Store interval in seconds")
	flag.Var(fFlag, "f", "File storage path")
	flag.Var(rFlag, "r", "Restore from file on start")
	flag.Var(dFlag, "d", "Database DSN")
	flag.Var(kFlag, "k", "Signing key")
	flag.Var(afFlag, "audit-file", "Audit log file path")
	flag.Var(auFlag, "audit-url", "Audit receiver URL")
	flag.Var(ckFlag, "crypto-key", "Path to RSA private key")
	flag.Var(tFlag, "t", "Trusted subnet in CIDR notation")
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
		GRPCAddr:      pickString(firstNonEmpty(fileCfg.GRPCAddress, defaultGRPCAddr), gFlag.Val, gFlag.IsSet, "GRPC_ADDRESS"),
		StoreInterval: pickInt(fileStoreInterval, iFlag.Val, iFlag.IsSet, "STORE_INTERVAL"),
		FilePath:      pickStringFromEnvs(fileStorePath, fFlag.Val, fFlag.IsSet, "STORE_FILE", "FILE_STORAGE_PATH"),
		Restore:       pickBool(fileRestore, rFlag.Val, rFlag.IsSet, "RESTORE"),
		DSN:           pickString(firstNonEmpty(fileCfg.DatabaseDSN, defaultDSN), dFlag.Val, dFlag.IsSet, "DATABASE_DSN"),
		Key:           cli.NormalizeKey(pickString(firstNonEmpty(fileCfg.Key, defaultKey), kFlag.Val, kFlag.IsSet, "KEY")),
		AuditFile:     pickString(firstNonEmpty(fileCfg.AuditFile, defaultAuditFile), afFlag.Val, afFlag.IsSet, "AUDIT_FILE"),
		AuditURL:      pickString(firstNonEmpty(fileCfg.AuditURL, defaultAuditURL), auFlag.Val, auFlag.IsSet, "AUDIT_URL"),
		CryptoKey:     pickString(firstNonEmpty(fileCfg.CryptoKey, defaultCryptoKey), ckFlag.Val, ckFlag.IsSet, "CRYPTO_KEY"),
		TrustedSubnet: pickString(firstNonEmpty(fileCfg.TrustedSubnet, defaultTrustedSubnet), tFlag.Val, tFlag.IsSet, "TRUSTED_SUBNET"),
	}

	if strings.TrimSpace(cfg.TrustedSubnet) != "" {
		if _, _, err := net.ParseCIDR(cfg.TrustedSubnet); err != nil {
			return serverConfig{}, err
		}
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

		stopCh := make(chan struct{})
		var tickerStopped bool

		if cfg.StoreInterval > 0 {
			ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)

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
		} else {
			tickerStopped = true
		}

		closeFn := func() {
			if !tickerStopped {
				close(stopCh)
				tickerStopped = true
			}

			if err := fileStorage.Save(context.Background()); err != nil {
				slog.Error("final save metrics failed", "error", err)
			}
		}

		return fileStorage, nil, closeFn, nil
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
