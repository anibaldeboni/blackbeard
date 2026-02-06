package stack

import (
	"context"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
)

// RunValidate validates the docker-compose configuration.
func RunValidate(ctx context.Context, cfg *config.Config, p *ui.Printer) error {
	p.Header("Validating Configuration")

	_, err := dkr.LoadProject(ctx, cfg.ComposeFile, cfg.EnvFile)
	if err != nil {
		p.Error("Docker Compose configuration has errors")
		return err
	}

	p.Success("Docker Compose configuration is valid")
	return nil
}
