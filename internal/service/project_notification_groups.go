package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrTelegramBotDisabled       = errors.New("telegram bot is not configured")
	ErrTelegramBotGroupNotFound  = errors.New("бот ще не бачить цю групу в оновленнях Telegram")
	ErrTelegramBotGroupAmbiguous = errors.New("знайдено кілька груп з такою назвою")
	ErrTelegramBotGroupConflict  = errors.New("ця Telegram група вже належить іншому власнику")
	ErrTelegramBotGroupMissing   = errors.New("telegram group subscription not found")
)

type ProjectTelegramBotGroup struct {
	VirtualUserID int64     `json:"virtualUserId"`
	TelegramID    int64     `json:"telegramId"`
	Title         string    `json:"title"`
	ChatType      string    `json:"chatType"`
	Username      string    `json:"username"`
	AddedAt       time.Time `json:"addedAt"`
}

type ProjectTelegramBotGroupUpsert struct {
	OwnerID    int64
	ProjectID  int
	TelegramID int64
	Title      string
	ChatType   string
	Username   string
}

func (s *ProjectNotificationService) ListTelegramBotGroupsByProject(ctx context.Context, ownerID int64, projectID int) ([]ProjectTelegramBotGroup, error) {
	if ownerID <= 0 || projectID <= 0 {
		return nil, ErrProjectInvalidData
	}
	if s == nil || s.store == nil || s.projectCatalog == nil {
		return []ProjectTelegramBotGroup{}, nil
	}
	if _, err := s.projectCatalog.GetByIDForUser(ctx, projectID, ownerID); err != nil {
		return nil, err
	}
	return s.store.ListTelegramBotGroupsByProjectID(ctx, ownerID, projectID)
}

func (s *ProjectNotificationService) AddTelegramBotGroupToProject(ctx context.Context, ownerID int64, projectID int, title string) (ProjectTelegramBotGroup, error) {
	if ownerID <= 0 || projectID <= 0 {
		return ProjectTelegramBotGroup{}, ErrProjectInvalidData
	}
	if strings.TrimSpace(title) == "" {
		return ProjectTelegramBotGroup{}, ErrProjectInvalidData
	}
	if s == nil || s.store == nil || s.projectCatalog == nil || s.telegramBot == nil || !s.telegramBot.Enabled() {
		return ProjectTelegramBotGroup{}, ErrTelegramBotDisabled
	}
	if _, err := s.projectCatalog.GetByIDForUser(ctx, projectID, ownerID); err != nil {
		return ProjectTelegramBotGroup{}, err
	}

	group, err := s.telegramBot.FindGroupChatByTitle(ctx, title)
	if err != nil {
		return ProjectTelegramBotGroup{}, err
	}

	item, err := s.store.UpsertTelegramBotGroupSubscription(ctx, ProjectTelegramBotGroupUpsert{
		OwnerID:    ownerID,
		ProjectID:  projectID,
		TelegramID: group.TelegramID,
		Title:      group.Title,
		ChatType:   group.ChatType,
		Username:   group.Username,
	})
	if err != nil {
		return ProjectTelegramBotGroup{}, err
	}
	return item, nil
}

func (s *ProjectNotificationService) RemoveTelegramBotGroupFromProject(ctx context.Context, ownerID int64, projectID int, virtualUserID int64) error {
	if ownerID <= 0 || projectID <= 0 || virtualUserID <= 0 {
		return ErrProjectInvalidData
	}
	if s == nil || s.store == nil || s.projectCatalog == nil {
		return ErrTelegramBotGroupMissing
	}
	if _, err := s.projectCatalog.GetByIDForUser(ctx, projectID, ownerID); err != nil {
		return err
	}
	removed, err := s.store.RemoveTelegramBotGroupSubscription(ctx, ownerID, projectID, virtualUserID)
	if err != nil {
		return fmt.Errorf("remove telegram bot group subscription: %w", err)
	}
	if !removed {
		return ErrTelegramBotGroupMissing
	}
	return nil
}
