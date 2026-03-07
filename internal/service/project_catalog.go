package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	ErrProjectInvalidData = errors.New("invalid project data")
	ErrProjectInvalidSlug = errors.New("invalid project slug")
	ErrProjectSlugTaken   = errors.New("project slug is already taken")
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectForbidden   = errors.New("forbidden")
)

var slugPattern = regexp.MustCompile(`^[a-z0-9-]+$`)

const reservedProjectSlugAPI = "api"

type Project struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Slug              string    `json:"slug"`
	UserID            int64     `json:"userId"`
	City              string    `json:"city"`
	Description       string    `json:"description"`
	Secret            string    `json:"secret,omitempty"`
	IsPublic          bool      `json:"isPublic"`
	HasOutageSchedule bool      `json:"hasOutageSchedule"`
	CreatedAt         time.Time `json:"createdAt"`
}

type ProjectCreateInput struct {
	Name     string
	Slug     string
	UserID   int64
	City     string
	Secret   string
	IsPublic bool
}

type ProjectUpdateInput struct {
	ID       int
	Name     string
	Slug     string
	City     string
	UserID   int64
	IsPublic bool
}

type ProjectStore interface {
	ListProjects(ctx context.Context) ([]Project, error)
	ListProjectsByUserID(ctx context.Context, userID int64) ([]Project, error)
	GetProjectBySlug(ctx context.Context, slug string) (Project, bool, error)
	GetProjectByID(ctx context.Context, id int) (Project, bool, error)
	CreateProject(ctx context.Context, in ProjectCreateInput) (Project, error)
	UpdateProject(ctx context.Context, in ProjectUpdateInput) (Project, error)
	DeleteProject(ctx context.Context, id int) error
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

func (s *ProjectCatalogService) ListByUserID(ctx context.Context, userID int64) ([]Project, error) {
	if s == nil || s.store == nil {
		return []Project{}, nil
	}
	if userID <= 0 {
		return nil, ErrProjectInvalidData
	}
	return s.store.ListProjectsByUserID(ctx, userID)
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

func (s *ProjectCatalogService) GetByID(ctx context.Context, id int) (Project, bool, error) {
	if s == nil || s.store == nil {
		return Project{}, false, nil
	}
	if id <= 0 {
		return Project{}, false, ErrProjectInvalidData
	}
	return s.store.GetProjectByID(ctx, id)
}

func (s *ProjectCatalogService) GetByIDForUser(ctx context.Context, projectID int, userID int64) (Project, error) {
	if userID <= 0 || projectID <= 0 {
		return Project{}, ErrProjectInvalidData
	}
	project, found, err := s.GetByID(ctx, projectID)
	if err != nil {
		return Project{}, err
	}
	if !found {
		return Project{}, ErrProjectNotFound
	}
	if project.UserID != userID {
		return Project{}, ErrProjectForbidden
	}
	return project, nil
}

func (s *ProjectCatalogService) CreateForUser(ctx context.Context, userID int64, name, city, slug string, isPublic bool) (Project, error) {
	if s == nil || s.store == nil {
		return Project{}, fmt.Errorf("project store is not configured")
	}
	if userID <= 0 {
		return Project{}, ErrProjectInvalidData
	}

	name = strings.TrimSpace(name)
	city = strings.TrimSpace(city)
	slug, err := normalizeSlug(slug)
	if err != nil {
		return Project{}, err
	}
	if name == "" || city == "" {
		return Project{}, ErrProjectInvalidData
	}

	secret, err := generateProjectSecret()
	if err != nil {
		return Project{}, fmt.Errorf("generate project secret: %w", err)
	}

	project, err := s.store.CreateProject(ctx, ProjectCreateInput{
		UserID:   userID,
		Name:     name,
		City:     city,
		Slug:     slug,
		Secret:   secret,
		IsPublic: isPublic,
	})
	if err != nil {
		return Project{}, err
	}
	return project, nil
}

func (s *ProjectCatalogService) UpdateForUser(ctx context.Context, userID int64, projectID int, name, city, slug string, isPublic bool) (Project, error) {
	if s == nil || s.store == nil {
		return Project{}, fmt.Errorf("project store is not configured")
	}
	if userID <= 0 || projectID <= 0 {
		return Project{}, ErrProjectInvalidData
	}

	name = strings.TrimSpace(name)
	city = strings.TrimSpace(city)
	slug, err := normalizeSlug(slug)
	if err != nil {
		return Project{}, err
	}
	if name == "" || city == "" {
		return Project{}, ErrProjectInvalidData
	}

	current, err := s.GetByIDForUser(ctx, projectID, userID)
	if err != nil {
		return Project{}, err
	}

	project, err := s.store.UpdateProject(ctx, ProjectUpdateInput{
		ID:       current.ID,
		UserID:   current.UserID,
		Name:     name,
		City:     city,
		Slug:     slug,
		IsPublic: isPublic,
	})
	if err != nil {
		return Project{}, err
	}
	project.Secret = current.Secret
	return project, nil
}

func (s *ProjectCatalogService) DeleteForUser(ctx context.Context, userID int64, projectID int) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("project store is not configured")
	}
	if userID <= 0 || projectID <= 0 {
		return ErrProjectInvalidData
	}

	current, err := s.GetByIDForUser(ctx, projectID, userID)
	if err != nil {
		return err
	}
	return s.store.DeleteProject(ctx, current.ID)
}

func normalizeSlug(raw string) (string, error) {
	slug := strings.ToLower(strings.TrimSpace(raw))
	if len(slug) < 3 {
		return "", ErrProjectInvalidSlug
	}
	if !slugPattern.MatchString(slug) {
		return "", ErrProjectInvalidSlug
	}
	if slug == reservedProjectSlugAPI {
		return "", ErrProjectInvalidSlug
	}
	return slug, nil
}

func generateProjectSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
