package firmwareapi

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/djumpen/gridlogger/internal/service"
)

type Handler struct {
	builds    *service.FirmwareBuildService
	authToken string
	mux       *http.ServeMux
}

type createJobInput struct {
	SSID          string `json:"ssid"`
	Password      string `json:"password"`
	PingURL       string `json:"pingUrl"`
	ProjectSecret string `json:"projectSecret"`
}

func NewHandler(builds *service.FirmwareBuildService, authToken string) *Handler {
	h := &Handler{
		builds:    builds,
		authToken: strings.TrimSpace(authToken),
		mux:       http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	h.mux.HandleFunc("GET /healthz", h.handleHealth)
	h.mux.HandleFunc("GET /readyz", h.handleHealth)
	h.mux.HandleFunc("POST /v1/projects/{projectId}/jobs", h.handleCreateJob)
	h.mux.HandleFunc("GET /v1/projects/{projectId}/jobs/{jobId}", h.handleGetJob)
	h.mux.HandleFunc("GET /v1/projects/{projectId}/jobs/{jobId}/manifest.json", h.handleManifest)
	h.mux.HandleFunc("GET /v1/projects/{projectId}/jobs/{jobId}/files/{fileName}", h.handleFile)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
		return
	}
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var in createJobInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "invalid json payload", http.StatusBadRequest)
		return
	}

	job, err := h.builds.StartBuild(r.Context(), projectID, service.FirmwareCompileInput{
		SSID:          strings.TrimSpace(in.SSID),
		Password:      strings.TrimSpace(in.Password),
		PingURL:       strings.TrimSpace(in.PingURL),
		ProjectSecret: strings.TrimSpace(in.ProjectSecret),
	})
	if err != nil {
		h.writeFirmwareError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"job": job})
}

func (h *Handler) handleGetJob(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
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

	job, err := h.builds.GetJob(projectID, jobID)
	if err != nil {
		h.writeFirmwareError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job})
}

func (h *Handler) handleManifest(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
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

	manifest, err := h.builds.BuildManifestByToken(projectID, jobID, token)
	if err != nil {
		h.writeFirmwareError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, manifest)
}

func (h *Handler) handleFile(w http.ResponseWriter, r *http.Request) {
	if !h.authorize(w, r) {
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

	path, err := h.builds.ArtifactPathByToken(projectID, jobID, token, fileName)
	if err != nil {
		h.writeFirmwareError(w, err)
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path)
}

func (h *Handler) authorize(w http.ResponseWriter, r *http.Request) bool {
	if h.authToken == "" {
		return true
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	const prefix = "Bearer "
	if len(authHeader) <= len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	if strings.TrimSpace(authHeader[len(prefix):]) != h.authToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

func parseProjectID(raw string) (int, error) {
	projectID, err := strconv.Atoi(raw)
	if err != nil || projectID <= 0 {
		return 0, errors.New("invalid projectID")
	}
	return projectID, nil
}

func (h *Handler) writeFirmwareError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrFirmwareDisabled):
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	case errors.Is(err, service.ErrFirmwareInvalidInput):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, service.ErrFirmwareForbidden):
		http.Error(w, err.Error(), http.StatusForbidden)
	case errors.Is(err, service.ErrFirmwareUnauthorized):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, service.ErrFirmwareNotReady):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, service.ErrFirmwareNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		log.Printf("firmware api request error: %v", err)
		http.Error(w, "firmware operation failed", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
