package service

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type firmwareFakeCompiler struct{}

func (c *firmwareFakeCompiler) Build(ctx context.Context, workDir string, in FirmwareCompileInput) ([]FirmwareArtifact, error) {
	artifactDir := filepath.Join(workDir, "artifacts")
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		return nil, err
	}
	mk := func(name string) (string, error) {
		path := filepath.Join(artifactDir, name)
		if err := os.WriteFile(path, []byte("ok"), 0o644); err != nil {
			return "", err
		}
		return path, nil
	}
	boot, err := mk("bootloader.bin")
	if err != nil {
		return nil, err
	}
	part, err := mk("partitions.bin")
	if err != nil {
		return nil, err
	}
	app, err := mk("app.bin")
	if err != nil {
		return nil, err
	}
	return []FirmwareArtifact{
		{Name: "bootloader.bin", Offset: 0x0, Path: boot},
		{Name: "partitions.bin", Offset: 0x8000, Path: part},
		{Name: "app.bin", Offset: 0x10000, Path: app},
	}, nil
}

func TestFirmwareBuildService_HappyPath(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	svc := NewFirmwareBuildService(
		&firmwareFakeCompiler{},
		true,
		root,
		3*time.Second,
		time.Hour,
	)

	job, err := svc.StartBuild(context.Background(), 11, FirmwareCompileInput{
		SSID:          "home_wifi",
		Password:      "pass12345",
		PingURL:       "https://svitlo.homes/api/projects/11/ping",
		ProjectSecret: "super-secret",
	})
	if err != nil {
		t.Fatalf("start build error: %v", err)
	}
	if job.ID == "" {
		t.Fatalf("job id is empty")
	}

	var final FirmwareJobSnapshot
	for i := 0; i < 30; i++ {
		current, err := svc.GetJob(11, job.ID)
		if err != nil {
			t.Fatalf("get job error: %v", err)
		}
		if current.Status == FirmwareJobStatusSucceeded {
			final = current
			break
		}
		time.Sleep(30 * time.Millisecond)
	}
	if final.Status != FirmwareJobStatusSucceeded {
		t.Fatalf("job did not finish, status=%s error=%s", final.Status, final.Error)
	}
	if final.ManifestToken == "" {
		t.Fatalf("manifest token is empty")
	}

	manifest, err := svc.BuildManifestByToken(11, job.ID, final.ManifestToken)
	if err != nil {
		t.Fatalf("manifest error: %v", err)
	}
	if manifest["name"] == "" {
		t.Fatalf("manifest name is empty")
	}

	if _, err := svc.ArtifactPathByToken(11, job.ID, final.ManifestToken, "app.bin"); err != nil {
		t.Fatalf("artifact lookup error: %v", err)
	}
}
