package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PingRepository struct {
	pool *pgxpool.Pool
}

func NewPingRepository(pool *pgxpool.Pool) *PingRepository {
	return &PingRepository{pool: pool}
}

func (r *PingRepository) InsertPing(ctx context.Context, projectID int, ts time.Time) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO pings(project_id, ts) VALUES($1, $2)`, projectID, ts)
	return err
}

func (r *PingRepository) GetPingsBetween(ctx context.Context, projectID int, from, to time.Time) ([]time.Time, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ts
		FROM pings
		WHERE project_id = $1
		  AND ts >= $2
		  AND ts <= $3
		ORDER BY ts ASC
	`, projectID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []time.Time
	for rows.Next() {
		var ts time.Time
		if err := rows.Scan(&ts); err != nil {
			return nil, err
		}
		out = append(out, ts)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *PingRepository) GetFirstPing(ctx context.Context, projectID int) (time.Time, bool, error) {
	var first sql.NullTime
	err := r.pool.QueryRow(ctx, `
		SELECT MIN(ts)
		FROM pings
		WHERE project_id = $1
	`, projectID).Scan(&first)
	if err != nil {
		return time.Time{}, false, err
	}
	if !first.Valid {
		return time.Time{}, false, nil
	}
	return first.Time, true, nil
}
