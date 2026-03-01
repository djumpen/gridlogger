package httpapi

import (
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

type Handler struct {
	svc                 *service.AvailabilityService
	telegramAuth        *service.TelegramAuthService
	sessionAuth         *service.SessionService
	defaultProjectID    int
	telegramBotUsername string
	sessionCookieName   string
	sessionCookieSecure bool
	sessionTTL          time.Duration
	mux                 *http.ServeMux
}

func NewHandler(
	svc *service.AvailabilityService,
	telegramAuth *service.TelegramAuthService,
	sessionAuth *service.SessionService,
	defaultProjectID int,
	telegramBotUsername string,
	sessionCookieName string,
	sessionCookieSecure bool,
	sessionTTL time.Duration,
) *Handler {
	h := &Handler{
		svc:                 svc,
		telegramAuth:        telegramAuth,
		sessionAuth:         sessionAuth,
		defaultProjectID:    defaultProjectID,
		telegramBotUsername: telegramBotUsername,
		sessionCookieName:   sessionCookieName,
		sessionCookieSecure: sessionCookieSecure,
		sessionTTL:          sessionTTL,
		mux:                 http.NewServeMux(),
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

	h.mux.HandleFunc("GET /auth/telegram/config", h.handleTelegramConfig)
	h.mux.HandleFunc("POST /auth/telegram/callback", h.handleTelegramCallback)
	h.mux.HandleFunc("GET /me", h.handleMe)
	h.mux.HandleFunc("POST /auth/logout", h.handleLogout)
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

	token, err := h.sessionAuth.IssueToken(account.TelegramID)
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
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	claims, err := h.sessionAuth.ParseToken(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	account, found, err := h.telegramAuth.GetAccountByID(r.Context(), claims.TelegramID)
	if err != nil {
		log.Printf("load user error: %v", err)
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	if !found {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
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
