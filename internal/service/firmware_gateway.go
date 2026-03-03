package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type FirmwareGatewayService struct {
	enabled        bool
	projectCatalog *ProjectCatalogService
	client         *FirmwareBuildClient
	pingBaseURL    string
}

type FirmwareBuildClient struct {
	baseURL   string
	authToken string
	client    *http.Client
}

type firmwareJobEnvelope struct {
	Job FirmwareJobSnapshot `json:"job"`
}

func NewFirmwareGatewayService(enabled bool, projectCatalog *ProjectCatalogService, client *FirmwareBuildClient, pingBaseURL string) *FirmwareGatewayService {
	return &FirmwareGatewayService{
		enabled:        enabled,
		projectCatalog: projectCatalog,
		client:         client,
		pingBaseURL:    strings.TrimRight(strings.TrimSpace(pingBaseURL), "/"),
	}
}

func NewFirmwareBuildClient(baseURL, authToken string, timeout time.Duration) *FirmwareBuildClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &FirmwareBuildClient{
		baseURL:   strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		authToken: strings.TrimSpace(authToken),
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (s *FirmwareGatewayService) StartBuildForUser(ctx context.Context, userID int64, projectID int, ssid, password string) (FirmwareJobSnapshot, error) {
	if s == nil || !s.enabled || s.client == nil || s.projectCatalog == nil || s.pingBaseURL == "" {
		return FirmwareJobSnapshot{}, ErrFirmwareDisabled
	}
	if userID <= 0 || projectID <= 0 {
		return FirmwareJobSnapshot{}, ErrFirmwareInvalidInput
	}
	ssid = strings.TrimSpace(ssid)
	password = strings.TrimSpace(password)
	if err := validateWifiCreds(ssid, password); err != nil {
		return FirmwareJobSnapshot{}, err
	}

	project, err := s.projectCatalog.GetByIDForUser(ctx, projectID, userID)
	if err != nil {
		return FirmwareJobSnapshot{}, err
	}
	if strings.TrimSpace(project.Secret) == "" {
		return FirmwareJobSnapshot{}, ErrFirmwareInvalidInput
	}

	return s.client.CreateJob(ctx, project.ID, FirmwareCompileInput{
		SSID:          ssid,
		Password:      password,
		PingURL:       fmt.Sprintf("%s/api/projects/%d/ping", s.pingBaseURL, project.ID),
		ProjectSecret: project.Secret,
	})
}

func (s *FirmwareGatewayService) GetJobForUser(ctx context.Context, userID int64, projectID int, jobID string) (FirmwareJobSnapshot, error) {
	if s == nil || !s.enabled || s.client == nil || s.projectCatalog == nil {
		return FirmwareJobSnapshot{}, ErrFirmwareDisabled
	}
	if userID <= 0 || projectID <= 0 || strings.TrimSpace(jobID) == "" {
		return FirmwareJobSnapshot{}, ErrFirmwareInvalidInput
	}
	if _, err := s.projectCatalog.GetByIDForUser(ctx, projectID, userID); err != nil {
		return FirmwareJobSnapshot{}, err
	}
	return s.client.GetJob(ctx, projectID, jobID)
}

func (s *FirmwareGatewayService) LoadManifestByToken(ctx context.Context, projectID int, jobID, token string) ([]byte, error) {
	if s == nil || !s.enabled || s.client == nil {
		return nil, ErrFirmwareDisabled
	}
	return s.client.GetManifest(ctx, projectID, jobID, token)
}

func (s *FirmwareGatewayService) OpenArtifactByToken(ctx context.Context, projectID int, jobID, token, fileName string) (io.ReadCloser, string, int64, error) {
	if s == nil || !s.enabled || s.client == nil {
		return nil, "", 0, ErrFirmwareDisabled
	}
	return s.client.DownloadArtifact(ctx, projectID, jobID, token, fileName)
}

func (c *FirmwareBuildClient) CreateJob(ctx context.Context, projectID int, in FirmwareCompileInput) (FirmwareJobSnapshot, error) {
	body, err := json.Marshal(map[string]any{
		"ssid":          in.SSID,
		"password":      in.Password,
		"pingUrl":       in.PingURL,
		"projectSecret": in.ProjectSecret,
	})
	if err != nil {
		return FirmwareJobSnapshot{}, fmt.Errorf("marshal firmware build request: %w", err)
	}

	endpoint := fmt.Sprintf("/v1/projects/%d/jobs", projectID)
	resp, err := c.do(ctx, http.MethodPost, endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return FirmwareJobSnapshot{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return FirmwareJobSnapshot{}, mapFirmwareClientError(resp.StatusCode, readBodyText(resp.Body))
	}

	var payload firmwareJobEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return FirmwareJobSnapshot{}, fmt.Errorf("decode firmware create response: %w", err)
	}
	return payload.Job, nil
}

func (c *FirmwareBuildClient) GetJob(ctx context.Context, projectID int, jobID string) (FirmwareJobSnapshot, error) {
	endpoint := fmt.Sprintf("/v1/projects/%d/jobs/%s", projectID, url.PathEscape(strings.TrimSpace(jobID)))
	resp, err := c.do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return FirmwareJobSnapshot{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return FirmwareJobSnapshot{}, mapFirmwareClientError(resp.StatusCode, readBodyText(resp.Body))
	}

	var payload firmwareJobEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return FirmwareJobSnapshot{}, fmt.Errorf("decode firmware status response: %w", err)
	}
	return payload.Job, nil
}

func (c *FirmwareBuildClient) GetManifest(ctx context.Context, projectID int, jobID, token string) ([]byte, error) {
	endpoint := fmt.Sprintf(
		"/v1/projects/%d/jobs/%s/manifest.json?token=%s",
		projectID,
		url.PathEscape(strings.TrimSpace(jobID)),
		url.QueryEscape(strings.TrimSpace(token)),
	)
	resp, err := c.do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, mapFirmwareClientError(resp.StatusCode, readBodyText(resp.Body))
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read firmware manifest response: %w", err)
	}
	return raw, nil
}

func (c *FirmwareBuildClient) DownloadArtifact(ctx context.Context, projectID int, jobID, token, fileName string) (io.ReadCloser, string, int64, error) {
	fileName = path.Base(strings.TrimSpace(fileName))
	if fileName == "." || fileName == "/" || fileName == "" {
		return nil, "", 0, ErrFirmwareInvalidInput
	}

	endpoint := fmt.Sprintf(
		"/v1/projects/%d/jobs/%s/files/%s?token=%s",
		projectID,
		url.PathEscape(strings.TrimSpace(jobID)),
		url.PathEscape(fileName),
		url.QueryEscape(strings.TrimSpace(token)),
	)
	resp, err := c.do(ctx, http.MethodGet, endpoint, "", nil)
	if err != nil {
		return nil, "", 0, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		return nil, "", 0, mapFirmwareClientError(resp.StatusCode, readBodyText(resp.Body))
	}
	return resp.Body, strings.TrimSpace(resp.Header.Get("Content-Type")), resp.ContentLength, nil
}

func (c *FirmwareBuildClient) do(ctx context.Context, method, endpoint, contentType string, body io.Reader) (*http.Response, error) {
	if c == nil || strings.TrimSpace(c.baseURL) == "" || c.client == nil {
		return nil, ErrFirmwareDisabled
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("create firmware request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.authToken)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("firmware service request failed: %w", err)
	}
	return resp, nil
}

func mapFirmwareClientError(statusCode int, message string) error {
	message = strings.TrimSpace(message)
	if message == "" {
		message = fmt.Sprintf("firmware service returned status %d", statusCode)
	}

	switch statusCode {
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrFirmwareInvalidInput, message)
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: %s", ErrFirmwareUnauthorized, message)
	case http.StatusForbidden:
		return fmt.Errorf("%w: %s", ErrFirmwareForbidden, message)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", ErrFirmwareNotFound, message)
	case http.StatusConflict:
		return fmt.Errorf("%w: %s", ErrFirmwareNotReady, message)
	case http.StatusServiceUnavailable:
		return fmt.Errorf("%w: %s", ErrFirmwareDisabled, message)
	default:
		return errorsJoinMessage(message)
	}
}

func readBodyText(r io.Reader) string {
	if r == nil {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(r, 64*1024))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func errorsJoinMessage(message string) error {
	return fmt.Errorf("firmware service error: %s", message)
}
