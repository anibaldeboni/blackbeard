package config

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/joho/godotenv"
)

const (
	NetworkName = "jollyroger"
	ProjectName = "media-stack"
)

// Config holds all configuration derived from .env and environment.
type Config struct {
	ProjectDir     string
	ComposeFile    string
	EnvFile        string
	EnvExample     string
	BackupDir      string
	NetworkName    string
	PUID           int
	PGID           int
	TZ             string
	DownloadsPath  string
	ConfigBasePath string
	GPUVideoGroup  string
	GPURenderGroup string
}

// Load reads the .env file and environment to populate Config.
func Load(projectDir string) (*Config, error) {
	envFile := filepath.Join(projectDir, ".env")
	envExample := filepath.Join(projectDir, ".env.example")

	// Load .env if it exists (ignore error if not present)
	_ = godotenv.Load(envFile)

	cfg := &Config{
		ProjectDir:     projectDir,
		ComposeFile:    filepath.Join(projectDir, "docker-compose.yml"),
		EnvFile:        envFile,
		EnvExample:     envExample,
		NetworkName:    NetworkName,
		PUID:           getEnvInt("PUID", os.Getuid()),
		PGID:           getEnvInt("PGID", os.Getgid()),
		TZ:             getEnv("TZ", "America/Sao_Paulo"),
		DownloadsPath:  getEnv("DOWNLOADS_PATH", "/media/STORAGE/downloads"),
		ConfigBasePath: getEnv("CONFIG_BASE_PATH", filepath.Join(projectDir, "config")),
		GPUVideoGroup:  getEnv("GPU_VIDEO_GROUP", "44"),
		GPURenderGroup: getEnv("GPU_RENDER_GROUP", "105"),
	}

	cfg.BackupDir = getEnv("BACKUP_DIR", filepath.Join(projectDir, "backups"))

	return cfg, nil
}

// EnvFileExists returns true if the .env file exists.
func (c *Config) EnvFileExists() bool {
	_, err := os.Stat(c.EnvFile)
	return err == nil
}

// EnvExampleExists returns true if the .env.example file exists.
func (c *Config) EnvExampleExists() bool {
	_, err := os.Stat(c.EnvExample)
	return err == nil
}


func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
