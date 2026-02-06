package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/volume"
)

func ensureAlpine(ctx context.Context, clients *dkr.Clients) error {
	reader, err := clients.Engine.ImagePull(ctx, "alpine:latest", image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling alpine: %w", err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader) // Must drain reader
	return nil
}

func backupVolume(ctx context.Context, clients *dkr.Clients, p *ui.Printer, volumeName, backupPath string) error {
	p.Info(fmt.Sprintf("Backing up volume: %s", volumeName))

	absBackupPath, err := filepath.Abs(backupPath)
	if err != nil {
		return err
	}

	resp, err := clients.Engine.ContainerCreate(ctx, &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"tar", "czf", fmt.Sprintf("/backup/%s.tar.gz", volumeName), "-C", "/data", "."},
	}, &container.HostConfig{
		Binds: []string{
			volumeName + ":/data:ro",
			absBackupPath + ":/backup",
		},
	}, nil, nil, "")
	if err != nil {
		return fmt.Errorf("creating backup container: %w", err)
	}

	if err := clients.Engine.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		clients.Engine.ContainerRemove(ctx, resp.ID, container.RemoveOptions{})
		return fmt.Errorf("starting backup container: %w", err)
	}

	statusCh, errCh := clients.Engine.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("waiting for backup: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("backup failed with exit code %d", status.StatusCode)
		}
	}

	clients.Engine.ContainerRemove(ctx, resp.ID, container.RemoveOptions{})

	backupFile := filepath.Join(absBackupPath, volumeName+".tar.gz")
	info, err := os.Stat(backupFile)
	if err == nil {
		p.Success(fmt.Sprintf("Backup completed: %s.tar.gz (%s)", volumeName, formatSize(info.Size())))
	} else {
		p.Success(fmt.Sprintf("Backup completed: %s.tar.gz", volumeName))
	}

	return nil
}

// RunBackupAll backs up all volumes with the backup.enable=true label.
func RunBackupAll(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer) error {
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(cfg.BackupDir, timestamp)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("creating backup dir: %w", err)
	}

	p.Header("Starting Backup Process")

	volumes, err := clients.Engine.VolumeList(ctx, volume.ListOptions{
		Filters: filters.NewArgs(filters.Arg("label", "backup.enable=true")),
	})
	if err != nil {
		return fmt.Errorf("listing volumes: %w", err)
	}

	if len(volumes.Volumes) == 0 {
		p.Warning("No volumes found with label 'backup.enable=true'")
		return nil
	}

	total := len(volumes.Volumes)
	p.Info(fmt.Sprintf("Found %d volumes to backup", total))
	p.Info(fmt.Sprintf("Backup destination: %s", backupPath))

	if err := ensureAlpine(ctx, clients); err != nil {
		return err
	}

	for i, v := range volumes.Volumes {
		p.Println("")
		p.Info(fmt.Sprintf("[%d/%d] Processing %s", i+1, total, v.Name))
		if err := backupVolume(ctx, clients, p, v.Name, backupPath); err != nil {
			p.Error(fmt.Sprintf("Backup failed: %s - %s", v.Name, err))
		}
	}

	p.Println("")
	p.Header("Backup Summary")
	p.Success("Backup completed successfully")
	p.Info(fmt.Sprintf("Location: %s", backupPath))

	return nil
}

// RunBackupVolume backs up a specific volume.
func RunBackupVolume(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer, volumeName string) error {
	timestamp := time.Now().Format("20060102_150405")
	backupPath := filepath.Join(cfg.BackupDir, timestamp)

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("creating backup dir: %w", err)
	}

	if err := ensureAlpine(ctx, clients); err != nil {
		return err
	}

	return backupVolume(ctx, clients, p, volumeName, backupPath)
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%dB", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
