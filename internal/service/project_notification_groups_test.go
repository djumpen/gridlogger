package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type projectNotificationStoreStub struct {
	groups       []ProjectTelegramBotGroup
	upsertInput  ProjectTelegramBotGroupUpsert
	upsertResult ProjectTelegramBotGroup
	removeResult bool
}

func (s *projectNotificationStoreStub) GetProjectNotificationSubscription(context.Context, int64, int) (bool, error) {
	return false, nil
}

func (s *projectNotificationStoreStub) SetProjectNotificationSubscription(context.Context, int64, int, bool) (bool, error) {
	return false, nil
}

func (s *projectNotificationStoreStub) ListActiveSubscribedProjectsByUserID(context.Context, int64) ([]Project, error) {
	return nil, nil
}

func (s *projectNotificationStoreStub) ListProjectIDsWithActiveSubscriptions(context.Context) ([]int, error) {
	return nil, nil
}

func (s *projectNotificationStoreStub) ListActiveSubscribersByProjectID(context.Context, int) ([]ProjectNotificationSubscriber, error) {
	return nil, nil
}

func (s *projectNotificationStoreStub) ListTelegramBotGroupsByProjectID(_ context.Context, ownerID int64, projectID int) ([]ProjectTelegramBotGroup, error) {
	if ownerID != 7 || projectID != 11 {
		return nil, ErrProjectForbidden
	}
	return s.groups, nil
}

func (s *projectNotificationStoreStub) UpsertTelegramBotGroupSubscription(_ context.Context, in ProjectTelegramBotGroupUpsert) (ProjectTelegramBotGroup, error) {
	s.upsertInput = in
	if s.upsertResult.VirtualUserID == 0 {
		s.upsertResult = ProjectTelegramBotGroup{
			VirtualUserID: 91,
			TelegramID:    in.TelegramID,
			Title:         in.Title,
			ChatType:      in.ChatType,
			Username:      in.Username,
			AddedAt:       time.Unix(1710000000, 0).UTC(),
		}
	}
	return s.upsertResult, nil
}

func (s *projectNotificationStoreStub) RemoveTelegramBotGroupSubscription(context.Context, int64, int, int64) (bool, error) {
	return s.removeResult, nil
}

func (s *projectNotificationStoreStub) GetProjectStatusState(context.Context, int) (ProjectStatusState, bool, error) {
	return ProjectStatusState{}, false, nil
}

func (s *projectNotificationStoreStub) UpsertProjectStatusState(context.Context, ProjectStatusStateUpsert) error {
	return nil
}

func (s *projectNotificationStoreStub) TryAcquireNotificationDispatcherLock(context.Context) (bool, error) {
	return false, nil
}

func (s *projectNotificationStoreStub) ReleaseNotificationDispatcherLock(context.Context) error {
	return nil
}

type projectStoreStub struct {
	project Project
}

func (s *projectStoreStub) ListProjects(context.Context) ([]Project, error) {
	return nil, nil
}

func (s *projectStoreStub) ListProjectsByUserID(_ context.Context, userID int64) ([]Project, error) {
	if s.project.UserID != userID {
		return []Project{}, nil
	}
	return []Project{s.project}, nil
}

func (s *projectStoreStub) GetProjectBySlug(context.Context, string) (Project, bool, error) {
	return Project{}, false, nil
}

func (s *projectStoreStub) GetProjectByID(_ context.Context, id int) (Project, bool, error) {
	if s.project.ID != id {
		return Project{}, false, nil
	}
	return s.project, true, nil
}

func (s *projectStoreStub) CreateProject(context.Context, ProjectCreateInput) (Project, error) {
	return Project{}, errors.New("not implemented")
}

func (s *projectStoreStub) UpdateProject(context.Context, ProjectUpdateInput) (Project, error) {
	return Project{}, errors.New("not implemented")
}

func (s *projectStoreStub) DeleteProject(context.Context, int) error {
	return errors.New("not implemented")
}

func TestProjectNotificationServiceAddTelegramBotGroupToProject(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bottoken/getUpdates" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"result": [
				{"update_id": 1, "my_chat_member": {"chat": {"id": -1009876543210, "type": "supergroup", "title": "Світло ЖК Сонце", "username": "sunlight_group"}}}
			]
		}`))
	}))
	defer server.Close()

	bot := NewTelegramBotService("token")
	bot.client = server.Client()
	bot.baseURL = server.URL

	store := &projectNotificationStoreStub{}
	projectCatalog := NewProjectCatalogService(&projectStoreStub{
		project: Project{ID: 11, UserID: 7, Name: "ЖК Сонце"},
	})
	svc := NewProjectNotificationService(store, projectCatalog, nil, bot, 2*time.Minute)

	group, err := svc.AddTelegramBotGroupToProject(context.Background(), 7, 11, "Світло ЖК Сонце")
	if err != nil {
		t.Fatalf("AddTelegramBotGroupToProject error: %v", err)
	}
	if group.TelegramID != -1009876543210 {
		t.Fatalf("unexpected telegram id: %d", group.TelegramID)
	}
	if store.upsertInput.OwnerID != 7 || store.upsertInput.ProjectID != 11 {
		t.Fatalf("unexpected upsert input: %+v", store.upsertInput)
	}
	if store.upsertInput.Title != "Світло ЖК Сонце" {
		t.Fatalf("unexpected title: %q", store.upsertInput.Title)
	}
	if store.upsertInput.ChatType != "supergroup" {
		t.Fatalf("unexpected chat type: %q", store.upsertInput.ChatType)
	}
}

func TestProjectNotificationServiceRemoveTelegramBotGroupFromProjectMissing(t *testing.T) {
	t.Parallel()

	store := &projectNotificationStoreStub{}
	projectCatalog := NewProjectCatalogService(&projectStoreStub{
		project: Project{ID: 11, UserID: 7, Name: "ЖК Сонце"},
	})
	svc := NewProjectNotificationService(store, projectCatalog, nil, nil, 2*time.Minute)

	err := svc.RemoveTelegramBotGroupFromProject(context.Background(), 7, 11, 91)
	if !errors.Is(err, ErrTelegramBotGroupMissing) {
		t.Fatalf("expected ErrTelegramBotGroupMissing, got %v", err)
	}
}
