package db

import (
	"context"

	"github.com/djumpen/gridlogger/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ProjectRepository struct {
	pool *pgxpool.Pool
}

func NewProjectRepository(pool *pgxpool.Pool) *ProjectRepository {
	return &ProjectRepository{pool: pool}
}

func (r *ProjectRepository) ListProjects(ctx context.Context) ([]service.Project, error) {
	const q = `
		SELECT id, name, slug, user_id, city, description, created_at
		FROM projects
		ORDER BY name ASC, id ASC
	`

	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]service.Project, 0)
	for rows.Next() {
		var p service.Project
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Slug,
			&p.UserID,
			&p.City,
			&p.Description,
			&p.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *ProjectRepository) GetProjectBySlug(ctx context.Context, slug string) (service.Project, bool, error) {
	const q = `
		SELECT id, name, slug, user_id, city, description, created_at
		FROM projects
		WHERE slug = $1
	`

	var p service.Project
	err := r.pool.QueryRow(ctx, q, slug).Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.UserID,
		&p.City,
		&p.Description,
		&p.CreatedAt,
	)
	if err == nil {
		return p, true, nil
	}
	if err == pgx.ErrNoRows {
		return service.Project{}, false, nil
	}
	return service.Project{}, false, err
}
