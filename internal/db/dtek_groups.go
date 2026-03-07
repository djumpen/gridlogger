package db

import (
	"context"
	"errors"

	"github.com/djumpen/gridlogger/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DTEKGroupRepository struct {
	pool *pgxpool.Pool
}

func NewDTEKGroupRepository(pool *pgxpool.Pool) *DTEKGroupRepository {
	return &DTEKGroupRepository{pool: pool}
}

func (r *DTEKGroupRepository) GetByProjectID(ctx context.Context, projectID int) (service.DTEKGroupConfig, bool, error) {
	const q = `
		SELECT project_id, region_id, region_name, dso_id, dso_name, street_id, street_name,
			house_id, house_name, group_key, created_at, updated_at
		FROM dtek_groups
		WHERE project_id = $1
	`

	var cfg service.DTEKGroupConfig
	err := r.pool.QueryRow(ctx, q, projectID).Scan(
		&cfg.ProjectID,
		&cfg.RegionID,
		&cfg.RegionName,
		&cfg.DSOID,
		&cfg.DSOName,
		&cfg.StreetID,
		&cfg.StreetName,
		&cfg.HouseID,
		&cfg.HouseName,
		&cfg.GroupKey,
		&cfg.CreatedAt,
		&cfg.UpdatedAt,
	)
	if err == nil {
		return cfg, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return service.DTEKGroupConfig{}, false, nil
	}
	return service.DTEKGroupConfig{}, false, err
}

func (r *DTEKGroupRepository) Upsert(ctx context.Context, cfg service.DTEKGroupConfig) (service.DTEKGroupConfig, error) {
	const q = `
		INSERT INTO dtek_groups (
			project_id, region_id, region_name, dso_id, dso_name, street_id, street_name,
			house_id, house_name, group_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (project_id) DO UPDATE
		SET region_id = EXCLUDED.region_id,
			region_name = EXCLUDED.region_name,
			dso_id = EXCLUDED.dso_id,
			dso_name = EXCLUDED.dso_name,
			street_id = EXCLUDED.street_id,
			street_name = EXCLUDED.street_name,
			house_id = EXCLUDED.house_id,
			house_name = EXCLUDED.house_name,
			group_key = EXCLUDED.group_key,
			updated_at = now()
		RETURNING project_id, region_id, region_name, dso_id, dso_name, street_id, street_name,
			house_id, house_name, group_key, created_at, updated_at
	`

	var out service.DTEKGroupConfig
	err := r.pool.QueryRow(
		ctx,
		q,
		cfg.ProjectID,
		cfg.RegionID,
		cfg.RegionName,
		cfg.DSOID,
		cfg.DSOName,
		cfg.StreetID,
		cfg.StreetName,
		cfg.HouseID,
		cfg.HouseName,
		cfg.GroupKey,
	).Scan(
		&out.ProjectID,
		&out.RegionID,
		&out.RegionName,
		&out.DSOID,
		&out.DSOName,
		&out.StreetID,
		&out.StreetName,
		&out.HouseID,
		&out.HouseName,
		&out.GroupKey,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		return service.DTEKGroupConfig{}, err
	}
	return out, nil
}

func (r *DTEKGroupRepository) DeleteByProjectID(ctx context.Context, projectID int) error {
	const q = `DELETE FROM dtek_groups WHERE project_id = $1`
	_, err := r.pool.Exec(ctx, q, projectID)
	return err
}
