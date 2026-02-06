package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ResolveProjectDir determines the blackbeard project root.
// Priority: flagValue > BLACKBEARD_DIR env > walk up from cwd looking for docker-compose.yml.
func ResolveProjectDir(flagValue string) (string, error) {
	if flagValue != "" {
		abs, err := filepath.Abs(flagValue)
		if err != nil {
			return "", fmt.Errorf("resolving project dir: %w", err)
		}
		if _, err := os.Stat(filepath.Join(abs, "docker-compose.yml")); err != nil {
			return "", fmt.Errorf("docker-compose.yml not found in %s", abs)
		}
		return abs, nil
	}

	if envDir := os.Getenv("BLACKBEARD_DIR"); envDir != "" {
		abs, err := filepath.Abs(envDir)
		if err != nil {
			return "", fmt.Errorf("resolving BLACKBEARD_DIR: %w", err)
		}
		if _, err := os.Stat(filepath.Join(abs, "docker-compose.yml")); err != nil {
			return "", fmt.Errorf("docker-compose.yml not found in BLACKBEARD_DIR=%s", abs)
		}
		return abs, nil
	}

	// Walk up from cwd looking for docker-compose.yml
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "docker-compose.yml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("docker-compose.yml not found (use --project-dir or set BLACKBEARD_DIR)")
}
