package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerationDefaultsAndEnvironment(t *testing.T) {
	for _, name := range []string{
		"IOSU_PORT", "IOSU_JWT_KEY", "IOSU_ADMIN_PASSWORD", "IOSU_DATA_DIR", "IOSU_DB_PATH",
		"IOSU_GENERATION_PARALLEL_RUNS", "IOSU_GENERATION_TIMEOUT_SECONDS", "IOSU_GENERATION_MAX_OUTPUT_BYTES",
	} {
		t.Setenv(name, "")
		os.Unsetenv(name)
	}
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `jwt_key = "` + strings.Repeat("k", 32) + `"
data_directory = "/data"
db_path = "/db"
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Parse(path)
	if err != nil {
		t.Fatalf("parse defaults: %v", err)
	}
	if cfg.GenerationParallelRuns != 4 || cfg.GenerationTimeoutSeconds != 30 || cfg.GenerationMaxOutputBytes != 128<<10 {
		t.Fatalf("defaults: %+v", cfg)
	}
	t.Setenv("IOSU_GENERATION_PARALLEL_RUNS", "2")
	t.Setenv("IOSU_GENERATION_TIMEOUT_SECONDS", "60")
	t.Setenv("IOSU_GENERATION_MAX_OUTPUT_BYTES", "1048576")
	cfg, err = Parse(path)
	if err != nil {
		t.Fatalf("parse environment: %v", err)
	}
	if cfg.GenerationParallelRuns != 2 || cfg.GenerationTimeoutSeconds != 60 || cfg.GenerationMaxOutputBytes != 1048576 {
		t.Fatalf("environment: %+v", cfg)
	}
}

func TestGenerationConfigRejectsNegativeValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	body := `jwt_key = "` + strings.Repeat("k", 32) + `"
data_directory = "/data"
db_path = "/db"
generation_parallel_runs = -1
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Parse(path); err == nil {
		t.Fatal("negative generation setting was accepted")
	}
}
