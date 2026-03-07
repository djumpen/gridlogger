package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrYasnoInvalidInput    = errors.New("invalid yasno data")
	ErrYasnoNotConfigured   = errors.New("yasno schedule is not configured for this project")
	ErrYasnoLookupNotFound  = errors.New("yasno lookup not found")
	ErrYasnoScheduleMissing = errors.New("yasno schedule was not found for this group")
	ErrYasnoUnavailable     = errors.New("yasno api is unavailable")
)

type DTEKGroupConfig struct {
	ProjectID  int       `json:"projectId"`
	RegionID   int       `json:"regionId"`
	RegionName string    `json:"regionName"`
	DSOID      int       `json:"dsoId"`
	DSOName    string    `json:"dsoName"`
	StreetID   int       `json:"streetId"`
	StreetName string    `json:"streetName"`
	HouseID    int       `json:"houseId"`
	HouseName  string    `json:"houseName"`
	GroupKey   string    `json:"group"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type YasnoSelectionInput struct {
	RegionID   int    `json:"regionId"`
	RegionName string `json:"regionName"`
	DSOID      int    `json:"dsoId"`
	DSOName    string `json:"dsoName"`
	StreetID   int    `json:"streetId"`
	StreetName string `json:"streetName"`
	HouseID    int    `json:"houseId"`
	HouseName  string `json:"houseName"`
}

type YasnoRegion struct {
	ID   int        `json:"id"`
	Name string     `json:"name"`
	DSOs []YasnoDSO `json:"dsos"`
}

type YasnoDSO struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type YasnoStreet struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type YasnoHouse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type YasnoSchedule struct {
	GroupKey  string             `json:"group"`
	Address   string             `json:"address"`
	UpdatedAt *time.Time         `json:"updatedAt,omitempty"`
	Days      []YasnoScheduleDay `json:"days"`
}

type YasnoScheduleDay struct {
	Key          string              `json:"key"`
	Label        string              `json:"label"`
	WeekdayShort string              `json:"weekdayShort"`
	Date         string              `json:"date"`
	Status       string              `json:"status"`
	Slots        []YasnoScheduleSlot `json:"slots"`
}

type YasnoScheduleSlot struct {
	StartMinute int    `json:"startMinute"`
	EndMinute   int    `json:"endMinute"`
	Type        string `json:"type"`
}

type DTEKGroupStore interface {
	GetByProjectID(ctx context.Context, projectID int) (DTEKGroupConfig, bool, error)
	Upsert(ctx context.Context, cfg DTEKGroupConfig) (DTEKGroupConfig, error)
	DeleteByProjectID(ctx context.Context, projectID int) error
}

type YasnoDirectoryClient interface {
	ListRegions(ctx context.Context) ([]yasnoRegionResponse, error)
	SearchStreets(ctx context.Context, regionID, dsoID int, query string) ([]yasnoStreetResponse, error)
	SearchHouses(ctx context.Context, regionID, streetID, dsoID int, query string) ([]yasnoHouseResponse, error)
	ResolveGroup(ctx context.Context, regionID, streetID, houseID, dsoID int) (string, error)
	GetPlannedOutages(ctx context.Context, regionID, dsoID int) (map[string]yasnoPlannedGroupResponse, error)
}

type YasnoScheduleService struct {
	projectCatalog *ProjectCatalogService
	store          DTEKGroupStore
	client         YasnoDirectoryClient
	location       *time.Location
}

type YasnoClient struct {
	baseURL  string
	client   *http.Client
	location *time.Location
}

type yasnoRegionResponse struct {
	ID   int                `json:"id"`
	Name string             `json:"value"`
	DSOs []yasnoDSOResponse `json:"dsos"`
}

type yasnoDSOResponse struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type yasnoStreetResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type yasnoHouseResponse struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type yasnoGroupResponse struct {
	Group    any `json:"group"`
	Subgroup any `json:"subgroup"`
}

type yasnoPlannedGroupResponse struct {
	UpdatedOn string                   `json:"updatedOn"`
	Today     *yasnoPlannedDayResponse `json:"today"`
	Tomorrow  *yasnoPlannedDayResponse `json:"tomorrow"`
}

type yasnoPlannedDayResponse struct {
	Date   string                    `json:"date"`
	Status string                    `json:"status"`
	Slots  []yasnoPlannedSlotReponse `json:"slots"`
}

type yasnoPlannedSlotReponse struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Type  string `json:"type"`
}

func NewYasnoScheduleService(projectCatalog *ProjectCatalogService, store DTEKGroupStore, client YasnoDirectoryClient) *YasnoScheduleService {
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		loc = time.UTC
	}
	return &YasnoScheduleService{
		projectCatalog: projectCatalog,
		store:          store,
		client:         client,
		location:       loc,
	}
}

func NewYasnoClient(baseURL string, timeout time.Duration) *YasnoClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	loc, err := time.LoadLocation("Europe/Kyiv")
	if err != nil {
		loc = time.UTC
	}
	return &YasnoClient{
		baseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		client: &http.Client{
			Timeout: timeout,
		},
		location: loc,
	}
}

func (s *YasnoScheduleService) ListRegions(ctx context.Context, userID int64, projectID int) ([]YasnoRegion, error) {
	if _, err := s.requireProjectOwner(ctx, userID, projectID); err != nil {
		return nil, err
	}
	regions, err := s.client.ListRegions(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]YasnoRegion, 0, len(regions))
	for _, item := range regions {
		regionName := strings.TrimSpace(item.Name)
		if regionName == "" {
			continue
		}
		dsos := make([]YasnoDSO, 0, len(item.DSOs))
		for _, dso := range item.DSOs {
			dsoName := strings.TrimSpace(dso.Name)
			if dsoName == "" {
				continue
			}
			dsos = append(dsos, YasnoDSO{
				ID:   dso.ID,
				Name: dsoName,
			})
		}
		out = append(out, YasnoRegion{
			ID:   item.ID,
			Name: regionName,
			DSOs: dsos,
		})
	}
	return out, nil
}

func (s *YasnoScheduleService) SearchStreets(ctx context.Context, userID int64, projectID, regionID, dsoID int, query string) ([]YasnoStreet, error) {
	if _, err := s.requireProjectOwner(ctx, userID, projectID); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if regionID <= 0 || dsoID <= 0 || query == "" {
		return nil, ErrYasnoInvalidInput
	}
	items, err := s.client.SearchStreets(ctx, regionID, dsoID, query)
	if err != nil {
		return nil, err
	}
	out := make([]YasnoStreet, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = strings.TrimSpace(item.Value)
		}
		if item.ID <= 0 || name == "" {
			continue
		}
		out = append(out, YasnoStreet{ID: item.ID, Name: name})
	}
	return out, nil
}

func (s *YasnoScheduleService) SearchHouses(ctx context.Context, userID int64, projectID, regionID, streetID, dsoID int, query string) ([]YasnoHouse, error) {
	if _, err := s.requireProjectOwner(ctx, userID, projectID); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if regionID <= 0 || streetID <= 0 || dsoID <= 0 || query == "" {
		return nil, ErrYasnoInvalidInput
	}
	items, err := s.client.SearchHouses(ctx, regionID, streetID, dsoID, query)
	if err != nil {
		return nil, err
	}
	out := make([]YasnoHouse, 0, len(items))
	for _, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = strings.TrimSpace(item.Value)
		}
		if item.ID <= 0 || name == "" {
			continue
		}
		out = append(out, YasnoHouse{ID: item.ID, Name: name})
	}
	return out, nil
}

func (s *YasnoScheduleService) PreviewForUser(ctx context.Context, userID int64, projectID int, in YasnoSelectionInput) (DTEKGroupConfig, YasnoSchedule, error) {
	if _, err := s.requireProjectOwner(ctx, userID, projectID); err != nil {
		return DTEKGroupConfig{}, YasnoSchedule{}, err
	}
	return s.previewBySelection(ctx, projectID, in)
}

func (s *YasnoScheduleService) SaveForUser(ctx context.Context, userID int64, projectID int, in YasnoSelectionInput) (DTEKGroupConfig, YasnoSchedule, error) {
	if _, err := s.requireProjectOwner(ctx, userID, projectID); err != nil {
		return DTEKGroupConfig{}, YasnoSchedule{}, err
	}
	cfg, schedule, err := s.previewBySelection(ctx, projectID, in)
	if err != nil {
		return DTEKGroupConfig{}, YasnoSchedule{}, err
	}
	saved, err := s.store.Upsert(ctx, cfg)
	if err != nil {
		return DTEKGroupConfig{}, YasnoSchedule{}, err
	}
	return saved, schedule, nil
}

func (s *YasnoScheduleService) GetForUser(ctx context.Context, userID int64, projectID int) (*DTEKGroupConfig, *YasnoSchedule, string, error) {
	if _, err := s.requireProjectOwner(ctx, userID, projectID); err != nil {
		return nil, nil, "", err
	}
	cfg, found, err := s.store.GetByProjectID(ctx, projectID)
	if err != nil {
		return nil, nil, "", err
	}
	if !found {
		return nil, nil, "", nil
	}
	schedule, err := s.scheduleForConfig(ctx, cfg)
	if err != nil {
		return &cfg, nil, err.Error(), nil
	}
	return &cfg, &schedule, "", nil
}

func (s *YasnoScheduleService) DeleteForUser(ctx context.Context, userID int64, projectID int) error {
	if _, err := s.requireProjectOwner(ctx, userID, projectID); err != nil {
		return err
	}
	return s.store.DeleteByProjectID(ctx, projectID)
}

func (s *YasnoScheduleService) GetPublicSchedule(ctx context.Context, projectID int) (YasnoSchedule, error) {
	if s == nil || s.projectCatalog == nil || s.store == nil || s.client == nil {
		return YasnoSchedule{}, ErrYasnoUnavailable
	}
	if projectID <= 0 {
		return YasnoSchedule{}, ErrProjectInvalidData
	}
	if _, found, err := s.projectCatalog.GetByID(ctx, projectID); err != nil {
		return YasnoSchedule{}, err
	} else if !found {
		return YasnoSchedule{}, ErrProjectNotFound
	}
	cfg, found, err := s.store.GetByProjectID(ctx, projectID)
	if err != nil {
		return YasnoSchedule{}, err
	}
	if !found {
		return YasnoSchedule{}, ErrYasnoNotConfigured
	}
	return s.scheduleForConfig(ctx, cfg)
}

func (s *YasnoScheduleService) previewBySelection(ctx context.Context, projectID int, in YasnoSelectionInput) (DTEKGroupConfig, YasnoSchedule, error) {
	if s == nil || s.projectCatalog == nil || s.store == nil || s.client == nil {
		return DTEKGroupConfig{}, YasnoSchedule{}, ErrYasnoUnavailable
	}
	if err := validateYasnoSelectionInput(in); err != nil {
		return DTEKGroupConfig{}, YasnoSchedule{}, err
	}

	groupKey, err := s.client.ResolveGroup(ctx, in.RegionID, in.StreetID, in.HouseID, in.DSOID)
	if err != nil {
		return DTEKGroupConfig{}, YasnoSchedule{}, err
	}
	cfg := DTEKGroupConfig{
		ProjectID:  projectID,
		RegionID:   in.RegionID,
		RegionName: strings.TrimSpace(in.RegionName),
		DSOID:      in.DSOID,
		DSOName:    strings.TrimSpace(in.DSOName),
		StreetID:   in.StreetID,
		StreetName: strings.TrimSpace(in.StreetName),
		HouseID:    in.HouseID,
		HouseName:  strings.TrimSpace(in.HouseName),
		GroupKey:   groupKey,
	}
	schedule, err := s.scheduleForConfig(ctx, cfg)
	if err != nil {
		return DTEKGroupConfig{}, YasnoSchedule{}, err
	}
	return cfg, schedule, nil
}

func (s *YasnoScheduleService) scheduleForConfig(ctx context.Context, cfg DTEKGroupConfig) (YasnoSchedule, error) {
	planned, err := s.client.GetPlannedOutages(ctx, cfg.RegionID, cfg.DSOID)
	if err != nil {
		return YasnoSchedule{}, err
	}
	groupSchedule, ok := planned[cfg.GroupKey]
	if !ok {
		return YasnoSchedule{}, ErrYasnoScheduleMissing
	}

	days := make([]YasnoScheduleDay, 0, 2)
	if groupSchedule.Today != nil {
		day, err := s.mapPlannedDay("today", "Сьогодні", groupSchedule.Today)
		if err != nil {
			return YasnoSchedule{}, err
		}
		days = append(days, day)
	}
	if groupSchedule.Tomorrow != nil {
		day, err := s.mapPlannedDay("tomorrow", "Завтра", groupSchedule.Tomorrow)
		if err != nil {
			return YasnoSchedule{}, err
		}
		days = append(days, day)
	}
	if len(days) == 0 {
		return YasnoSchedule{}, ErrYasnoScheduleMissing
	}

	var updatedAt *time.Time
	if parsed, ok := parseYasnoTimestamp(groupSchedule.UpdatedOn, s.location); ok {
		updatedAt = &parsed
	}

	return YasnoSchedule{
		GroupKey:  cfg.GroupKey,
		Address:   formatOutageAddress(cfg.StreetName, cfg.HouseName),
		UpdatedAt: updatedAt,
		Days:      days,
	}, nil
}

func (s *YasnoScheduleService) mapPlannedDay(key, label string, day *yasnoPlannedDayResponse) (YasnoScheduleDay, error) {
	if day == nil {
		return YasnoScheduleDay{}, ErrYasnoScheduleMissing
	}
	dayDate, err := parseYasnoDate(day.Date, s.location)
	if err != nil {
		return YasnoScheduleDay{}, fmt.Errorf("%w: invalid yasno day", ErrYasnoUnavailable)
	}
	slots := make([]YasnoScheduleSlot, 0, len(day.Slots))
	for _, slot := range day.Slots {
		if slot.Start < 0 || slot.End < 0 || slot.End < slot.Start || slot.End > 1440 {
			return YasnoScheduleDay{}, fmt.Errorf("%w: invalid yasno slot", ErrYasnoUnavailable)
		}
		slots = append(slots, YasnoScheduleSlot{
			StartMinute: slot.Start,
			EndMinute:   slot.End,
			Type:        strings.TrimSpace(slot.Type),
		})
	}
	return YasnoScheduleDay{
		Key:          key,
		Label:        label,
		WeekdayShort: weekdayShortUA(dayDate.Weekday()),
		Date:         dayDate.Format("2006-01-02"),
		Status:       strings.TrimSpace(day.Status),
		Slots:        slots,
	}, nil
}

func (s *YasnoScheduleService) requireProjectOwner(ctx context.Context, userID int64, projectID int) (Project, error) {
	if s == nil || s.projectCatalog == nil {
		return Project{}, ErrYasnoUnavailable
	}
	return s.projectCatalog.GetByIDForUser(ctx, projectID, userID)
}

func validateYasnoSelectionInput(in YasnoSelectionInput) error {
	if in.RegionID <= 0 || in.DSOID <= 0 || in.StreetID <= 0 || in.HouseID <= 0 {
		return ErrYasnoInvalidInput
	}
	if strings.TrimSpace(in.RegionName) == "" || strings.TrimSpace(in.DSOName) == "" {
		return ErrYasnoInvalidInput
	}
	if strings.TrimSpace(in.StreetName) == "" || strings.TrimSpace(in.HouseName) == "" {
		return ErrYasnoInvalidInput
	}
	return nil
}

func formatOutageAddress(streetName, houseName string) string {
	streetName = strings.TrimSpace(streetName)
	houseName = strings.TrimSpace(houseName)
	if streetName == "" {
		return houseName
	}
	if houseName == "" {
		return streetName
	}
	return fmt.Sprintf("%s, %s", streetName, houseName)
}

func weekdayShortUA(day time.Weekday) string {
	switch day {
	case time.Monday:
		return "Пн"
	case time.Tuesday:
		return "Вт"
	case time.Wednesday:
		return "Ср"
	case time.Thursday:
		return "Чт"
	case time.Friday:
		return "Пт"
	case time.Saturday:
		return "Сб"
	case time.Sunday:
		return "Нд"
	default:
		return ""
	}
}

func parseYasnoDate(raw string, loc *time.Location) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, errors.New("empty date")
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if layout == time.RFC3339 {
			if parsed, err := time.Parse(layout, raw); err == nil {
				return parsed.In(loc), nil
			}
			continue
		}
		if parsed, err := time.ParseInLocation(layout, raw, loc); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, errors.New("unsupported date")
}

func parseYasnoTimestamp(raw string, loc *time.Location) (time.Time, bool) {
	parsed, err := parseYasnoDate(raw, loc)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

func (c *YasnoClient) ListRegions(ctx context.Context) ([]yasnoRegionResponse, error) {
	var out []yasnoRegionResponse
	if err := c.getJSON(ctx, "/addresses/v2/regions", nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *YasnoClient) SearchStreets(ctx context.Context, regionID, dsoID int, query string) ([]yasnoStreetResponse, error) {
	var out []yasnoStreetResponse
	if err := c.getJSON(ctx, "/addresses/v2/streets", url.Values{
		"regionId": []string{strconv.Itoa(regionID)},
		"dsoId":    []string{strconv.Itoa(dsoID)},
		"query":    []string{query},
	}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *YasnoClient) SearchHouses(ctx context.Context, regionID, streetID, dsoID int, query string) ([]yasnoHouseResponse, error) {
	var out []yasnoHouseResponse
	if err := c.getJSON(ctx, "/addresses/v2/houses", url.Values{
		"regionId": []string{strconv.Itoa(regionID)},
		"streetId": []string{strconv.Itoa(streetID)},
		"dsoId":    []string{strconv.Itoa(dsoID)},
		"query":    []string{query},
	}, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *YasnoClient) ResolveGroup(ctx context.Context, regionID, streetID, houseID, dsoID int) (string, error) {
	var out yasnoGroupResponse
	if err := c.getJSON(ctx, "/addresses/v2/group", url.Values{
		"regionId": []string{strconv.Itoa(regionID)},
		"streetId": []string{strconv.Itoa(streetID)},
		"houseId":  []string{strconv.Itoa(houseID)},
		"dsoId":    []string{strconv.Itoa(dsoID)},
	}, &out); err != nil {
		return "", err
	}
	group, err := numberLikeToString(out.Group)
	if err != nil {
		return "", fmt.Errorf("%w: invalid yasno group", ErrYasnoUnavailable)
	}
	subgroup, err := numberLikeToString(out.Subgroup)
	if err != nil {
		return "", fmt.Errorf("%w: invalid yasno subgroup", ErrYasnoUnavailable)
	}
	return group + "." + subgroup, nil
}

func (c *YasnoClient) GetPlannedOutages(ctx context.Context, regionID, dsoID int) (map[string]yasnoPlannedGroupResponse, error) {
	var out map[string]yasnoPlannedGroupResponse
	endpoint := fmt.Sprintf("/regions/%d/dsos/%d/planned-outages", regionID, dsoID)
	if err := c.getJSON(ctx, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *YasnoClient) getJSON(ctx context.Context, endpoint string, query url.Values, out any) error {
	if c == nil || c.client == nil || c.baseURL == "" {
		return ErrYasnoUnavailable
	}
	fullURL := c.baseURL + endpoint
	if len(query) > 0 {
		fullURL += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("%w: create yasno request", ErrYasnoUnavailable)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: yasno request failed", ErrYasnoUnavailable)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return ErrYasnoLookupNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: yasno returned %d", ErrYasnoUnavailable, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: decode yasno response", ErrYasnoUnavailable)
	}
	return nil
}

func numberLikeToString(v any) (string, error) {
	switch value := v.(type) {
	case string:
		value = strings.TrimSpace(value)
		if value == "" {
			return "", errors.New("empty string")
		}
		return value, nil
	case float64:
		if value != float64(int64(value)) {
			return "", errors.New("not integer")
		}
		return strconv.FormatInt(int64(value), 10), nil
	case int:
		return strconv.Itoa(value), nil
	case int64:
		return strconv.FormatInt(value, 10), nil
	case json.Number:
		if _, err := value.Int64(); err != nil {
			return "", err
		}
		return value.String(), nil
	default:
		return "", errors.New("unsupported type")
	}
}
