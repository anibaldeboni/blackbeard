package cleanup

import (
	"context"
	"fmt"

	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types/container"
)

// RunProtected shows images currently in use by containers.
func RunProtected(ctx context.Context, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Protected Images (In Use by Containers)")

	containers, err := clients.Engine.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	if len(containers) == 0 {
		p.Info("No containers found")
		return nil
	}

	table := ui.NewTable(p.Out, "IMAGE", "CONTAINER", "STATUS")
	for _, c := range containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0][1:]
		}
		table.Row(c.Image, name, c.Status)
	}
	table.Flush()

	return nil
}
