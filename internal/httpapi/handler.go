package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/djumpen/gridlogger/internal/service"
)

const (
	projectSecretHeader = "X-Project-Secret"
)

type Handler struct {
	svc                  *service.AvailabilityService
	projectCatalog       *service.ProjectCatalogService
	projectNotifications *service.ProjectNotificationService
	firmwareBuilds       *service.FirmwareGatewayService
	yasnoSchedules       *service.YasnoScheduleService
	telegramAuth         *service.TelegramAuthService
	sessionAuth          *service.SessionService
	defaultProjectID     int
	telegramBotUsername  string
	sessionCookieName    string
	sessionCookieSecure  bool
	sessionTTL           time.Duration
	mux                  *http.ServeMux
}

func NewHandler(
	svc *service.AvailabilityService,
	projectCatalog *service.ProjectCatalogService,
	projectNotifications *service.ProjectNotificationService,
	firmwareBuilds *service.FirmwareGatewayService,
	yasnoSchedules *service.YasnoScheduleService,
	telegramAuth *service.TelegramAuthService,
	sessionAuth *service.SessionService,
	defaultProjectID int,
	telegramBotUsername string,
	sessionCookieName string,
	sessionCookieSecure bool,
	sessionTTL time.Duration,
) *Handler {
	h := &Handler{
		svc:                  svc,
		projectCatalog:       projectCatalog,
		projectNotifications: projectNotifications,
		firmwareBuilds:       firmwareBuilds,
		yasnoSchedules:       yasnoSchedules,
		telegramAuth:         telegramAuth,
		sessionAuth:          sessionAuth,
		defaultProjectID:     defaultProjectID,
		telegramBotUsername:  telegramBotUsername,
		sessionCookieName:    sessionCookieName,
		sessionCookieSecure:  sessionCookieSecure,
		sessionTTL:           sessionTTL,
		mux:                  http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	h.mux.HandleFunc("GET /healthz", h.handleHealth)
	h.mux.HandleFunc("GET /readyz", h.handleHealth)
	h.mux.HandleFunc("GET /api/default-project", h.handleDefaultProject)
	h.mux.HandleFunc("GET /api/projects", h.handleProjectsList)
	h.mux.HandleFunc("GET /api/project-slugs/{slug}", h.handleProjectBySlug)
	h.mux.HandleFunc("GET /api/settings", h.handleSettings)
	h.mux.HandleFunc("GET /api/settings/subscriptions", h.handleSettingsSubscriptions)
	h.mux.HandleFunc("POST /api/settings/projects", h.handleSettingsCreateProject)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}", h.handleSettingsProject)
	h.mux.HandleFunc("POST /api/settings/projects/{projectId}", h.handleSettingsProjectUpdate)
	h.mux.HandleFunc("DELETE /api/settings/projects/{projectId}", h.handleSettingsProjectDelete)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}/yasno", h.handleSettingsProjectYasnoGet)
	h.mux.HandleFunc("POST /api/settings/projects/{projectId}/yasno", h.handleSettingsProjectYasnoSave)
	h.mux.HandleFunc("DELETE /api/settings/projects/{projectId}/yasno", h.handleSettingsProjectYasnoDelete)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}/yasno/regions", h.handleSettingsProjectYasnoRegions)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}/yasno/streets", h.handleSettingsProjectYasnoStreets)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}/yasno/houses", h.handleSettingsProjectYasnoHouses)
	h.mux.HandleFunc("POST /api/settings/projects/{projectId}/yasno/preview", h.handleSettingsProjectYasnoPreview)
	h.mux.HandleFunc("POST /api/settings/projects/{projectId}/firmware/jobs", h.handleFirmwareJobCreate)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}/firmware/jobs/{jobId}", h.handleFirmwareJobGet)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}/firmware/jobs/{jobId}/manifest.json", h.handleFirmwareManifest)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}/firmware/jobs/{jobId}/files/{fileName}", h.handleFirmwareFile)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}/telegram-bot/groups", h.handleTelegramBotGroupsList)
	h.mux.HandleFunc("POST /api/settings/projects/{projectId}/telegram-bot/groups", h.handleTelegramBotGroupsCreate)
	h.mux.HandleFunc("DELETE /api/settings/projects/{projectId}/telegram-bot/groups/{virtualUserId}", h.handleTelegramBotGroupsDelete)
	h.mux.HandleFunc("GET /api/projects/{projectId}/notifications/subscription", h.handleProjectNotificationSubscriptionGet)
	h.mux.HandleFunc("POST /api/projects/{projectId}/notifications/subscription", h.handleProjectNotificationSubscriptionPost)
	h.mux.HandleFunc("POST /api/projects/{projectId}/ping", h.handlePingRoute)
	h.mux.HandleFunc("GET /api/projects/{projectId}/ping", h.handlePingRoute)
	h.mux.HandleFunc("GET /api/projects/{projectId}/availability", h.handleAvailabilityRoute)
	h.mux.HandleFunc("GET /api/projects/{projectId}/yasno", h.handleProjectYasnoSchedule)

	h.mux.HandleFunc("GET /api/auth/telegram/config", h.handleTelegramConfig)
	h.mux.HandleFunc("POST /api/auth/telegram/callback", h.handleTelegramCallback)
	h.mux.HandleFunc("GET /api/me", h.handleMe)
	h.mux.HandleFunc("POST /api/auth/logout", h.handleLogout)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleDefaultProject(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]int{"projectId": h.defaultProjectID})
}

func (h *Handler) handleProjectsList(w http.ResponseWriter, r *http.Request) {
	if h.projectCatalog == nil {
		writeJSON(w, http.StatusOK, map[string]any{"projects": []service.Project{}})
		return
	}

	projects, err := h.projectCatalog.List(r.Context())
	if err != nil {
		log.Printf("list projects error: %v", err)
		http.Error(w, "failed to list projects", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handler) handleProjectBySlug(w http.ResponseWriter, r *http.Request) {
	if h.projectCatalog == nil {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	slug := strings.TrimSpace(r.PathValue("slug"))
	project, found, err := h.projectCatalog.GetBySlug(r.Context(), slug)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if !found {
		http.Error(w, "project not found", http.StatusNotFound)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"project": project})
}

func (h *Handler) handlePingRoute(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !h.validateProjectSecret(r.Context(), projectID, strings.TrimSpace(r.Header.Get(projectSecretHeader))) {
		http.Error(w, "invalid project secret", http.StatusUnauthorized)
		return
	}

	if err := h.svc.RecordPing(r.Context(), projectID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleAvailabilityRoute(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	from, to, err := parseWindow(r.URL.Query().Get("from"), r.URL.Query().Get("to"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	intervals, stats, err := h.svc.BuildIntervals(r.Context(), projectID, from, to)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"projectId":   projectID,
		"from":        from,
		"to":          to,
		"intervals":   intervals,
		"stats":       stats,
		"timezone":    "Europe/Kyiv",
		"sampleEvery": "30s",
	})
}

func (h *Handler) handleSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	projects, err := h.projectCatalog.ListByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("settings list projects error: user_id=%d err=%v", userID, err)
		http.Error(w, "failed to load projects", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handler) handleSettingsSubscriptions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.projectNotifications == nil {
		http.Error(w, "project notifications are not configured", http.StatusServiceUnavailable)
		return
	}

	projects, err := h.projectNotifications.ListActiveSubscribedProjectsByUserID(r.Context(), userID)
	if err != nil {
		log.Printf("settings list subscriptions error: user_id=%d err=%v", userID, err)
		http.Error(w, "failed to load subscriptions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (h *Handler) handleSettingsCreateProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}

	in, err := parseProjectInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project, err := h.projectCatalog.CreateForUser(r.Context(), userID, in.Name, in.City, in.Slug, in.IsPublic)
	if err != nil {
		h.writeProjectError(w, err)
		return
	}

	redirectTo := fmt.Sprintf("/a/settings/project/%d", project.ID)
	w.Header().Set("Location", redirectTo)
	writeJSON(w, http.StatusCreated, map[string]any{
		"project":    project,
		"redirectTo": redirectTo,
	})
}

func (h *Handler) handleSettingsProject(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project, err := h.projectCatalog.GetByIDForUser(r.Context(), projectID, userID)
	if err != nil {
		h.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": project})
}

func (h *Handler) handleSettingsProjectUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in, err := parseProjectInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	project, err := h.projectCatalog.UpdateForUser(r.Context(), userID, projectID, in.Name, in.City, in.Slug, in.IsPublic)
	if err != nil {
		h.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": project})
}

func (h *Handler) handleSettingsProjectDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.projectCatalog.DeleteForUser(r.Context(), userID, projectID); err != nil {
		h.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleSettingsProjectYasnoGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.yasnoSchedules == nil {
		http.Error(w, "yasno schedules are not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	config, schedule, scheduleError, err := h.yasnoSchedules.GetForUser(r.Context(), userID, projectID)
	if err != nil {
		h.writeYasnoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"config":        config,
		"schedule":      schedule,
		"scheduleError": scheduleError,
	})
}

func (h *Handler) handleSettingsProjectYasnoSave(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.yasnoSchedules == nil {
		http.Error(w, "yasno schedules are not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	in, err := parseYasnoSelectionInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	config, schedule, err := h.yasnoSchedules.SaveForUser(r.Context(), userID, projectID, in)
	if err != nil {
		h.writeYasnoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"config":   config,
		"schedule": schedule,
	})
}

func (h *Handler) handleSettingsProjectYasnoDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.yasnoSchedules == nil {
		http.Error(w, "yasno schedules are not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.yasnoSchedules.DeleteForUser(r.Context(), userID, projectID); err != nil {
		h.writeYasnoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleSettingsProjectYasnoRegions(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.yasnoSchedules == nil {
		http.Error(w, "yasno schedules are not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	regions, err := h.yasnoSchedules.ListRegions(r.Context(), userID, projectID)
	if err != nil {
		h.writeYasnoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"regions": regions})
}

func (h *Handler) handleSettingsProjectYasnoStreets(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.yasnoSchedules == nil {
		http.Error(w, "yasno schedules are not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	regionID, err := parsePositiveQueryInt(r, "regionId")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dsoID, err := parsePositiveQueryInt(r, "dsoId")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	items, err := h.yasnoSchedules.SearchStreets(r.Context(), userID, projectID, regionID, dsoID, query)
	if err != nil {
		h.writeYasnoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"streets": items})
}

func (h *Handler) handleSettingsProjectYasnoHouses(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.yasnoSchedules == nil {
		http.Error(w, "yasno schedules are not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	regionID, err := parsePositiveQueryInt(r, "regionId")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	streetID, err := parsePositiveQueryInt(r, "streetId")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	dsoID, err := parsePositiveQueryInt(r, "dsoId")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	items, err := h.yasnoSchedules.SearchHouses(r.Context(), userID, projectID, regionID, streetID, dsoID, query)
	if err != nil {
		h.writeYasnoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"houses": items})
}

func (h *Handler) handleSettingsProjectYasnoPreview(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.yasnoSchedules == nil {
		http.Error(w, "yasno schedules are not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	in, err := parseYasnoSelectionInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	config, schedule, err := h.yasnoSchedules.PreviewForUser(r.Context(), userID, projectID, in)
	if err != nil {
		h.writeYasnoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"config":   config,
		"schedule": schedule,
	})
}

func (h *Handler) handleProjectYasnoSchedule(w http.ResponseWriter, r *http.Request) {
	if h.yasnoSchedules == nil {
		http.Error(w, "yasno schedules are not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	schedule, err := h.yasnoSchedules.GetPublicSchedule(r.Context(), projectID)
	if err != nil {
		h.writeYasnoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"schedule": schedule})
}

func (h *Handler) handleFirmwareJobCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.firmwareBuilds == nil {
		http.Error(w, "firmware builder is not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in, err := parseFirmwareJobCreateInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	job, err := h.firmwareBuilds.StartBuildForUser(r.Context(), userID, projectID, in.SSID, in.Password)
	if err != nil {
		h.writeFirmwareError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job": mapJobResponse(projectID, job)})
}

func (h *Handler) handleFirmwareJobGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.firmwareBuilds == nil {
		http.Error(w, "firmware builder is not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	jobID := strings.TrimSpace(r.PathValue("jobId"))
	if jobID == "" {
		http.Error(w, "jobId is required", http.StatusBadRequest)
		return
	}

	job, err := h.firmwareBuilds.GetJobForUser(r.Context(), userID, projectID, jobID)
	if err != nil {
		h.writeFirmwareError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": mapJobResponse(projectID, job)})
}

func (h *Handler) handleFirmwareManifest(w http.ResponseWriter, r *http.Request) {
	if h.firmwareBuilds == nil {
		http.Error(w, "firmware builder is not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	jobID := strings.TrimSpace(r.PathValue("jobId"))
	if jobID == "" {
		http.Error(w, "jobId is required", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "token is required", http.StatusUnauthorized)
		return
	}

	manifestRaw, err := h.firmwareBuilds.LoadManifestByToken(r.Context(), projectID, jobID, token)
	if err != nil {
		h.writeFirmwareError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(manifestRaw)
}

func (h *Handler) handleFirmwareFile(w http.ResponseWriter, r *http.Request) {
	if h.firmwareBuilds == nil {
		http.Error(w, "firmware builder is not configured", http.StatusServiceUnavailable)
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	jobID := strings.TrimSpace(r.PathValue("jobId"))
	if jobID == "" {
		http.Error(w, "jobId is required", http.StatusBadRequest)
		return
	}
	fileName := strings.TrimSpace(r.PathValue("fileName"))
	if fileName == "" {
		http.Error(w, "fileName is required", http.StatusBadRequest)
		return
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		http.Error(w, "token is required", http.StatusUnauthorized)
		return
	}

	reader, contentType, _, err := h.firmwareBuilds.OpenArtifactByToken(r.Context(), projectID, jobID, token, fileName)
	if err != nil {
		h.writeFirmwareError(w, err)
		return
	}
	defer reader.Close()

	w.Header().Set("Cache-Control", "no-store")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	w.Header().Set("Content-Type", contentType)
	_, _ = io.Copy(w, reader)
}

func (h *Handler) handleProjectNotificationSubscriptionGet(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.projectNotifications == nil {
		http.Error(w, "project notifications are not configured", http.StatusServiceUnavailable)
		return
	}

	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	subscribed, err := h.projectNotifications.GetSubscriptionForUser(r.Context(), userID, projectID)
	if err != nil {
		h.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subscribed": subscribed,
	})
}

func (h *Handler) handleProjectNotificationSubscriptionPost(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.projectNotifications == nil {
		http.Error(w, "project notifications are not configured", http.StatusServiceUnavailable)
		return
	}

	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in, err := parseSubscriptionInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	subscribed, err := h.projectNotifications.SetSubscriptionForUser(r.Context(), userID, projectID, *in.Subscribed)
	if err != nil {
		h.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"subscribed": subscribed,
	})
}

func (h *Handler) handleTelegramBotGroupsList(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.projectNotifications == nil {
		http.Error(w, "project notifications are not configured", http.StatusServiceUnavailable)
		return
	}

	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	groups, err := h.projectNotifications.ListTelegramBotGroupsByProject(r.Context(), userID, projectID)
	if err != nil {
		h.writeTelegramBotGroupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"botUsername": h.telegramBotUsername,
		"groups":      groups,
	})
}

func (h *Handler) handleTelegramBotGroupsCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.projectNotifications == nil {
		http.Error(w, "project notifications are not configured", http.StatusServiceUnavailable)
		return
	}

	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in, err := parseTelegramBotGroupInput(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	group, err := h.projectNotifications.AddTelegramBotGroupToProject(r.Context(), userID, projectID, in.Title)
	if err != nil {
		h.writeTelegramBotGroupError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"group": group,
	})
}

func (h *Handler) handleTelegramBotGroupsDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireUserID(w, r)
	if !ok {
		return
	}
	if h.projectNotifications == nil {
		http.Error(w, "project notifications are not configured", http.StatusServiceUnavailable)
		return
	}

	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	virtualUserID, err := strconv.ParseInt(strings.TrimSpace(r.PathValue("virtualUserId")), 10, 64)
	if err != nil || virtualUserID <= 0 {
		http.Error(w, "invalid virtualUserId", http.StatusBadRequest)
		return
	}

	if err := h.projectNotifications.RemoveTelegramBotGroupFromProject(r.Context(), userID, projectID, virtualUserID); err != nil {
		h.writeTelegramBotGroupError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleTelegramConfig(w http.ResponseWriter, _ *http.Request) {
	enabled := h.authEnabled()
	resp := map[string]any{
		"enabled": enabled,
	}
	if enabled {
		resp["botUsername"] = h.telegramBotUsername
		resp["requestAccess"] = "write"
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) handleTelegramCallback(w http.ResponseWriter, r *http.Request) {
	setNoStoreHeaders(w)
	if !h.authEnabled() {
		http.Error(w, "telegram auth is disabled", http.StatusServiceUnavailable)
		return
	}

	fields, err := parseTelegramCallbackPayload(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	account, err := h.telegramAuth.Authenticate(r.Context(), fields)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrTelegramHashMismatch),
			errors.Is(err, service.ErrTelegramAuthStale),
			errors.Is(err, service.ErrTelegramAuthFuture):
			http.Error(w, err.Error(), http.StatusUnauthorized)
		case errors.Is(err, service.ErrTelegramReplay):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, service.ErrTelegramUserBlocked):
			http.Error(w, err.Error(), http.StatusForbidden)
		case errors.Is(err, service.ErrTelegramInvalidData):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("telegram auth unexpected error: %v", err)
			http.Error(w, "telegram auth failed", http.StatusInternalServerError)
		}
		return
	}

	token, err := h.sessionAuth.IssueToken(account.UserID)
	if err != nil {
		log.Printf("issue session error: %v", err)
		http.Error(w, "failed to issue session", http.StatusInternalServerError)
		return
	}

	h.setSessionCookie(w, token)
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"user":  account,
	})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	setNoStoreHeaders(w)
	if !h.authEnabled() {
		http.Error(w, "telegram auth is disabled", http.StatusServiceUnavailable)
		return
	}

	token := h.readSessionToken(r)
	if token == "" {
		http.Error(w, "неавторизовано", http.StatusUnauthorized)
		return
	}
	claims, err := h.sessionAuth.ParseToken(token)
	if err != nil {
		http.Error(w, "неавторизовано", http.StatusUnauthorized)
		return
	}

	account, found, err := h.telegramAuth.GetAccountByUserID(r.Context(), claims.UserID)
	if err != nil {
		log.Printf("load user error: %v", err)
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "неавторизовано", http.StatusUnauthorized)
		return
	}
	if account.IsBlocked {
		http.Error(w, "account is blocked", http.StatusForbidden)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": account})
}

func (h *Handler) handleLogout(w http.ResponseWriter, _ *http.Request) {
	setNoStoreHeaders(w)
	if !h.authEnabled() {
		http.Error(w, "telegram auth is disabled", http.StatusServiceUnavailable)
		return
	}

	h.clearSessionCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) authEnabled() bool {
	return h.telegramAuth != nil && h.sessionAuth != nil && h.telegramAuth.Enabled() && h.sessionAuth.Enabled() && h.telegramBotUsername != ""
}

func (h *Handler) setSessionCookie(w http.ResponseWriter, token string) {
	expiresAt := time.Now().UTC().Add(h.sessionTTL)
	cookie := &http.Cookie{
		Name:     h.sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int(h.sessionTTL.Seconds()),
	}
	http.SetCookie(w, cookie)
}

func (h *Handler) clearSessionCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     h.sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

func (h *Handler) readSessionToken(r *http.Request) string {
	if authHeader := strings.TrimSpace(r.Header.Get("Authorization")); authHeader != "" {
		if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
			return strings.TrimSpace(authHeader[7:])
		}
	}
	cookie, err := r.Cookie(h.sessionCookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func parseProjectID(raw string) (int, error) {
	projectID, err := strconv.Atoi(raw)
	if err != nil || projectID <= 0 {
		return 0, errors.New("invalid projectID")
	}
	return projectID, nil
}

func parseWindow(fromRaw, toRaw string) (time.Time, time.Time, error) {
	if fromRaw == "" || toRaw == "" {
		return time.Time{}, time.Time{}, errors.New("from and to query parameters are required")
	}
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid from")
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, errors.New("invalid to")
	}
	return from.UTC(), to.UTC(), nil
}

type projectInput struct {
	Name     string `json:"name"`
	City     string `json:"city"`
	Slug     string `json:"slug"`
	IsPublic bool   `json:"isPublic"`
}

type subscriptionInput struct {
	Subscribed *bool `json:"subscribed"`
}

type firmwareJobCreateInput struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

type telegramBotGroupInput struct {
	Title string `json:"title"`
}

type yasnoSelectionInput struct {
	RegionID   int    `json:"regionId"`
	RegionName string `json:"regionName"`
	DSOID      int    `json:"dsoId"`
	DSOName    string `json:"dsoName"`
	StreetID   int    `json:"streetId"`
	StreetName string `json:"streetName"`
	HouseID    int    `json:"houseId"`
	HouseName  string `json:"houseName"`
}

func parseProjectInput(r *http.Request) (projectInput, error) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(ct, "application/json") {
		in := projectInput{IsPublic: true}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return projectInput{}, errors.New("invalid json payload")
		}
		return in, nil
	}

	if err := r.ParseForm(); err != nil {
		return projectInput{}, errors.New("invalid form payload")
	}
	return projectInput{
		Name:     r.PostForm.Get("name"),
		City:     r.PostForm.Get("city"),
		Slug:     r.PostForm.Get("slug"),
		IsPublic: parseOptionalBoolForm(r.PostForm.Get("is_public"), true),
	}, nil
}

func parseOptionalBoolForm(raw string, fallback bool) bool {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseSubscriptionInput(r *http.Request) (subscriptionInput, error) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(ct, "application/json") {
		var in subscriptionInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return subscriptionInput{}, errors.New("invalid json payload")
		}
		if in.Subscribed == nil {
			return subscriptionInput{}, errors.New("subscribed is required")
		}
		return in, nil
	}

	if err := r.ParseForm(); err != nil {
		return subscriptionInput{}, errors.New("invalid form payload")
	}
	raw := strings.TrimSpace(r.PostForm.Get("subscribed"))
	if raw == "" {
		return subscriptionInput{}, errors.New("subscribed is required")
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return subscriptionInput{}, errors.New("subscribed must be true/false")
	}
	return subscriptionInput{Subscribed: &v}, nil
}

func parseFirmwareJobCreateInput(r *http.Request) (firmwareJobCreateInput, error) {
	var in firmwareJobCreateInput
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if !strings.Contains(ct, "application/json") {
		return firmwareJobCreateInput{}, errors.New("content type must be application/json")
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return firmwareJobCreateInput{}, errors.New("invalid json payload")
	}
	in.SSID = strings.TrimSpace(in.SSID)
	in.Password = strings.TrimSpace(in.Password)
	if in.SSID == "" || in.Password == "" {
		return firmwareJobCreateInput{}, errors.New("ssid and password are required")
	}
	return in, nil
}

func parseTelegramBotGroupInput(r *http.Request) (telegramBotGroupInput, error) {
	var in telegramBotGroupInput
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if !strings.Contains(ct, "application/json") {
		return telegramBotGroupInput{}, errors.New("content type must be application/json")
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return telegramBotGroupInput{}, errors.New("invalid json payload")
	}
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return telegramBotGroupInput{}, errors.New("title is required")
	}
	return in, nil
}

func parseYasnoSelectionInput(r *http.Request) (service.YasnoSelectionInput, error) {
	var in yasnoSelectionInput
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if !strings.Contains(ct, "application/json") {
		return service.YasnoSelectionInput{}, errors.New("content type must be application/json")
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		return service.YasnoSelectionInput{}, errors.New("invalid json payload")
	}
	out := service.YasnoSelectionInput{
		RegionID:   in.RegionID,
		RegionName: strings.TrimSpace(in.RegionName),
		DSOID:      in.DSOID,
		DSOName:    strings.TrimSpace(in.DSOName),
		StreetID:   in.StreetID,
		StreetName: strings.TrimSpace(in.StreetName),
		HouseID:    in.HouseID,
		HouseName:  strings.TrimSpace(in.HouseName),
	}
	if out.RegionID <= 0 || out.DSOID <= 0 || out.StreetID <= 0 || out.HouseID <= 0 {
		return service.YasnoSelectionInput{}, errors.New("regionId, dsoId, streetId, and houseId are required")
	}
	if out.RegionName == "" || out.DSOName == "" || out.StreetName == "" || out.HouseName == "" {
		return service.YasnoSelectionInput{}, errors.New("regionName, dsoName, streetName, and houseName are required")
	}
	return out, nil
}

func parsePositiveQueryInt(r *http.Request, key string) (int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid %s", key)
	}
	return value, nil
}

func (h *Handler) writeProjectError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProjectInvalidData), errors.Is(err, service.ErrProjectInvalidSlug):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, service.ErrProjectSlugTaken):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, service.ErrProjectNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, service.ErrProjectForbidden):
		http.Error(w, err.Error(), http.StatusForbidden)
	default:
		log.Printf("project request error: %v", err)
		http.Error(w, "project operation failed", http.StatusInternalServerError)
	}
}

func (h *Handler) writeFirmwareError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrFirmwareDisabled):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, service.ErrFirmwareInvalidInput), errors.Is(err, service.ErrProjectInvalidData):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, service.ErrFirmwareForbidden), errors.Is(err, service.ErrProjectForbidden):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, service.ErrFirmwareUnauthorized):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, service.ErrFirmwareNotReady):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, service.ErrFirmwareNotFound), errors.Is(err, service.ErrProjectNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		log.Printf("firmware request error: %v", err)
		http.Error(w, "firmware operation failed", http.StatusInternalServerError)
	}
}

func (h *Handler) writeTelegramBotGroupError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProjectInvalidData):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, service.ErrProjectForbidden):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, service.ErrProjectNotFound), errors.Is(err, service.ErrTelegramBotGroupMissing), errors.Is(err, service.ErrTelegramBotGroupNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, service.ErrTelegramBotDisabled):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, service.ErrTelegramBotGroupAmbiguous):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, service.ErrTelegramBotGroupConflict):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		log.Printf("telegram bot group request error: %v", err)
		http.Error(w, "telegram bot group operation failed", http.StatusInternalServerError)
	}
}

func (h *Handler) writeYasnoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrProjectInvalidData), errors.Is(err, service.ErrYasnoInvalidInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, service.ErrProjectForbidden):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, service.ErrProjectNotFound), errors.Is(err, service.ErrYasnoLookupNotFound), errors.Is(err, service.ErrYasnoNotConfigured), errors.Is(err, service.ErrYasnoScheduleMissing):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, service.ErrYasnoUnavailable):
		http.Error(w, err.Error(), http.StatusBadGateway)
	default:
		log.Printf("yasno request error: %v", err)
		http.Error(w, "yasno operation failed", http.StatusInternalServerError)
	}
}

func mapJobResponse(projectID int, job service.FirmwareJobSnapshot) map[string]any {
	resp := map[string]any{
		"id":        job.ID,
		"projectId": job.ProjectID,
		"status":    job.Status,
		"error":     job.Error,
		"createdAt": job.CreatedAt,
		"updatedAt": job.UpdatedAt,
		"expiresAt": job.ExpiresAt,
		"parts":     job.Parts,
	}
	if job.Status == service.FirmwareJobStatusSucceeded && job.ManifestToken != "" {
		resp["manifestUrl"] = fmt.Sprintf(
			"/api/settings/projects/%d/firmware/jobs/%s/manifest.json?token=%s",
			projectID,
			job.ID,
			job.ManifestToken,
		)
	}
	return resp
}

func (h *Handler) requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	setNoStoreHeaders(w)
	if !h.authEnabled() {
		http.Error(w, "telegram auth is disabled", http.StatusServiceUnavailable)
		return 0, false
	}

	token := h.readSessionToken(r)
	if token == "" {
		http.Error(w, "неавторизовано", http.StatusUnauthorized)
		return 0, false
	}
	claims, err := h.sessionAuth.ParseToken(token)
	if err != nil {
		http.Error(w, "неавторизовано", http.StatusUnauthorized)
		return 0, false
	}

	account, found, err := h.telegramAuth.GetAccountByUserID(r.Context(), claims.UserID)
	if err != nil {
		log.Printf("load user error: %v", err)
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return 0, false
	}
	if !found {
		http.Error(w, "неавторизовано", http.StatusUnauthorized)
		return 0, false
	}
	if account.IsBlocked {
		http.Error(w, "account is blocked", http.StatusForbidden)
		return 0, false
	}
	return claims.UserID, true
}

func (h *Handler) validateProjectSecret(ctx context.Context, projectID int, providedSecret string) bool {
	if h.projectCatalog == nil || projectID <= 0 {
		log.Printf("reject ping secret: project catalog is not configured project_id=%d", projectID)
		return false
	}

	project, found, err := h.projectCatalog.GetByID(ctx, projectID)
	if err != nil {
		log.Printf("reject ping secret: project lookup failed project_id=%d err=%v", projectID, err)
		return false
	}
	if !found {
		log.Printf("reject ping secret: project not found project_id=%d", projectID)
		return false
	}

	headerPresent := providedSecret != ""
	matches := headerPresent && subtle.ConstantTimeCompare([]byte(providedSecret), []byte(project.Secret)) == 1
	if matches {
		return true
	}

	log.Printf(
		"reject ping secret mismatch project_id=%d header_present=%t secret_match=%t",
		projectID,
		headerPresent,
		matches,
	)
	return false
}

func parseTelegramCallbackPayload(r *http.Request) (map[string]string, error) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(ct, "application/json") {
		dec := json.NewDecoder(r.Body)
		dec.UseNumber()
		var raw map[string]any
		if err := dec.Decode(&raw); err != nil {
			return nil, errors.New("invalid telegram json payload")
		}
		return normalizeTelegramFields(raw)
	}

	if err := r.ParseForm(); err != nil {
		return nil, errors.New("invalid telegram form payload")
	}

	out := make(map[string]string, len(r.PostForm))
	for key, values := range r.PostForm {
		if len(values) == 0 {
			out[key] = ""
			continue
		}
		out[key] = values[0]
	}
	if len(out) == 0 {
		return nil, errors.New("telegram payload is empty")
	}
	return out, nil
}

func normalizeTelegramFields(raw map[string]any) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, errors.New("telegram payload is empty")
	}

	out := make(map[string]string, len(raw))
	for key, value := range raw {
		str, err := stringifyTelegramField(value)
		if err != nil {
			return nil, fmt.Errorf("invalid telegram payload field %q", key)
		}
		out[key] = str
	}
	return out, nil
}

func stringifyTelegramField(v any) (string, error) {
	switch t := v.(type) {
	case nil:
		return "", nil
	case string:
		return t, nil
	case json.Number:
		if _, err := t.Int64(); err != nil {
			return "", err
		}
		return t.String(), nil
	case float64:
		if t != math.Trunc(t) {
			return "", errors.New("non-integer number")
		}
		return strconv.FormatInt(int64(t), 10), nil
	case bool:
		return strconv.FormatBool(t), nil
	default:
		raw, err := json.Marshal(t)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func setNoStoreHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("Vary", "Cookie, Authorization")
}
