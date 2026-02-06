package stack

import (
	"context"
	"fmt"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/compose/v2/pkg/api"
)

// RunStatus shows container status.
func RunStatus(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Stack Status")

	containers, err := clients.Compose.Ps(ctx, config.ProjectName, api.PsOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("getting status: %w", err)
	}

	if len(containers) == 0 {
		p.Warning("No containers found for project 'media-stack'")
		return nil
	}

	table := ui.NewTable(p.Out, "NAME", "IMAGE", "STATUS", "PORTS")
	for _, c := range containers {
		ports := ""
		for _, pub := range c.Publishers {
			if pub.PublishedPort > 0 {
				if ports != "" {
					ports += ", "
				}
				ports += fmt.Sprintf("%d->%d/%s", pub.PublishedPort, pub.TargetPort, pub.Protocol)
			}
		}
		table.Row(c.Name, c.Image, c.Status, ports)
	}
	table.Flush()

	return nil
}

// RunHealth shows health check status.
func RunHealth(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Health Status")

	containers, err := clients.Compose.Ps(ctx, config.ProjectName, api.PsOptions{
		All: true,
	})
	if err != nil {
		return fmt.Errorf("getting health: %w", err)
	}

	if len(containers) == 0 {
		p.Warning("No containers found for project 'media-stack'")
		return nil
	}

	table := ui.NewTable(p.Out, "NAME", "STATUS", "STATE")
	for _, c := range containers {
		table.Row(c.Name, c.Status, c.State)
	}
	table.Flush()

	return nil
}
