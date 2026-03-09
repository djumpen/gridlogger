package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

const (
	projectStatusAvailable = "available"
	projectStatusOutage    = "outage"
	projectStatusUnknown   = "unknown"
)

type ProjectNotificationSubscriber struct {
	UserID     int64
	TelegramID int64
	Username   string
	FirstName  string
	LastName   string
}

type ProjectStatusState struct {
	ProjectID          int
	LastStatus         string
	LastChangedAt      time.Time
	LastNotifiedStatus string
	LastNotifiedAt     *time.Time
	UpdatedAt          time.Time
}

type ProjectStatusStateUpsert struct {
	ProjectID          int
	LastStatus         string
	LastChangedAt      time.Time
	LastNotifiedStatus string
	LastNotifiedAt     *time.Time
}

type ProjectNotificationStore interface {
	GetProjectNotificationSubscription(ctx context.Context, userID int64, projectID int) (bool, error)
	SetProjectNotificationSubscription(ctx context.Context, userID int64, projectID int, subscribed bool) (bool, error)
	CountActiveProjectNotificationSubscriptions(ctx context.Context, projectID int) (int, error)
	ListActiveSubscribedProjectsByUserID(ctx context.Context, userID int64) ([]Project, error)
	ListProjectIDsWithActiveSubscriptions(ctx context.Context) ([]int, error)
	ListActiveSubscribersByProjectID(ctx context.Context, projectID int) ([]ProjectNotificationSubscriber, error)
	ListTelegramBotGroupsByProjectID(ctx context.Context, ownerID int64, projectID int) ([]ProjectTelegramBotGroup, error)
	UpsertTelegramBotGroupSubscription(ctx context.Context, in ProjectTelegramBotGroupUpsert) (ProjectTelegramBotGroup, error)
	RemoveTelegramBotGroupSubscription(ctx context.Context, ownerID int64, projectID int, virtualUserID int64) (bool, error)
	GetProjectStatusState(ctx context.Context, projectID int) (ProjectStatusState, bool, error)
	UpsertProjectStatusState(ctx context.Context, in ProjectStatusStateUpsert) error
	TryAcquireNotificationDispatcherLock(ctx context.Context) (bool, error)
	ReleaseNotificationDispatcherLock(ctx context.Context) error
}

type ProjectPingStatusStore interface {
	GetLastPingAt(ctx context.Context, projectID int) (time.Time, bool, error)
}

type ProjectNotificationService struct {
	store           ProjectNotificationStore
	projectCatalog  *ProjectCatalogService
	pings           ProjectPingStatusStore
	telegramBot     *TelegramBotService
	outageThreshold time.Duration
	nowFn           func() time.Time
}

func NewProjectNotificationService(
	store ProjectNotificationStore,
	projectCatalog *ProjectCatalogService,
	pings ProjectPingStatusStore,
	telegramBot *TelegramBotService,
	outageThreshold time.Duration,
) *ProjectNotificationService {
	return &ProjectNotificationService{
		store:           store,
		projectCatalog:  projectCatalog,
		pings:           pings,
		telegramBot:     telegramBot,
		outageThreshold: outageThreshold,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s *ProjectNotificationService) GetSubscriptionForUser(ctx context.Context, userID int64, projectID int) (bool, error) {
	if userID <= 0 || projectID <= 0 {
		return false, ErrProjectInvalidData
	}
	if s == nil || s.store == nil || s.projectCatalog == nil {
		return false, errors.New("project notifications are not configured")
	}
	_, found, err := s.projectCatalog.GetByID(ctx, projectID)
	if err != nil {
		return false, err
	}
	if !found {
		return false, ErrProjectNotFound
	}
	return s.store.GetProjectNotificationSubscription(ctx, userID, projectID)
}

func (s *ProjectNotificationService) SetSubscriptionForUser(ctx context.Context, userID int64, projectID int, subscribed bool) (bool, error) {
	if userID <= 0 || projectID <= 0 {
		return false, ErrProjectInvalidData
	}
	if s == nil || s.store == nil || s.projectCatalog == nil {
		return false, errors.New("project notifications are not configured")
	}
	_, found, err := s.projectCatalog.GetByID(ctx, projectID)
	if err != nil {
		return false, err
	}
	if !found {
		return false, ErrProjectNotFound
	}
	return s.store.SetProjectNotificationSubscription(ctx, userID, projectID, subscribed)
}

func (s *ProjectNotificationService) ListActiveSubscribedProjectsByUserID(ctx context.Context, userID int64) ([]Project, error) {
	if userID <= 0 {
		return nil, ErrProjectInvalidData
	}
	if s == nil || s.store == nil {
		return []Project{}, nil
	}
	return s.store.ListActiveSubscribedProjectsByUserID(ctx, userID)
}

func (s *ProjectNotificationService) CountActiveSubscriptionsByProjectID(ctx context.Context, projectID int) (int, error) {
	if projectID <= 0 {
		return 0, ErrProjectInvalidData
	}
	if s == nil || s.store == nil || s.projectCatalog == nil {
		return 0, errors.New("project notifications are not configured")
	}
	_, found, err := s.projectCatalog.GetByID(ctx, projectID)
	if err != nil {
		return 0, err
	}
	if !found {
		return 0, ErrProjectNotFound
	}
	return s.store.CountActiveProjectNotificationSubscriptions(ctx, projectID)
}

func (s *ProjectNotificationService) PollAndNotify(ctx context.Context) error {
	if s == nil || s.store == nil || s.projectCatalog == nil || s.pings == nil || s.telegramBot == nil {
		return nil
	}

	locked, err := s.store.TryAcquireNotificationDispatcherLock(ctx)
	if err != nil {
		return fmt.Errorf("acquire notification dispatcher lock: %w", err)
	}
	if !locked {
		return nil
	}
	defer func() {
		if err := s.store.ReleaseNotificationDispatcherLock(context.Background()); err != nil {
			log.Printf("release notification dispatcher lock error: %v", err)
		}
	}()

	projectIDs, err := s.store.ListProjectIDsWithActiveSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("list subscribed projects: %w", err)
	}
	for _, projectID := range projectIDs {
		if err := s.processProject(ctx, projectID); err != nil {
			log.Printf("process project notifications error: project_id=%d err=%v", projectID, err)
		}
	}
	return nil
}

func (s *ProjectNotificationService) processProject(ctx context.Context, projectID int) error {
	project, found, err := s.projectCatalog.GetByID(ctx, projectID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}

	now := s.nowFn().UTC()
	currentStatus, err := s.currentProjectStatus(ctx, projectID, now)
	if err != nil {
		return err
	}

	state, foundState, err := s.store.GetProjectStatusState(ctx, projectID)
	if err != nil {
		return err
	}
	if !foundState {
		return s.store.UpsertProjectStatusState(ctx, ProjectStatusStateUpsert{
			ProjectID:          projectID,
			LastStatus:         currentStatus,
			LastChangedAt:      now,
			LastNotifiedStatus: "",
			LastNotifiedAt:     nil,
		})
	}

	if state.LastStatus == currentStatus {
		return nil
	}

	shouldNotify := shouldNotifyTransition(state.LastStatus, currentStatus)
	lastNotifiedStatus := state.LastNotifiedStatus
	lastNotifiedAt := state.LastNotifiedAt

	if shouldNotify {
		subscribers, err := s.store.ListActiveSubscribersByProjectID(ctx, projectID)
		if err != nil {
			return err
		}
		message := buildProjectStatusNotificationMessage(project, currentStatus, now.Sub(state.LastChangedAt))
		for _, sub := range subscribers {
			if err := s.telegramBot.SendTelegramMessage(ctx, sub.TelegramID, message); err != nil {
				log.Printf(
					"send telegram status notification failed: project_id=%d user_id=%d telegram_id=%d err=%v",
					projectID,
					sub.UserID,
					sub.TelegramID,
					err,
				)
				continue
			}
		}
		log.Printf(
			"sent project status notifications: project_id=%d previous_status=%s current_status=%s subscribers=%d",
			projectID,
			state.LastStatus,
			currentStatus,
			len(subscribers),
		)
		lastNotifiedStatus = currentStatus
		lastNotifiedAt = &now
	}

	return s.store.UpsertProjectStatusState(ctx, ProjectStatusStateUpsert{
		ProjectID:          projectID,
		LastStatus:         currentStatus,
		LastChangedAt:      now,
		LastNotifiedStatus: lastNotifiedStatus,
		LastNotifiedAt:     lastNotifiedAt,
	})
}

func (s *ProjectNotificationService) currentProjectStatus(ctx context.Context, projectID int, now time.Time) (string, error) {
	lastPingAt, found, err := s.pings.GetLastPingAt(ctx, projectID)
	if err != nil {
		return "", err
	}
	if !found {
		return projectStatusUnknown, nil
	}
	if now.Sub(lastPingAt.UTC()) <= s.outageThreshold {
		return projectStatusAvailable, nil
	}
	return projectStatusOutage, nil
}

func shouldNotifyTransition(previous, next string) bool {
	prev := strings.ToLower(strings.TrimSpace(previous))
	cur := strings.ToLower(strings.TrimSpace(next))
	if prev == cur {
		return false
	}
	if prev == projectStatusUnknown || cur == projectStatusUnknown {
		return false
	}
	return (prev == projectStatusAvailable || prev == projectStatusOutage) &&
		(cur == projectStatusAvailable || cur == projectStatusOutage)
}

func buildProjectStatusNotificationMessage(project Project, status string, duration time.Duration) string {
	name := strings.TrimSpace(project.Name)
	if name == "" {
		name = fmt.Sprintf("Проєкт #%d", project.ID)
	}
	link := fmt.Sprintf("https://svitlo.homes/%s", project.Slug)
	durationLabel := formatDurationUA(duration)

	switch status {
	case projectStatusAvailable:
		return fmt.Sprintf(
			"🟢 Світло зʼявилося: %s\nВідключення тривало %s\n%s",
			name,
			durationLabel,
			link,
		)
	case projectStatusOutage:
		return fmt.Sprintf(
			"🔴 Світло відсутнє: %s\nСвітло було доступне %s\n%s",
			name,
			durationLabel,
			link,
		)
	default:
		return fmt.Sprintf("ℹ️ Оновлення статусу: %s\n%s", name, link)
	}
}

func formatDurationUA(duration time.Duration) string {
	if duration <= 0 {
		return "0хв"
	}

	totalMinutes := int(duration.Round(time.Minute).Minutes())
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	if hours == 0 {
		return fmt.Sprintf("%dхв", minutes)
	}
	if minutes == 0 {
		return fmt.Sprintf("%dгод", hours)
	}
	return fmt.Sprintf("%dгод %dхв", hours, minutes)
}
