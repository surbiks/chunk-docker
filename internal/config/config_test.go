package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAppliesDefaultsAndValidates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")

	content := `
server:
  work_dir: "/tmp/server"
  temp_dir: "/tmp/server/tmp"
  chunk_size: "32MiB"
  registry: "docker.io"
  repository: "example/chunks"
  release: "v1"
client:
  work_dir: "/tmp/client"
  temp_dir: "/tmp/client/tmp"
  output_dir: "/tmp/output"
`

	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.DockerBinary != "docker" {
		t.Fatalf("expected default docker binary, got %q", cfg.Server.DockerBinary)
	}
	if cfg.Server.ChunksPerImage != 1 {
		t.Fatalf("expected default chunks_per_image to be 1, got %d", cfg.Server.ChunksPerImage)
	}
	if cfg.Server.ImageChunkBaseDir != "/chunks" {
		t.Fatalf("expected default image chunk base dir, got %q", cfg.Server.ImageChunkBaseDir)
	}
	if cfg.Server.ManifestRegistry != "docker.io" {
		t.Fatalf("expected manifest registry to default to server registry, got %q", cfg.Server.ManifestRegistry)
	}
	if cfg.Client.Download.Parallelism != 1 {
		t.Fatalf("expected default parallelism 1, got %d", cfg.Client.Download.Parallelism)
	}
}

func TestValidateRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	cfg := &Root{
		Server: ServerConfig{
			ChunksPerImage:    0,
			ImageChunkBaseDir: "relative",
			Push:              Push{Retries: 0},
		},
		Client: ClientConfig{
			Download: Download{Parallelism: 0, Retries: 0},
		},
	}

	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected Validate to fail")
	}

	message := err.Error()
	for _, fragment := range []string{
		"server:",
		"work_dir is required",
		"chunk_size must be greater than zero",
		"release is required",
		"image_chunk_base_dir must be absolute",
		"client:",
		"output_dir is required",
	} {
		if !strings.Contains(message, fragment) {
			t.Fatalf("expected %q in validation error, got %q", fragment, message)
		}
	}
}
