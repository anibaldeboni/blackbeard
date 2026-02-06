package stack

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
)

// EnsureConfigDirs discovers config directories from docker-compose.yml
// and creates any that are missing. Returns the number of dirs created and errors.
func EnsureConfigDirs(ctx context.Context, cfg *config.Config, p *ui.Printer) (total, created, errors int) {
	project, err := dkr.LoadProject(ctx, cfg.ComposeFile, cfg.EnvFile)
	if err != nil {
		p.Error(fmt.Sprintf("reading docker-compose.yml: %s", err))
		return 0, 0, 1
	}

	configDirs := dkr.ConfigDirsFromProject(project, cfg.ConfigBasePath)
	total = len(configDirs)

	for _, dir := range configDirs {
		fullPath := filepath.Join(cfg.ConfigBasePath, dir)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			if err := os.MkdirAll(fullPath, 0755); err != nil {
				p.Error(fmt.Sprintf("creating %s: %s", dir, err))
				errors++
			} else {
				p.Success(fmt.Sprintf("Created %s", dir))
				created++
			}
		} else {
			p.Info(fmt.Sprintf("  %s (exists)", dir))
		}
	}

	return total, created, errors
}

// RunDirs checks config directories discovered from docker-compose.yml
// and creates any that are missing.
func RunDirs(ctx context.Context, cfg *config.Config, p *ui.Printer) error {
	p.Header("Config Directories")
	p.Println("")

	total, created, errs := EnsureConfigDirs(ctx, cfg, p)
	if total == 0 && errs == 0 {
		p.Warning("No config directories found in docker-compose.yml")
		return nil
	}

	p.Println("")
	if created > 0 {
		p.Success(fmt.Sprintf("Created %d directory(ies)", created))
	}
	if errs > 0 {
		return fmt.Errorf("failed to create %d directory(ies)", errs)
	}
	if created == 0 && errs == 0 {
		p.Success(fmt.Sprintf("All %d directories already exist", total))
	}

	return nil
}
