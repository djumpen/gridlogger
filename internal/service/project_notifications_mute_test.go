package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// dispatchStoreStub drives one processProject run: a project that was available
// is now in outage, with one active subscriber.
type dispatchStoreStub struct {
	projectNotificationStoreStub
	state    ProjectStatusState
	upserted *ProjectStatusStateUpsert
}

func (s *dispatchStoreStub) ListActiveSubscribersByProjectID(context.Context, int) ([]ProjectNotificationSubscriber, error) {
	return []ProjectNotificationSubscriber{{UserID: 1, TelegramID: 555}}, nil
}

func (s *dispatchStoreStub) GetProjectStatusState(context.Context, int) (ProjectStatusState, bool, error) {
	return s.state, true, nil
}

func (s *dispatchStoreStub) UpsertProjectStatusState(_ context.Context, in ProjectStatusStateUpsert) error {
	s.upserted = &in
	return nil
}

type lastPingStub struct{ at time.Time }

func (s lastPingStub) GetLastPingAt(context.Context, int) (time.Time, bool, error) {
	return s.at, true, nil
}

func TestProcessProjectRespectsNotificationsEnabled(t *testing.T) {
	t.Parallel()

	for _, enabled := range []bool{true, false} {
		enabled := enabled
		name := map[bool]string{true: "enabled", false: "disabled"}[enabled]
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var sent atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/bottoken/sendMessage" {
					sent.Add(1)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok": true}`))
			}))
			defer server.Close()

			bot := NewTelegramBotService("token")
			bot.client = server.Client()
			bot.baseURL = server.URL

			now := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
			store := &dispatchStoreStub{state: ProjectStatusState{
				ProjectID:     11,
				LastStatus:    projectStatusAvailable,
				LastChangedAt: now.Add(-time.Hour),
			}}
			catalog := NewProjectCatalogService(&projectStoreStub{
				project: Project{ID: 11, UserID: 7, Name: "ЖК Сонце", NotificationsEnabled: enabled},
			})
			svc := NewProjectNotificationService(store, catalog, lastPingStub{at: now.Add(-10 * time.Minute)}, bot, 4*time.Minute)
			svc.nowFn = func() time.Time { return now }

			if err := svc.processProject(context.Background(), 11); err != nil {
				t.Fatalf("processProject error: %v", err)
			}

			wantSent := int32(0)
			wantNotified := ""
			if enabled {
				wantSent = 1
				wantNotified = projectStatusOutage
			}
			if got := sent.Load(); got != wantSent {
				t.Fatalf("sendMessage calls: got %d want %d", got, wantSent)
			}
			if store.upserted == nil {
				t.Fatal("expected status state to be persisted")
			}
			if store.upserted.LastStatus != projectStatusOutage {
				t.Fatalf("status must be tracked even when muted: got %q", store.upserted.LastStatus)
			}
			if store.upserted.LastNotifiedStatus != wantNotified {
				t.Fatalf("last notified status: got %q want %q", store.upserted.LastNotifiedStatus, wantNotified)
			}
		})
	}
}

type updateCaptureStoreStub struct {
	projectStoreStub
	got ProjectUpdateInput
}

func (s *updateCaptureStoreStub) UpdateProject(_ context.Context, in ProjectUpdateInput) (Project, error) {
	s.got = in
	return Project{ID: in.ID, NotificationsEnabled: in.NotificationsEnabled}, nil
}

func TestUpdateForUserNotificationsEnabled(t *testing.T) {
	t.Parallel()

	off := false
	tests := []struct {
		name    string
		current bool
		input   *bool
		want    bool
	}{
		{name: "nil keeps enabled", current: true, input: nil, want: true},
		{name: "nil keeps disabled", current: false, input: nil, want: false},
		{name: "explicit disable", current: true, input: &off, want: false},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := &updateCaptureStoreStub{projectStoreStub: projectStoreStub{
				project: Project{ID: 11, UserID: 7, NotificationsEnabled: tc.current},
			}}
			svc := NewProjectCatalogService(store)
			if _, err := svc.UpdateForUser(context.Background(), 7, 11, "Name", "Київ", "slug-11", true, tc.input); err != nil {
				t.Fatalf("UpdateForUser error: %v", err)
			}
			if store.got.NotificationsEnabled != tc.want {
				t.Fatalf("NotificationsEnabled: got %v want %v", store.got.NotificationsEnabled, tc.want)
			}
		})
	}
}
