package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
)

// RunListBackups lists available backups on disk.
func RunListBackups(_ context.Context, cfg *config.Config, p *ui.Printer) error {
	p.Header("Available Backups")

	entries, err := os.ReadDir(cfg.BackupDir)
	if err != nil {
		p.Warning(fmt.Sprintf("No backups found in %s", cfg.BackupDir))
		return nil
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		backupDir := filepath.Join(cfg.BackupDir, entry.Name())
		files, _ := filepath.Glob(filepath.Join(backupDir, "*.tar.gz"))
		if len(files) == 0 {
			continue
		}

		found = true
		// Calculate total size
		var totalSize int64
		for _, f := range files {
			if info, err := os.Stat(f); err == nil {
				totalSize += info.Size()
			}
		}

		p.Println("")
		p.Info(fmt.Sprintf("Backup: %s", entry.Name()))
		p.Println(fmt.Sprintf("  Size: %s", formatSize(totalSize)))
		p.Println(fmt.Sprintf("  Files: %d volumes", len(files)))
		p.Println(fmt.Sprintf("  Location: %s", backupDir))
	}

	if !found {
		p.Warning(fmt.Sprintf("No backups found in %s", cfg.BackupDir))
	}

	return nil
}
