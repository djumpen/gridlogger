package db

import (
	"context"
	"errors"

	"github.com/djumpen/gridlogger/internal/service"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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
		SELECT id, name, slug, user_id, city, description, is_public, created_at
		FROM projects
		WHERE is_public = TRUE
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
			&p.IsPublic,
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

func (r *ProjectRepository) ListProjectsByUserID(ctx context.Context, userID int64) ([]service.Project, error) {
	const q = `
		SELECT id, name, slug, user_id, city, description, is_public, created_at
		FROM projects
		WHERE user_id = $1
		ORDER BY id ASC
	`

	rows, err := r.pool.Query(ctx, q, userID)
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
			&p.IsPublic,
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
		SELECT id, name, slug, user_id, city, description, is_public, created_at
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
		&p.IsPublic,
		&p.CreatedAt,
	)
	if err == nil {
		return p, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return service.Project{}, false, nil
	}
	return service.Project{}, false, err
}

func (r *ProjectRepository) GetProjectByID(ctx context.Context, id int) (service.Project, bool, error) {
	const q = `
		SELECT id, name, slug, user_id, city, description, secret, is_public, created_at
		FROM projects
		WHERE id = $1
	`

	var p service.Project
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.UserID,
		&p.City,
		&p.Description,
		&p.Secret,
		&p.IsPublic,
		&p.CreatedAt,
	)
	if err == nil {
		return p, true, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return service.Project{}, false, nil
	}
	return service.Project{}, false, err
}

func (r *ProjectRepository) CreateProject(ctx context.Context, in service.ProjectCreateInput) (service.Project, error) {
	const q = `
		INSERT INTO projects (name, slug, user_id, city, secret, is_public)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, slug, user_id, city, description, secret, is_public, created_at
	`

	var p service.Project
	err := r.pool.QueryRow(ctx, q, in.Name, in.Slug, in.UserID, in.City, in.Secret, in.IsPublic).Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.UserID,
		&p.City,
		&p.Description,
		&p.Secret,
		&p.IsPublic,
		&p.CreatedAt,
	)
	if err == nil {
		return p, nil
	}
	if isSlugUniqueViolation(err) {
		return service.Project{}, service.ErrProjectSlugTaken
	}
	return service.Project{}, err
}

func (r *ProjectRepository) UpdateProject(ctx context.Context, in service.ProjectUpdateInput) (service.Project, error) {
	const q = `
		UPDATE projects
		SET name = $1,
			slug = $2,
			city = $3,
			is_public = $4
		WHERE id = $5
		RETURNING id, name, slug, user_id, city, description, secret, is_public, created_at
	`

	var p service.Project
	err := r.pool.QueryRow(ctx, q, in.Name, in.Slug, in.City, in.IsPublic, in.ID).Scan(
		&p.ID,
		&p.Name,
		&p.Slug,
		&p.UserID,
		&p.City,
		&p.Description,
		&p.Secret,
		&p.IsPublic,
		&p.CreatedAt,
	)
	if err == nil {
		return p, nil
	}
	if isSlugUniqueViolation(err) {
		return service.Project{}, service.ErrProjectSlugTaken
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return service.Project{}, service.ErrProjectNotFound
	}
	return service.Project{}, err
}

func (r *ProjectRepository) DeleteProject(ctx context.Context, id int) error {
	const q = `DELETE FROM projects WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, q, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return service.ErrProjectNotFound
	}
	return nil
}

func isSlugUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if pgErr.Code != "23505" {
		return false
	}
	return pgErr.ConstraintName == "idx_projects_slug"
}
