package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"

	models "github.com/LemuriiL/MetricsAllerts/internal/model"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) SetGauge(name string, value float64) {
	_ = s.execRetry(func(ctx context.Context) error {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO metrics (id, type, value, delta)
			 VALUES ($1, $2, $3, NULL)
			 ON CONFLICT (id) DO UPDATE
			 SET type = EXCLUDED.type, value = EXCLUDED.value, delta = NULL`,
			name, models.Gauge, value,
		)
		return err
	})
}

func (s *PostgresStorage) GetGauge(name string) (float64, bool) {
	var out float64
	var ok bool

	_ = s.execRetry(func(ctx context.Context) error {
		var v sql.NullFloat64
		err := s.db.QueryRowContext(ctx, `SELECT value FROM metrics WHERE id=$1 AND type=$2`, name, models.Gauge).Scan(&v)
		if err != nil || !v.Valid {
			ok = false
			return err
		}
		out = v.Float64
		ok = true
		return nil
	})

	return out, ok
}

func (s *PostgresStorage) SetCounter(name string, value int64) {
	_ = s.execRetry(func(ctx context.Context) error {
		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO metrics (id, type, delta, value)
			 VALUES ($1, $2, $3, NULL)
			 ON CONFLICT (id) DO UPDATE
			 SET type = EXCLUDED.type,
			     delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta,
			     value = NULL`,
			name, models.Counter, value,
		)
		return err
	})
}

func (s *PostgresStorage) GetCounter(name string) (int64, bool) {
	var out int64
	var ok bool

	_ = s.execRetry(func(ctx context.Context) error {
		var v sql.NullInt64
		err := s.db.QueryRowContext(ctx, `SELECT delta FROM metrics WHERE id=$1 AND type=$2`, name, models.Counter).Scan(&v)
		if err != nil || !v.Valid {
			ok = false
			return err
		}
		out = v.Int64
		ok = true
		return nil
	})

	return out, ok
}

func (s *PostgresStorage) GetAllGauges() map[string]float64 {
	res := map[string]float64{}

	_ = s.execRetry(func(ctx context.Context) error {
		rows, err := s.db.QueryContext(ctx, `SELECT id, value FROM metrics WHERE type=$1`, models.Gauge)
		if err != nil {
			return err
		}
		defer rows.Close()

		tmp := map[string]float64{}
		for rows.Next() {
			var id string
			var v sql.NullFloat64
			if err := rows.Scan(&id, &v); err == nil && v.Valid {
				tmp[id] = v.Float64
			}
		}

		if err := rows.Err(); err != nil {
			return err
		}

		res = tmp
		return nil
	})

	return res
}

func (s *PostgresStorage) GetAllCounters() map[string]int64 {
	res := map[string]int64{}

	_ = s.execRetry(func(ctx context.Context) error {
		rows, err := s.db.QueryContext(ctx, `SELECT id, delta FROM metrics WHERE type=$1`, models.Counter)
		if err != nil {
			return err
		}
		defer rows.Close()

		tmp := map[string]int64{}
		for rows.Next() {
			var id string
			var v sql.NullInt64
			if err := rows.Scan(&id, &v); err == nil && v.Valid {
				tmp[id] = v.Int64
			}
		}

		if err := rows.Err(); err != nil {
			return err
		}

		res = tmp
		return nil
	})

	return res
}

func (s *PostgresStorage) UpdateBatch(ms []models.Metrics) error {
	return s.execRetry(func(ctx context.Context) error {
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		for i := range ms {
			m := ms[i]
			switch m.MType {
			case models.Gauge:
				if m.Value == nil {
					_ = tx.Rollback()
					return sql.ErrNoRows
				}
				_, err = tx.ExecContext(
					ctx,
					`INSERT INTO metrics (id, type, value, delta)
					 VALUES ($1, $2, $3, NULL)
					 ON CONFLICT (id) DO UPDATE
					 SET type = EXCLUDED.type, value = EXCLUDED.value, delta = NULL`,
					m.ID, models.Gauge, *m.Value,
				)
				if err != nil {
					_ = tx.Rollback()
					return err
				}
			case models.Counter:
				if m.Delta == nil {
					_ = tx.Rollback()
					return sql.ErrNoRows
				}
				_, err = tx.ExecContext(
					ctx,
					`INSERT INTO metrics (id, type, delta, value)
					 VALUES ($1, $2, $3, NULL)
					 ON CONFLICT (id) DO UPDATE
					 SET type = EXCLUDED.type,
					     delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta,
					     value = NULL`,
					m.ID, models.Counter, *m.Delta,
				)
				if err != nil {
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

func (s *PostgresStorage) execRetry(fn func(ctx context.Context) error) error {
	waits := []time.Duration{1 * time.Second, 3 * time.Second, 5 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := fn(ctx)
	if err == nil {
		return nil
	}

	for i := 0; i < len(waits); i++ {
		if !isRetryableDBError(err) {
			return err
		}
		time.Sleep(waits[i])

		ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
		err = fn(ctx2)
		cancel2()

		if err == nil {
			return nil
		}
	}

	return err
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
	if len(msg) >= 2 && msg[0] == '0' && msg[1] == '8' {
		return true
	}

	return false
}
