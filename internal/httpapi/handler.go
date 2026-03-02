package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
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
	// TODO: Keep disabled until explicit product/security decision to enforce.
	enforceProjectSecret = false
)

type Handler struct {
	svc                  *service.AvailabilityService
	projectCatalog       *service.ProjectCatalogService
	projectNotifications *service.ProjectNotificationService
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
	h.mux.HandleFunc("POST /api/settings/projects", h.handleSettingsCreateProject)
	h.mux.HandleFunc("GET /api/settings/projects/{projectId}", h.handleSettingsProject)
	h.mux.HandleFunc("POST /api/settings/projects/{projectId}", h.handleSettingsProjectUpdate)
	h.mux.HandleFunc("GET /api/projects/{projectId}/notifications/subscription", h.handleProjectNotificationSubscriptionGet)
	h.mux.HandleFunc("POST /api/projects/{projectId}/notifications/subscription", h.handleProjectNotificationSubscriptionPost)
	h.mux.HandleFunc("POST /api/projects/{projectId}/ping", h.handlePingRoute)
	h.mux.HandleFunc("GET /api/projects/{projectId}/ping", h.handlePingRoute)
	h.mux.HandleFunc("GET /api/projects/{projectId}/availability", h.handleAvailabilityRoute)

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

	h.warnOnProjectSecretMismatch(r.Context(), projectID, strings.TrimSpace(r.Header.Get(projectSecretHeader)))

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

	project, err := h.projectCatalog.CreateForUser(r.Context(), userID, in.Name, in.City, in.Slug)
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

	project, err := h.projectCatalog.UpdateForUser(r.Context(), userID, projectID, in.Name, in.City, in.Slug)
	if err != nil {
		h.writeProjectError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"project": project})
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
	Name string `json:"name"`
	City string `json:"city"`
	Slug string `json:"slug"`
}

type subscriptionInput struct {
	Subscribed *bool `json:"subscribed"`
}

func parseProjectInput(r *http.Request) (projectInput, error) {
	ct := strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type")))
	if strings.Contains(ct, "application/json") {
		var in projectInput
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return projectInput{}, errors.New("invalid json payload")
		}
		return in, nil
	}

	if err := r.ParseForm(); err != nil {
		return projectInput{}, errors.New("invalid form payload")
	}
	return projectInput{
		Name: r.PostForm.Get("name"),
		City: r.PostForm.Get("city"),
		Slug: r.PostForm.Get("slug"),
	}, nil
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

func (h *Handler) requireUserID(w http.ResponseWriter, r *http.Request) (int64, bool) {
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

func (h *Handler) warnOnProjectSecretMismatch(ctx context.Context, projectID int, providedSecret string) {
	if h.projectCatalog == nil || projectID <= 0 {
		return
	}

	project, found, err := h.projectCatalog.GetByID(ctx, projectID)
	if err != nil {
		log.Printf("warn ping secret: project lookup failed project_id=%d err=%v", projectID, err)
		return
	}
	if !found {
		return
	}

	headerPresent := providedSecret != ""
	matches := headerPresent && subtle.ConstantTimeCompare([]byte(providedSecret), []byte(project.Secret)) == 1
	if matches {
		return
	}

	log.Printf(
		"warn ping secret mismatch project_id=%d header_present=%t secret_match=%t enforcement_enabled=%t",
		projectID,
		headerPresent,
		matches,
		enforceProjectSecret,
	)
	if enforceProjectSecret {
		// TODO: If enforcement is enabled in the future, reject ping with 401/403 here.
	}
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
