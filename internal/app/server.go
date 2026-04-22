package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/LemuriiL/MetricsAllerts/internal/config"
	"github.com/LemuriiL/MetricsAllerts/internal/server"
	"github.com/LemuriiL/MetricsAllerts/internal/storage"
)

type Closer func()

func RunServer(cfg config.ServerConfig) (Closer, error) {
	var (
		st storage.Storage
		db *sql.DB
	)

	stop := func() {}

	dsn := strings.TrimSpace(cfg.DSN)
	filePath := strings.TrimSpace(cfg.FilePath)

	if dsn != "" {
		pg, closeDB, err := storage.NewPostgresStorageFromDSN(dsn)
		if err != nil {
			return nil, err
		}

		st = pg
		db = pg.DB()
		stop = closeDB
	} else if filePath != "" {
		fs := storage.NewFileStorage(filePath, cfg.StoreInterval == 0)

		if cfg.Restore {
			if err := fs.Restore(context.Background()); err != nil {
				return nil, fmt.Errorf("restore file storage: %w", err)
			}
		}

		if cfg.StoreInterval > 0 {
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				ticker := time.NewTicker(time.Duration(cfg.StoreInterval) * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := fs.Save(context.Background()); err != nil {
							slog.Error("save file storage", "error", err)
						}
					}
				}
			}()

			prevStop := stop
			stop = func() {
				cancel()
				prevStop()
			}
		}

		st = fs
	} else {
		st = storage.NewMemStorage()
	}

	srv := server.New(st, db, "")

	go func() {
		if err := srv.Run(cfg.Address); err != nil {
			slog.Error("server stopped", "error", err)
		}
	}()

	return stop, nil
}
