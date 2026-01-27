package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/LemuriiL/MetricsAllerts/internal/model"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) SetGauge(name string, value float64) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _ = s.db.ExecContext(
		ctx,
		`INSERT INTO metrics (id, type, value, delta)
		 VALUES ($1, $2, $3, NULL)
		 ON CONFLICT (id) DO UPDATE
		 SET type = EXCLUDED.type, value = EXCLUDED.value, delta = NULL`,
		name, models.Gauge, value,
	)
}

func (s *PostgresStorage) GetGauge(name string) (float64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var v sql.NullFloat64
	err := s.db.QueryRowContext(ctx, `SELECT value FROM metrics WHERE id=$1 AND type=$2`, name, models.Gauge).Scan(&v)
	if err != nil || !v.Valid {
		return 0, false
	}
	return v.Float64, true
}

func (s *PostgresStorage) SetCounter(name string, value int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, _ = s.db.ExecContext(
		ctx,
		`INSERT INTO metrics (id, type, delta, value)
		 VALUES ($1, $2, $3, NULL)
		 ON CONFLICT (id) DO UPDATE
		 SET type = EXCLUDED.type,
		     delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta,
		     value = NULL`,
		name, models.Counter, value,
	)
}

func (s *PostgresStorage) GetCounter(name string) (int64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var v sql.NullInt64
	err := s.db.QueryRowContext(ctx, `SELECT delta FROM metrics WHERE id=$1 AND type=$2`, name, models.Counter).Scan(&v)
	if err != nil || !v.Valid {
		return 0, false
	}
	return v.Int64, true
}

func (s *PostgresStorage) GetAllGauges() map[string]float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `SELECT id, value FROM metrics WHERE type=$1`, models.Gauge)
	if err != nil {
		return map[string]float64{}
	}
	defer rows.Close()

	res := map[string]float64{}
	for rows.Next() {
		var id string
		var v sql.NullFloat64
		if err := rows.Scan(&id, &v); err == nil && v.Valid {
			res[id] = v.Float64
		}
	}
	return res
}

func (s *PostgresStorage) GetAllCounters() map[string]int64 {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rows, err := s.db.QueryContext(ctx, `SELECT id, delta FROM metrics WHERE type=$1`, models.Counter)
	if err != nil {
		return map[string]int64{}
	}
	defer rows.Close()

	res := map[string]int64{}
	for rows.Next() {
		var id string
		var v sql.NullInt64
		if err := rows.Scan(&id, &v); err == nil && v.Valid {
			res[id] = v.Int64
		}
	}
	return res
}
