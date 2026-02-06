package stack

import (
	"context"
	"fmt"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/compose/v2/pkg/api"
)

// RunUninstall removes containers and network.
func RunUninstall(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer, skipConfirm bool) error {
	p.Header("Uninstall Blackbeard Stack")

	p.Error("WARNING: This will remove:")
	p.Println("  - All running containers")
	p.Println(fmt.Sprintf("  - Docker network '%s'", cfg.NetworkName))
	p.Println("")
	p.Warning("Config directories and .env will NOT be removed")
	p.Println("")

	if !ui.ConfirmTypeFull("Are you sure?", "yes", skipConfirm) {
		p.Info("Uninstall cancelled")
		return nil
	}

	p.Info("Stopping containers...")
	_ = clients.Compose.Down(ctx, config.ProjectName, api.DownOptions{})

	p.Info("Removing network...")
	_ = clients.Engine.NetworkRemove(ctx, cfg.NetworkName)

	p.Success("Uninstall completed")
	p.Info("Config directories preserved in ./config/")
	p.Info("To fully remove, delete the project directory manually")

	return nil
}
