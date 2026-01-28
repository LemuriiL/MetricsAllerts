package storage

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresStorageWithDB struct {
	*PostgresStorage
	db *sql.DB
}

func (p *PostgresStorageWithDB) DB() *sql.DB {
	return p.db
}

func NewPostgresStorageFromDSN(dsn string) (*PostgresStorageWithDB, func(), error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open postgres: %w", err)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping postgres: %w", err)
	}

	if err := applyMigrations(db); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("apply migrations: %w", err)
	}

	closeFn := func() {
		_ = db.Close()
	}

	return &PostgresStorageWithDB{PostgresStorage: NewPostgresStorage(db), db: db}, closeFn, nil
}

func applyMigrations(db *sql.DB) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}

	return nil
}
