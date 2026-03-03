package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	FirmwareJobStatusQueued    = "queued"
	FirmwareJobStatusRunning   = "running"
	FirmwareJobStatusSucceeded = "succeeded"
	FirmwareJobStatusFailed    = "failed"
	maxFirmwareErrorChars      = 24000
	headFirmwareErrorChars     = 12000
	tailFirmwareErrorChars     = 12000
)

var (
	ErrFirmwareDisabled     = errors.New("firmware build is disabled")
	ErrFirmwareInvalidInput = errors.New("invalid firmware build input")
	ErrFirmwareNotFound     = errors.New("firmware job not found")
	ErrFirmwareForbidden    = errors.New("forbidden")
	ErrFirmwareNotReady     = errors.New("firmware job is not ready")
	ErrFirmwareUnauthorized = errors.New("unauthorized")
	ansiEscapeRe            = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]`)
)

type FirmwareCompileInput struct {
	SSID          string
	Password      string
	PingURL       string
	ProjectSecret string
}

type FirmwareArtifact struct {
	Name   string
	Offset uint32
	Path   string
}

type FirmwareCompiler interface {
	Build(ctx context.Context, workDir string, in FirmwareCompileInput) ([]FirmwareArtifact, error)
}

type FirmwareBuildService struct {
	enabled      bool
	compiler     FirmwareCompiler
	workRoot     string
	buildTimeout time.Duration
	jobTTL       time.Duration

	mu   sync.RWMutex
	jobs map[string]*firmwareJob
}

type firmwareJob struct {
	ID        string
	Token     string
	ProjectID int
	Status    string
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
	WorkDir   string
	Artifacts map[string]string
	Parts     []FirmwarePart
}

type FirmwarePart struct {
	Name   string `json:"name"`
	Offset uint32 `json:"offset"`
}

type FirmwareJobSnapshot struct {
	ID            string         `json:"id"`
	ProjectID     int            `json:"projectId"`
	Status        string         `json:"status"`
	Error         string         `json:"error,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	ExpiresAt     time.Time      `json:"expiresAt"`
	ManifestToken string         `json:"manifestToken,omitempty"`
	Parts         []FirmwarePart `json:"parts,omitempty"`
}

type ArduinoFirmwareCompiler struct {
	CLIPath     string
	FQBN        string
	TemplateDir string
}

func NewFirmwareBuildService(
	compiler FirmwareCompiler,
	enabled bool,
	workRoot string,
	buildTimeout time.Duration,
	jobTTL time.Duration,
) *FirmwareBuildService {
	return &FirmwareBuildService{
		enabled:      enabled,
		compiler:     compiler,
		workRoot:     workRoot,
		buildTimeout: buildTimeout,
		jobTTL:       jobTTL,
		jobs:         make(map[string]*firmwareJob),
	}
}

func NewArduinoFirmwareCompiler(cliPath, fqbn, templateDir string) *ArduinoFirmwareCompiler {
	return &ArduinoFirmwareCompiler{
		CLIPath:     strings.TrimSpace(cliPath),
		FQBN:        strings.TrimSpace(fqbn),
		TemplateDir: strings.TrimSpace(templateDir),
	}
}

func (s *FirmwareBuildService) StartBuild(ctx context.Context, projectID int, in FirmwareCompileInput) (FirmwareJobSnapshot, error) {
	if s == nil || !s.enabled || s.compiler == nil {
		return FirmwareJobSnapshot{}, ErrFirmwareDisabled
	}
	if projectID <= 0 {
		return FirmwareJobSnapshot{}, ErrFirmwareInvalidInput
	}
	in.SSID = strings.TrimSpace(in.SSID)
	in.Password = strings.TrimSpace(in.Password)
	in.PingURL = strings.TrimSpace(in.PingURL)
	in.ProjectSecret = strings.TrimSpace(in.ProjectSecret)

	if err := validateCompileInput(in); err != nil {
		return FirmwareJobSnapshot{}, err
	}

	jobID, err := randomHex(10)
	if err != nil {
		return FirmwareJobSnapshot{}, fmt.Errorf("generate firmware job id: %w", err)
	}
	token, err := randomHex(24)
	if err != nil {
		return FirmwareJobSnapshot{}, fmt.Errorf("generate firmware job token: %w", err)
	}

	now := time.Now().UTC()
	job := &firmwareJob{
		ID:        jobID,
		Token:     token,
		ProjectID: projectID,
		Status:    FirmwareJobStatusQueued,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(s.jobTTL),
		WorkDir:   filepath.Join(s.workRoot, jobID),
		Artifacts: map[string]string{},
	}

	s.mu.Lock()
	s.cleanupExpiredLocked(now)
	s.jobs[job.ID] = job
	s.mu.Unlock()

	go s.runBuild(jobID, in)
	return snapshotFromJob(job), nil
}

func (s *FirmwareBuildService) GetJob(projectID int, jobID string) (FirmwareJobSnapshot, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(now)

	job, ok := s.jobs[jobID]
	if !ok || job.ProjectID != projectID {
		return FirmwareJobSnapshot{}, ErrFirmwareNotFound
	}
	return snapshotFromJob(job), nil
}

func (s *FirmwareBuildService) BuildManifestByToken(projectID int, jobID, token string) (map[string]any, error) {
	job, err := s.authorizeByToken(projectID, jobID, token)
	if err != nil {
		return nil, err
	}
	if job.Status != FirmwareJobStatusSucceeded {
		return nil, ErrFirmwareNotReady
	}

	parts := make([]map[string]any, 0, len(job.Parts))
	for _, part := range job.Parts {
		parts = append(parts, map[string]any{
			"path":   fmt.Sprintf("files/%s?token=%s", part.Name, token),
			"offset": part.Offset,
		})
	}

	manifest := map[string]any{
		"name":    "GridLogger ESP32-C3",
		"version": "1",
		"builds": []map[string]any{
			{
				"chipFamily": "ESP32-C3",
				"parts":      parts,
			},
		},
	}
	return manifest, nil
}

func (s *FirmwareBuildService) ArtifactPathByToken(projectID int, jobID, token, fileName string) (string, error) {
	job, err := s.authorizeByToken(projectID, jobID, token)
	if err != nil {
		return "", err
	}
	if job.Status != FirmwareJobStatusSucceeded {
		return "", ErrFirmwareNotReady
	}
	if strings.TrimSpace(fileName) == "" {
		return "", ErrFirmwareNotFound
	}
	path, ok := job.Artifacts[fileName]
	if !ok {
		return "", ErrFirmwareNotFound
	}
	return path, nil
}

func (s *FirmwareBuildService) authorizeByToken(projectID int, jobID, token string) (*firmwareJob, error) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cleanupExpiredLocked(now)

	job, ok := s.jobs[jobID]
	if !ok || job.ProjectID != projectID {
		return nil, ErrFirmwareNotFound
	}
	if subtle.ConstantTimeCompare([]byte(token), []byte(job.Token)) != 1 {
		return nil, ErrFirmwareUnauthorized
	}
	return job, nil
}

func (s *FirmwareBuildService) runBuild(jobID string, in FirmwareCompileInput) {
	ctx, cancel := context.WithTimeout(context.Background(), s.buildTimeout)
	defer cancel()

	s.updateJob(jobID, func(job *firmwareJob) {
		job.Status = FirmwareJobStatusRunning
		job.Error = ""
		job.UpdatedAt = time.Now().UTC()
	})

	artifacts, err := s.compiler.Build(ctx, filepath.Join(s.workRoot, jobID), in)
	if err != nil {
		s.updateJob(jobID, func(job *firmwareJob) {
			job.Status = FirmwareJobStatusFailed
			job.Error = trimError(err)
			job.UpdatedAt = time.Now().UTC()
		})
		return
	}

	s.updateJob(jobID, func(job *firmwareJob) {
		job.Status = FirmwareJobStatusSucceeded
		job.Error = ""
		job.UpdatedAt = time.Now().UTC()
		job.Parts = make([]FirmwarePart, 0, len(artifacts))
		job.Artifacts = make(map[string]string, len(artifacts))
		for _, artifact := range artifacts {
			job.Parts = append(job.Parts, FirmwarePart{Name: artifact.Name, Offset: artifact.Offset})
			job.Artifacts[artifact.Name] = artifact.Path
		}
		sort.Slice(job.Parts, func(i, j int) bool {
			return job.Parts[i].Offset < job.Parts[j].Offset
		})
	})
}

func (s *FirmwareBuildService) updateJob(jobID string, fn func(job *firmwareJob)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.jobs[jobID]
	if !ok {
		return
	}
	fn(job)
}

func (s *FirmwareBuildService) cleanupExpiredLocked(now time.Time) {
	for id, job := range s.jobs {
		if now.Before(job.ExpiresAt) {
			continue
		}
		_ = os.RemoveAll(job.WorkDir)
		delete(s.jobs, id)
	}
}

func snapshotFromJob(job *firmwareJob) FirmwareJobSnapshot {
	snapshot := FirmwareJobSnapshot{
		ID:            job.ID,
		ProjectID:     job.ProjectID,
		Status:        job.Status,
		Error:         job.Error,
		CreatedAt:     job.CreatedAt,
		UpdatedAt:     job.UpdatedAt,
		ExpiresAt:     job.ExpiresAt,
		ManifestToken: job.Token,
	}
	if len(job.Parts) > 0 {
		snapshot.Parts = append([]FirmwarePart(nil), job.Parts...)
	}
	return snapshot
}

func validateCompileInput(in FirmwareCompileInput) error {
	if err := validateWifiCreds(in.SSID, in.Password); err != nil {
		return err
	}
	if in.PingURL == "" || len(in.PingURL) > 512 || strings.ContainsAny(in.PingURL, "\n\r") {
		return ErrFirmwareInvalidInput
	}
	if in.ProjectSecret == "" || len(in.ProjectSecret) > 256 || strings.ContainsAny(in.ProjectSecret, "\n\r") {
		return ErrFirmwareInvalidInput
	}
	return nil
}

func validateWifiCreds(ssid, password string) error {
	if ssid == "" || len(ssid) > 64 {
		return ErrFirmwareInvalidInput
	}
	if password == "" || len(password) > 64 {
		return ErrFirmwareInvalidInput
	}
	if strings.ContainsAny(ssid, "\n\r") || strings.ContainsAny(password, "\n\r") {
		return ErrFirmwareInvalidInput
	}
	return nil
}

func randomHex(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func trimError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	msg = ansiEscapeRe.ReplaceAllString(msg, "")
	if len(msg) <= maxFirmwareErrorChars {
		return msg
	}
	return msg[:headFirmwareErrorChars] + "\n... [truncated] ...\n" + msg[len(msg)-tailFirmwareErrorChars:]
}

func (c *ArduinoFirmwareCompiler) Build(ctx context.Context, workDir string, in FirmwareCompileInput) ([]FirmwareArtifact, error) {
	if strings.TrimSpace(c.CLIPath) == "" || strings.TrimSpace(c.FQBN) == "" || strings.TrimSpace(c.TemplateDir) == "" {
		return nil, fmt.Errorf("arduino compiler is not configured")
	}

	sourceDir := filepath.Join(workDir, "source")
	buildDir := filepath.Join(workDir, "build")
	artifactDir := filepath.Join(workDir, "artifacts")

	if err := os.RemoveAll(workDir); err != nil {
		return nil, fmt.Errorf("cleanup work dir: %w", err)
	}
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		return nil, fmt.Errorf("create source dir: %w", err)
	}
	if err := os.MkdirAll(buildDir, 0o755); err != nil {
		return nil, fmt.Errorf("create build dir: %w", err)
	}
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return nil, fmt.Errorf("create artifact dir: %w", err)
	}

	if err := copyDir(c.TemplateDir, sourceDir); err != nil {
		return nil, fmt.Errorf("copy firmware template: %w", err)
	}
	if err := ensureSketchMainFile(sourceDir); err != nil {
		return nil, fmt.Errorf("normalize sketch main file: %w", err)
	}
	if err := writeGeneratedConfigHeader(filepath.Join(sourceDir, "generated_config.h"), in); err != nil {
		return nil, fmt.Errorf("write firmware config: %w", err)
	}

	cmd := exec.CommandContext(
		ctx,
		c.CLIPath,
		"compile",
		"--fqbn", c.FQBN,
		"--build-path", buildDir,
		"--output-dir", buildDir,
		sourceDir,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("arduino-cli compile failed: %w (%s)", err, string(output))
	}

	bootloaderPath, err := findSingle(buildDir, "*.ino.bootloader.bin")
	if err != nil {
		return nil, fmt.Errorf("locate bootloader bin: %w", err)
	}
	partitionsPath, err := findSingle(buildDir, "*.ino.partitions.bin")
	if err != nil {
		return nil, fmt.Errorf("locate partitions bin: %w", err)
	}
	appPath, err := findAppBin(buildDir)
	if err != nil {
		return nil, fmt.Errorf("locate app bin: %w", err)
	}

	artifacts := []FirmwareArtifact{}

	bootloaderOut := filepath.Join(artifactDir, "bootloader.bin")
	if err := copyFile(bootloaderPath, bootloaderOut); err != nil {
		return nil, fmt.Errorf("copy bootloader artifact: %w", err)
	}
	artifacts = append(artifacts, FirmwareArtifact{Name: "bootloader.bin", Offset: 0x0, Path: bootloaderOut})

	partitionsOut := filepath.Join(artifactDir, "partitions.bin")
	if err := copyFile(partitionsPath, partitionsOut); err != nil {
		return nil, fmt.Errorf("copy partitions artifact: %w", err)
	}
	artifacts = append(artifacts, FirmwareArtifact{Name: "partitions.bin", Offset: 0x8000, Path: partitionsOut})

	bootAppPath, _ := findSingle(buildDir, "boot_app0.bin")
	if bootAppPath == "" {
		bootAppPath, _ = findBootApp0FromArduinoData()
	}
	if bootAppPath != "" {
		bootAppOut := filepath.Join(artifactDir, "boot_app0.bin")
		if err := copyFile(bootAppPath, bootAppOut); err != nil {
			return nil, fmt.Errorf("copy boot_app0 artifact: %w", err)
		}
		artifacts = append(artifacts, FirmwareArtifact{Name: "boot_app0.bin", Offset: 0xE000, Path: bootAppOut})
	}

	appOut := filepath.Join(artifactDir, "app.bin")
	if err := copyFile(appPath, appOut); err != nil {
		return nil, fmt.Errorf("copy app artifact: %w", err)
	}
	artifacts = append(artifacts, FirmwareArtifact{Name: "app.bin", Offset: 0x10000, Path: appOut})

	sort.Slice(artifacts, func(i, j int) bool {
		return artifacts[i].Offset < artifacts[j].Offset
	})
	return artifacts, nil
}

func writeGeneratedConfigHeader(path string, in FirmwareCompileInput) error {
	content := fmt.Sprintf(`#pragma once
static const char GRID_WIFI_SSID[] = "%s";
static const char GRID_WIFI_PASSWORD[] = "%s";
static const char GRID_PING_URL[] = "%s";
static const char GRID_PROJECT_SECRET[] = "%s";
`, cEscape(in.SSID), cEscape(in.Password), cEscape(in.PingURL), cEscape(in.ProjectSecret))
	return os.WriteFile(path, []byte(content), 0o600)
}

func cEscape(raw string) string {
	r := strings.NewReplacer(
		"\\", "\\\\",
		"\"", "\\\"",
		"\n", "",
		"\r", "",
	)
	return r.Replace(raw)
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

func findSingle(dir, pattern string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", errors.New("not found")
	}
	sort.Strings(matches)
	return matches[0], nil
}

func findAppBin(dir string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.ino.bin"))
	if err != nil {
		return "", err
	}
	for _, item := range matches {
		base := filepath.Base(item)
		if strings.Contains(base, ".bootloader.") || strings.Contains(base, ".partitions.") || strings.Contains(base, ".merged.") {
			continue
		}
		return item, nil
	}
	return "", errors.New("not found")
}

func findBootApp0FromArduinoData() (string, error) {
	root := strings.TrimSpace(os.Getenv("ARDUINO_DIRECTORIES_DATA"))
	if root == "" {
		return "", errors.New("arduino data dir is not configured")
	}
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "boot_app0.bin" {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", errors.New("not found")
	}
	return found, nil
}

func ensureSketchMainFile(sourceDir string) error {
	expected := filepath.Join(sourceDir, filepath.Base(sourceDir)+".ino")
	matches, err := filepath.Glob(filepath.Join(sourceDir, "*.ino"))
	if err != nil {
		return err
	}
	if len(matches) == 0 {
		return errors.New("no .ino files found in sketch directory")
	}
	sort.Strings(matches)

	keep := expected
	if _, err := os.Stat(expected); err != nil {
		// Move the first .ino file to the expected sketch filename.
		if err := os.Rename(matches[0], expected); err != nil {
			return err
		}
	}

	for _, candidate := range matches {
		if filepath.Clean(candidate) == filepath.Clean(keep) {
			continue
		}
		if err := os.Remove(candidate); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}
