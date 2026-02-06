package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
)

// RunCleanup removes backups older than keepDays.
func RunCleanup(_ context.Context, cfg *config.Config, p *ui.Printer, keepDays int) error {
	p.Header("Cleaning Old Backups")
	p.Warning(fmt.Sprintf("Removing backups older than %d days", keepDays))

	cutoff := time.Now().AddDate(0, 0, -keepDays)

	entries, err := os.ReadDir(cfg.BackupDir)
	if err != nil {
		p.Warning(fmt.Sprintf("No backups found in %s", cfg.BackupDir))
		return nil
	}

	removed := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			dirPath := filepath.Join(cfg.BackupDir, entry.Name())
			if err := os.RemoveAll(dirPath); err != nil {
				p.Error(fmt.Sprintf("removing %s: %s", entry.Name(), err))
			} else {
				p.Info(fmt.Sprintf("Removed: %s", entry.Name()))
				removed++
			}
		}
	}

	if removed == 0 {
		p.Info("No old backups to remove")
	} else {
		p.Success(fmt.Sprintf("Cleanup completed, removed %d backup(s)", removed))
	}

	return nil
}
