package service

import (
	"context"
	"errors"
	"strings"
	"time"
)

type Project struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	UserID      int64     `json:"userId"`
	City        string    `json:"city"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type ProjectStore interface {
	ListProjects(ctx context.Context) ([]Project, error)
	GetProjectBySlug(ctx context.Context, slug string) (Project, bool, error)
}

type ProjectCatalogService struct {
	store ProjectStore
}

func NewProjectCatalogService(store ProjectStore) *ProjectCatalogService {
	return &ProjectCatalogService{store: store}
}

func (s *ProjectCatalogService) List(ctx context.Context) ([]Project, error) {
	if s == nil || s.store == nil {
		return []Project{}, nil
	}
	return s.store.ListProjects(ctx)
}

func (s *ProjectCatalogService) GetBySlug(ctx context.Context, slug string) (Project, bool, error) {
	if s == nil || s.store == nil {
		return Project{}, false, nil
	}

	slug = strings.TrimSpace(slug)
	if slug == "" {
		return Project{}, false, errors.New("slug is required")
	}
	return s.store.GetProjectBySlug(ctx, slug)
}
