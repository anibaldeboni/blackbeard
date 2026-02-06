package backup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/volume"
)

// RunRestore restores a volume from a backup file.
func RunRestore(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer, backupFile, volumeName string, skipConfirm bool) error {
	if _, err := os.Stat(backupFile); os.IsNotExist(err) {
		p.Error(fmt.Sprintf("Backup file not found: %s", backupFile))
		return err
	}

	if volumeName == "" {
		base := filepath.Base(backupFile)
		volumeName = strings.TrimSuffix(base, ".tar.gz")
	}

	p.Warning(fmt.Sprintf("This will REPLACE all data in volume: %s", volumeName))
	if !ui.ConfirmYesNo("Are you sure?", skipConfirm) {
		p.Info("Restore cancelled")
		return nil
	}

	p.Info(fmt.Sprintf("Restoring volume: %s from %s", volumeName, backupFile))

	// Ensure volume exists
	clients.Engine.VolumeCreate(ctx, volume.CreateOptions{Name: volumeName})

	absFile, _ := filepath.Abs(backupFile)
	absDir := filepath.Dir(absFile)
	baseName := filepath.Base(absFile)

	if err := ensureAlpine(ctx, clients); err != nil {
		return err
	}

	resp, err := clients.Engine.ContainerCreate(ctx, &container.Config{
		Image: "alpine:latest",
		Cmd:   []string{"sh", "-c", fmt.Sprintf("rm -rf /data/* && tar xzf /backup/%s -C /data", baseName)},
	}, &container.HostConfig{
		Binds: []string{
			volumeName + ":/data",
			absDir + ":/backup:ro",
		},
	}, nil, nil, "")
	if err != nil {
		return fmt.Errorf("creating restore container: %w", err)
	}

	if err := clients.Engine.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		clients.Engine.ContainerRemove(ctx, resp.ID, container.RemoveOptions{})
		return fmt.Errorf("starting restore container: %w", err)
	}

	statusCh, errCh := clients.Engine.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("waiting for restore: %w", err)
		}
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("restore failed with exit code %d", status.StatusCode)
		}
	}

	clients.Engine.ContainerRemove(ctx, resp.ID, container.RemoveOptions{})

	p.Success(fmt.Sprintf("Restore completed: %s", volumeName))
	return nil
}
