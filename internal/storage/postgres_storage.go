package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) SetGauge(ctx context.Context, name string, value float64) error {
	return s.execRetry(ctx, func(ctx context.Context) error {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO metrics (id, type, value, delta)
			VALUES ($1, $2, $3, NULL)
			ON CONFLICT (id)
			DO UPDATE SET type = EXCLUDED.type, value = EXCLUDED.value, delta = NULL`,
			name,
			models.Gauge,
			value,
		)

		return err
	})
}

func (s *PostgresStorage) GetGauge(ctx context.Context, name string) (float64, bool, error) {
	var out float64
	var ok bool

	err := s.execRetry(ctx, func(ctx context.Context) error {
		var value sql.NullFloat64
		err := s.db.QueryRowContext(ctx, `SELECT value FROM metrics WHERE id=$1 AND type=$2`, name, models.Gauge).Scan(&value)
		if errors.Is(err, sql.ErrNoRows) {
			ok = false
			return nil
		}
		if err != nil {
			return err
		}
		if !value.Valid {
			ok = false
			return nil
		}

		out = value.Float64
		ok = true
		return nil
	})

	return out, ok, err
}

func (s *PostgresStorage) SetCounter(ctx context.Context, name string, value int64) error {
	return s.execRetry(ctx, func(ctx context.Context) error {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO metrics (id, type, delta, value)
			VALUES ($1, $2, $3, NULL)
			ON CONFLICT (id)
			DO UPDATE SET type = EXCLUDED.type, delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta, value = NULL`,
			name,
			models.Counter,
			value,
		)

		return err
	})
}

func (s *PostgresStorage) GetCounter(ctx context.Context, name string) (int64, bool, error) {
	var out int64
	var ok bool

	err := s.execRetry(ctx, func(ctx context.Context) error {
		var delta sql.NullInt64
		err := s.db.QueryRowContext(ctx, `SELECT delta FROM metrics WHERE id=$1 AND type=$2`, name, models.Counter).Scan(&delta)
		if errors.Is(err, sql.ErrNoRows) {
			ok = false
			return nil
		}
		if err != nil {
			return err
		}
		if !delta.Valid {
			ok = false
			return nil
		}

		out = delta.Int64
		ok = true
		return nil
	})

	return out, ok, err
}

func (s *PostgresStorage) GetAllGauges(ctx context.Context) (map[string]float64, error) {
	res := map[string]float64{}

	err := s.execRetry(ctx, func(ctx context.Context) error {
		rows, err := s.db.QueryContext(ctx, `SELECT id, value FROM metrics WHERE type=$1`, models.Gauge)
		if err != nil {
			return err
		}
		defer rows.Close()

		tmp := map[string]float64{}

		for rows.Next() {
			var id string
			var value sql.NullFloat64

			if err := rows.Scan(&id, &value); err == nil && value.Valid {
				tmp[id] = value.Float64
			}
		}

		if err := rows.Err(); err != nil {
			return err
		}

		res = tmp
		return nil
	})

	return res, err
}

func (s *PostgresStorage) GetAllCounters(ctx context.Context) (map[string]int64, error) {
	res := map[string]int64{}

	err := s.execRetry(ctx, func(ctx context.Context) error {
		rows, err := s.db.QueryContext(ctx, `SELECT id, delta FROM metrics WHERE type=$1`, models.Counter)
		if err != nil {
			return err
		}
		defer rows.Close()

		tmp := map[string]int64{}

		for rows.Next() {
			var id string
			var delta sql.NullInt64

			if err := rows.Scan(&id, &delta); err == nil && delta.Valid {
				tmp[id] = delta.Int64
			}
		}

		if err := rows.Err(); err != nil {
			return err
		}

		res = tmp
		return nil
	})

	return res, err
}

func (s *PostgresStorage) UpdateBatch(ctx context.Context, metrics []models.Metrics) error {
	return s.execRetry(ctx, func(ctx context.Context) error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		stmtGauge, err := tx.PrepareContext(ctx, `INSERT INTO metrics (id, type, value, delta)
			VALUES ($1, $2, $3, NULL)
			ON CONFLICT (id)
			DO UPDATE SET type = EXCLUDED.type, value = EXCLUDED.value, delta = NULL`)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		defer stmtGauge.Close()

		stmtCounter, err := tx.PrepareContext(ctx, `INSERT INTO metrics (id, type, delta, value)
			VALUES ($1, $2, $3, NULL)
			ON CONFLICT (id)
			DO UPDATE SET type = EXCLUDED.type, delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta, value = NULL`)
		if err != nil {
			_ = tx.Rollback()
			return err
		}
		defer stmtCounter.Close()

		for i := range metrics {
			metric := metrics[i]

			switch metric.MType {
			case models.Gauge:
				if metric.Value == nil {
					_ = tx.Rollback()
					return sql.ErrNoRows
				}

				if _, err := stmtGauge.ExecContext(ctx, metric.ID, models.Gauge, *metric.Value); err != nil {
					_ = tx.Rollback()
					return err
				}
			case models.Counter:
				if metric.Delta == nil {
					_ = tx.Rollback()
					return sql.ErrNoRows
				}

				if _, err := stmtCounter.ExecContext(ctx, metric.ID, models.Counter, *metric.Delta); err != nil {
					_ = tx.Rollback()
					return err
				}
			default:
				_ = tx.Rollback()
				return sql.ErrNoRows
			}
		}

		return tx.Commit()
	})
}

func (s *PostgresStorage) execRetry(parent context.Context, fn func(ctx context.Context) error) error {
	waits := []time.Duration{time.Second, 3 * time.Second, 5 * time.Second}

	err := s.execWithTimeout(parent, fn)
	if err == nil {
		return nil
	}

	for i := 0; i < len(waits); i++ {
		if !isRetryableDBError(err) {
			return err
		}

		timer := time.NewTimer(waits[i])

		select {
		case <-parent.Done():
			timer.Stop()
			return parent.Err()
		case <-timer.C:
		}

		err = s.execWithTimeout(parent, fn)
		if err == nil {
			return nil
		}
	}

	return err
}

func (s *PostgresStorage) execWithTimeout(parent context.Context, fn func(ctx context.Context) error) error {
	ctx, cancel := context.WithTimeout(parent, 3*time.Second)
	defer cancel()

	return fn(ctx)
}

func isRetryableDBError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, sql.ErrConnDone) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgerrcode.IsConnectionException(pgErr.Code)
	}

	msg := err.Error()
	return len(msg) >= 2 && msg[0] == '0' && msg[1] == '8'
}
