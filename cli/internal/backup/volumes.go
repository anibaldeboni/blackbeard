package backup

import (
	"context"
	"fmt"

	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/volume"
)

// RunListVolumes lists volumes marked for backup.
func RunListVolumes(ctx context.Context, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Volumes Marked for Backup")

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

	table := ui.NewTable(p.Out, "NAME", "DRIVER", "MOUNTPOINT")
	for _, v := range volumes.Volumes {
		table.Row(v.Name, v.Driver, v.Mountpoint)
	}
	table.Flush()

	return nil
}
