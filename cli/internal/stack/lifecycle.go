package stack

import (
	"context"
	"fmt"
	"time"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/compose/v2/pkg/api"
	"github.com/docker/docker/api/types/network"
)

func ensureNetwork(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer) error {
	_, err := clients.Engine.NetworkInspect(ctx, cfg.NetworkName, network.InspectOptions{})
	if err != nil {
		p.Warning(fmt.Sprintf("Network '%s' not found. Creating...", cfg.NetworkName))
		_, err = clients.Engine.NetworkCreate(ctx, cfg.NetworkName, network.CreateOptions{})
		if err != nil {
			return fmt.Errorf("creating network: %w", err)
		}
		p.Success(fmt.Sprintf("Network '%s' created", cfg.NetworkName))
	}
	return nil
}

// RunStart starts the entire stack or a specific service.
func RunStart(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer, service string) error {
	if !cfg.EnvFileExists() {
		p.Warning(".env file not found!")
		if cfg.EnvExampleExists() {
			p.Error("Please create .env from .env.example first: flint stack install")
		} else {
			p.Error(".env.example not found!")
		}
		return fmt.Errorf(".env file required")
	}

	if err := ensureNetwork(ctx, cfg, clients, p); err != nil {
		return err
	}

	// Validate
	if err := RunValidate(ctx, cfg, p); err != nil {
		return err
	}

	if service == "" {
		p.Header("Starting Media Stack")
	} else {
		p.Header(fmt.Sprintf("Starting %s", service))
	}

	project, err := dkr.LoadProject(ctx, cfg.ComposeFile, cfg.EnvFile)
	if err != nil {
		return err
	}

	startOptions := api.StartOptions{}
	if service != "" {
		startOptions.Services = []string{service}
	}

	err = clients.Compose.Up(ctx, project, api.UpOptions{
		Create: api.CreateOptions{},
		Start:  startOptions,
	})
	if err != nil {
		return fmt.Errorf("starting stack: %w", err)
	}

	if service == "" {
		p.Success("Stack started successfully")
		p.Println("")
		p.Warning("Waiting for services to be healthy (this may take a few minutes)...")
		time.Sleep(10 * time.Second)

		return RunHealth(ctx, cfg, clients, p)
	}

	p.Success(fmt.Sprintf("%s started successfully", service))
	return nil
}

// RunStop stops the entire stack or a specific service.
func RunStop(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer, service string) error {
	if service == "" {
		p.Header("Stopping Media Stack")
		err := clients.Compose.Down(ctx, config.ProjectName, api.DownOptions{})
		if err != nil {
			return fmt.Errorf("stopping stack: %w", err)
		}

		p.Success("Stack stopped successfully")
		return nil
	}

	p.Header(fmt.Sprintf("Stopping %s", service))

	err := clients.Compose.Stop(ctx, config.ProjectName, api.StopOptions{
		Services: []string{service},
	})
	if err != nil {
		return fmt.Errorf("stopping %s: %w", service, err)
	}

	p.Success(fmt.Sprintf("%s stopped successfully", service))
	return nil
}

// RunRestart restarts the entire stack (stop then start).
func RunRestart(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer) error {
	if err := RunStop(ctx, cfg, clients, p, ""); err != nil {
		return err
	}
	p.Println("")
	return RunStart(ctx, cfg, clients, p, "")
}

// RunRestartService restarts a specific service.
func RunRestartService(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer, service string) error {
	p.Header(fmt.Sprintf("Restarting %s", service))

	err := clients.Compose.Restart(ctx, config.ProjectName, api.RestartOptions{
		Services: []string{service},
	})
	if err != nil {
		return fmt.Errorf("restarting %s: %w", service, err)
	}

	p.Success(fmt.Sprintf("%s restarted successfully", service))
	return nil
}
