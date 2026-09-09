package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/BurntSushi/toml"
)

// AppName is used to derive the default config and data locations.
const AppName = "iosu"

// DefaultPath is where iosud and iosu look for the config file.
var DefaultPath = fmt.Sprintf("/etc/%s/config.toml", AppName)

type Config struct {
	ServerPort               string `toml:"server_port"`
	JWTKey                   string `toml:"jwt_key"`
	DefaultAdminPassword     string `toml:"default_admin_password"`
	DataDir                  string `toml:"data_directory"`
	DBPath                   string `toml:"db_path"`
	DevMode                  bool   `toml:"dev_mode"`
	GenerationParallelRuns   int    `toml:"generation_parallel_runs"`
	GenerationTimeoutSeconds int    `toml:"generation_timeout_seconds"`
	GenerationMaxOutputBytes int64  `toml:"generation_max_output_bytes"`
}

// Parse reads path, applies IOSU_* environment overrides and validates the
// result. Environment variables win over the file so secrets can be kept out
// of it entirely.
func Parse(path string) (*Config, error) {
	var c Config
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if _, err := toml.Decode(string(data), &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	env(&c.ServerPort, "IOSU_PORT")
	env(&c.JWTKey, "IOSU_JWT_KEY")
	env(&c.DefaultAdminPassword, "IOSU_ADMIN_PASSWORD")
	env(&c.DataDir, "IOSU_DATA_DIR")
	env(&c.DBPath, "IOSU_DB_PATH")
	if err := envInt(&c.GenerationParallelRuns, "IOSU_GENERATION_PARALLEL_RUNS"); err != nil {
		return nil, err
	}
	if err := envInt(&c.GenerationTimeoutSeconds, "IOSU_GENERATION_TIMEOUT_SECONDS"); err != nil {
		return nil, err
	}
	if err := envInt64(&c.GenerationMaxOutputBytes, "IOSU_GENERATION_MAX_OUTPUT_BYTES"); err != nil {
		return nil, err
	}

	if c.ServerPort == "" {
		c.ServerPort = "8990"
	}
	if c.DBPath == "" {
		return nil, errors.New("db_path is required")
	}
	if c.DataDir == "" {
		return nil, errors.New("data_directory is required")
	}
	if len(c.JWTKey) < 32 {
		return nil, errors.New("jwt_key must be at least 32 characters; generate one with: head -c 32 /dev/urandom | base64")
	}
	if c.GenerationParallelRuns < 0 || c.GenerationTimeoutSeconds < 0 || c.GenerationMaxOutputBytes < 0 {
		return nil, errors.New("generation limits cannot be negative")
	}
	if c.GenerationParallelRuns == 0 {
		c.GenerationParallelRuns = 4
	}
	if c.GenerationTimeoutSeconds == 0 {
		c.GenerationTimeoutSeconds = 30
	}
	if c.GenerationMaxOutputBytes == 0 {
		c.GenerationMaxOutputBytes = 128 << 10
	}
	return &c, nil
}

func env(dst *string, name string) {
	if v, ok := os.LookupEnv(name); ok {
		*dst = v
	}
}

func envInt(dst *int, name string) error {
	if v, ok := os.LookupEnv(name); ok {
		n, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("%s must be an integer", name)
		}
		*dst = n
	}
	return nil
}

func envInt64(dst *int64, name string) error {
	if v, ok := os.LookupEnv(name); ok {
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("%s must be an integer", name)
		}
		*dst = n
	}
	return nil
}
