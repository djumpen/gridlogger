package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/djumpen/gridlogger/internal/service"
)

type Handler struct {
	svc              *service.AvailabilityService
	defaultProjectID int
	mux              *http.ServeMux
}

func NewHandler(svc *service.AvailabilityService, defaultProjectID int) *Handler {
	h := &Handler{
		svc:              svc,
		defaultProjectID: defaultProjectID,
		mux:              http.NewServeMux(),
	}
	h.registerRoutes()
	return h
}

func (h *Handler) registerRoutes() {
	h.mux.HandleFunc("GET /healthz", h.handleHealth)
	h.mux.HandleFunc("GET /readyz", h.handleHealth)
	h.mux.HandleFunc("GET /api/default-project", h.handleDefaultProject)
	h.mux.HandleFunc("POST /api/projects/{projectId}/ping", h.handlePingRoute)
	h.mux.HandleFunc("GET /api/projects/{projectId}/ping", h.handlePingRoute)
	h.mux.HandleFunc("GET /api/projects/{projectId}/availability", h.handleAvailabilityRoute)
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *Handler) handleDefaultProject(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]int{"projectId": h.defaultProjectID})
}

func (h *Handler) handlePingRoute(w http.ResponseWriter, r *http.Request) {
	projectID, err := parseProjectID(r.PathValue("projectId"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"projectId":   projectID,
		"from":        from,
		"to":          to,
		"intervals":   intervals,
		"stats":       stats,
		"timezone":    "Europe/Kyiv",
		"sampleEvery": "30s",
	})
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
