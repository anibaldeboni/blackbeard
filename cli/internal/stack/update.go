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
func RunUpdate(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Updating Stack Images")

	project, err := dkr.LoadProject(ctx, cfg.ComposeFile, cfg.EnvFile)
	if err != nil {
		return err
	}

	err = clients.Compose.Pull(ctx, project, api.PullOptions{})
	if err != nil {
		return fmt.Errorf("pulling images: %w", err)
	}

	p.Success("Images updated successfully")
	p.Println("")
	p.Warning("Recreating containers with new images...")

	err = clients.Compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{
			Recreate: api.RecreateForce,
		},
		Start: api.StartOptions{},
	})
	if err != nil {
		return fmt.Errorf("recreating containers: %w", err)
	}

	p.Success("Stack updated successfully")
	return nil
}
