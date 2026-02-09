package stack

import (
	"context"
	"fmt"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/compose/v2/pkg/api"
)

// RunUpdate pulls new images and recreates containers.
func RunUpdate(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer, service string) error {
	if service == "" {
		p.Header("Updating Stack Images")
	} else {
		p.Header(fmt.Sprintf("Updating %s", service))
	}

	project, err := dkr.LoadProject(ctx, cfg.ComposeFile, cfg.EnvFile)
	if err != nil {
		return err
	}

	if service != "" {
		project, err = project.WithSelectedServices([]string{service})
		if err != nil {
			return err
		}
	}

	err = clients.Compose.Pull(ctx, project, api.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling images: %w", err)
	}

	p.Success("Images updated successfully")
	p.Println("")
	p.Warning("Recreating containers with new images...")

	startOptions := api.StartOptions{}
	if service != "" {
		startOptions.Services = []string{service}
	}

	err = clients.Compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			Recreate: api.RecreateForce,
		},
		Start: startOptions,
	})
	if err != nil {
		return fmt.Errorf("recreating containers: %w", err)
	}

	if service == "" {
		p.Success("Stack updated successfully")
		return nil
	}

	p.Success(fmt.Sprintf("%s updated successfully", service))
	return nil
}
