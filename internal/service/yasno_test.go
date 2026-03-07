package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type yasnoProjectStoreStub struct {
	project Project
}

func (s *yasnoProjectStoreStub) ListProjects(context.Context) ([]Project, error) {
	return nil, nil
}

func (s *yasnoProjectStoreStub) ListProjectsByUserID(_ context.Context, userID int64) ([]Project, error) {
	if s.project.UserID != userID {
		return []Project{}, nil
	}
	return []Project{s.project}, nil
}

func (s *yasnoProjectStoreStub) GetProjectBySlug(context.Context, string) (Project, bool, error) {
	return Project{}, false, nil
}

func (s *yasnoProjectStoreStub) GetProjectByID(_ context.Context, id int) (Project, bool, error) {
	if s.project.ID != id {
		return Project{}, false, nil
	}
	return s.project, true, nil
}

func (s *yasnoProjectStoreStub) CreateProject(context.Context, ProjectCreateInput) (Project, error) {
	return Project{}, errors.New("not implemented")
}

func (s *yasnoProjectStoreStub) UpdateProject(context.Context, ProjectUpdateInput) (Project, error) {
	return Project{}, errors.New("not implemented")
}

func (s *yasnoProjectStoreStub) DeleteProject(context.Context, int) error {
	return errors.New("not implemented")
}

type dtekGroupStoreStub struct {
	config DTEKGroupConfig
	found  bool
}

func (s *dtekGroupStoreStub) GetByProjectID(context.Context, int) (DTEKGroupConfig, bool, error) {
	return s.config, s.found, nil
}

func (s *dtekGroupStoreStub) Upsert(_ context.Context, cfg DTEKGroupConfig) (DTEKGroupConfig, error) {
	s.config = cfg
	s.found = true
	return cfg, nil
}

func (s *dtekGroupStoreStub) DeleteByProjectID(context.Context, int) error {
	s.found = false
	s.config = DTEKGroupConfig{}
	return nil
}

type yasnoClientStub struct {
	groupKey string
	planned  map[string]yasnoPlannedGroupResponse
	regions  []yasnoRegionResponse
	streets  []yasnoStreetResponse
	houses   []yasnoHouseResponse
	err      error
}

func (s *yasnoClientStub) ListRegions(context.Context) ([]yasnoRegionResponse, error) {
	return s.regions, s.err
}

func (s *yasnoClientStub) SearchStreets(context.Context, int, int, string) ([]yasnoStreetResponse, error) {
	return s.streets, s.err
}

func (s *yasnoClientStub) SearchHouses(context.Context, int, int, int, string) ([]yasnoHouseResponse, error) {
	return s.houses, s.err
}

func (s *yasnoClientStub) ResolveGroup(context.Context, int, int, int, int) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.groupKey, nil
}

func (s *yasnoClientStub) GetPlannedOutages(context.Context, int, int) (map[string]yasnoPlannedGroupResponse, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.planned, nil
}

func TestYasnoScheduleServicePreviewForUser(t *testing.T) {
	t.Parallel()

	projectCatalog := NewProjectCatalogService(&yasnoProjectStoreStub{
		project: Project{ID: 17, UserID: 42, Name: "Саксаганського 12А"},
	})
	store := &dtekGroupStoreStub{}
	client := &yasnoClientStub{
		groupKey: "3.1",
		planned: map[string]yasnoPlannedGroupResponse{
			"3.1": {
				UpdatedOn: "2026-03-07T08:15:00",
				Today: &yasnoPlannedDayResponse{
					Date:   "2026-03-07T00:00:00",
					Status: "ScheduleApplies",
					Slots: []yasnoPlannedSlotReponse{
						{Start: 480, End: 600, Type: "Definite"},
						{Start: 600, End: 1440, Type: "NotPlanned"},
					},
				},
				Tomorrow: &yasnoPlannedDayResponse{
					Date:   "2026-03-08T00:00:00",
					Status: "ScheduleApplies",
					Slots: []yasnoPlannedSlotReponse{
						{Start: 0, End: 240, Type: "Definite"},
					},
				},
			},
		},
	}

	svc := NewYasnoScheduleService(projectCatalog, store, client)
	config, schedule, err := svc.PreviewForUser(context.Background(), 42, 17, YasnoSelectionInput{
		RegionID:   1,
		RegionName: "Київ",
		DSOID:      11,
		DSOName:    "ДТЕК Київські електромережі",
		StreetID:   100,
		StreetName: "вул. Саксаганського",
		HouseID:    200,
		HouseName:  "12/А",
	})
	if err != nil {
		t.Fatalf("PreviewForUser error: %v", err)
	}

	if config.GroupKey != "3.1" {
		t.Fatalf("unexpected group key: %q", config.GroupKey)
	}
	if schedule.Address != "вул. Саксаганського, 12/А" {
		t.Fatalf("unexpected address: %q", schedule.Address)
	}
	if len(schedule.Days) != 2 {
		t.Fatalf("expected 2 days, got %d", len(schedule.Days))
	}
	if schedule.Days[0].WeekdayShort != "Сб" {
		t.Fatalf("unexpected weekday: %q", schedule.Days[0].WeekdayShort)
	}
	if schedule.UpdatedAt == nil {
		t.Fatalf("expected updatedAt to be set")
	}
}

func TestYasnoScheduleServiceGetForUserKeepsConfigOnScheduleError(t *testing.T) {
	t.Parallel()

	projectCatalog := NewProjectCatalogService(&yasnoProjectStoreStub{
		project: Project{ID: 17, UserID: 42},
	})
	store := &dtekGroupStoreStub{
		found: true,
		config: DTEKGroupConfig{
			ProjectID:  17,
			RegionID:   1,
			RegionName: "Київ",
			DSOID:      11,
			DSOName:    "ДТЕК Київські електромережі",
			StreetID:   100,
			StreetName: "вул. Саксаганського",
			HouseID:    200,
			HouseName:  "12/А",
			GroupKey:   "3.1",
			UpdatedAt:  time.Now().UTC(),
		},
	}
	client := &yasnoClientStub{
		planned: map[string]yasnoPlannedGroupResponse{},
	}

	svc := NewYasnoScheduleService(projectCatalog, store, client)
	config, schedule, scheduleError, err := svc.GetForUser(context.Background(), 42, 17)
	if err != nil {
		t.Fatalf("GetForUser error: %v", err)
	}
	if config == nil {
		t.Fatalf("expected config")
	}
	if schedule != nil {
		t.Fatalf("expected schedule to be nil on schedule error")
	}
	if scheduleError == "" {
		t.Fatalf("expected schedule error")
	}
}

func TestYasnoScheduleServiceSearchFiltersBlankLookupRows(t *testing.T) {
	t.Parallel()

	projectCatalog := NewProjectCatalogService(&yasnoProjectStoreStub{
		project: Project{ID: 17, UserID: 42},
	})
	client := &yasnoClientStub{
		regions: []yasnoRegionResponse{
			{
				ID:   25,
				Name: "Київ",
				DSOs: []yasnoDSOResponse{
					{ID: 902, Name: ""},
					{ID: 903, Name: "ДТЕК Київські електромережі"},
				},
			},
			{
				ID:   26,
				Name: "",
				DSOs: []yasnoDSOResponse{{ID: 1, Name: "ignored"}},
			},
		},
		streets: []yasnoStreetResponse{
			{ID: 1872, Name: ""},
			{ID: 1873, Value: "Саксаганського"},
		},
		houses: []yasnoHouseResponse{
			{ID: 0, Name: "bad"},
			{ID: 10, Name: ""},
			{ID: 11, Value: "12/А"},
		},
	}
	svc := NewYasnoScheduleService(projectCatalog, &dtekGroupStoreStub{}, client)

	regions, err := svc.ListRegions(context.Background(), 42, 17)
	if err != nil {
		t.Fatalf("ListRegions error: %v", err)
	}
	if len(regions) != 1 {
		t.Fatalf("expected 1 region after filtering, got %d", len(regions))
	}
	if len(regions[0].DSOs) != 1 {
		t.Fatalf("expected 1 dso after filtering, got %d", len(regions[0].DSOs))
	}

	streets, err := svc.SearchStreets(context.Background(), 42, 17, 25, 903, "Саксаганського")
	if err != nil {
		t.Fatalf("SearchStreets error: %v", err)
	}
	if len(streets) != 1 || streets[0].Name != "Саксаганського" {
		t.Fatalf("unexpected filtered streets: %#v", streets)
	}

	houses, err := svc.SearchHouses(context.Background(), 42, 17, 25, 1873, 903, "12")
	if err != nil {
		t.Fatalf("SearchHouses error: %v", err)
	}
	if len(houses) != 1 || houses[0].Name != "12/А" {
		t.Fatalf("unexpected filtered houses: %#v", houses)
	}
}
