package cleanup

import (
	"context"
	"fmt"

	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

// RunCleanAll performs a complete Docker cleanup.
func RunCleanAll(ctx context.Context, clients *dkr.Clients, p *ui.Printer, skipConfirm bool) error {
	p.Header("Complete Docker Cleanup")

	p.Error("WARNING: This will remove:")
	p.Println("  - All stopped containers")
	p.Println("  - All unused networks")
	p.Println("  - All unused images")
	p.Println("  - All build cache")
	p.Println("")
	p.Warning("Volumes will NOT be removed for safety")
	p.Println("")

	if !ui.ConfirmTypeFull("Are you ABSOLUTELY sure?", "yes", skipConfirm) {
		p.Info("Operation cancelled")
		return nil
	}

	p.Info("Removing stopped containers...")
	_, err := clients.Engine.ContainersPrune(ctx, filters.Args{})
	if err != nil {
		p.Error(fmt.Sprintf("pruning containers: %s", err))
	}

	p.Info("Removing unused networks...")
	_, err = clients.Engine.NetworksPrune(ctx, filters.Args{})
	if err != nil {
		p.Error(fmt.Sprintf("pruning networks: %s", err))
	}

	p.Info("Removing unused images...")
	_, err = clients.Engine.ImagesPrune(ctx, filters.NewArgs(
		filters.Arg("dangling", "false"),
	))
	if err != nil {
		p.Error(fmt.Sprintf("pruning images: %s", err))
	}

	p.Info("Removing build cache...")
	_, err = clients.Engine.BuildCachePrune(ctx, types.BuildCachePruneOptions{})
	if err != nil {
		p.Error(fmt.Sprintf("pruning build cache: %s", err))
	}

	p.Success("Complete cleanup finished")
	return nil
}
