package cleanup

import (
	"context"
	"fmt"

	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
)

// RunListImages lists images (dangling, all, and used by containers).
func RunListImages(ctx context.Context, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Unused Images")

	// Dangling images
	p.Println("Dangling Images (no tag):")
	dangling, err := clients.Engine.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("dangling", "true")),
	})
	if err != nil {
		return fmt.Errorf("listing dangling images: %w", err)
	}

	if len(dangling) == 0 {
		p.Println("  None found")
	} else {
		table := ui.NewTable(p.Out, "REPOSITORY", "TAG", "ID", "SIZE")
		for _, img := range dangling {
			repo := "<none>"
			tag := "<none>"
			id := img.ID
			if len(id) > 19 {
				id = id[:19]
			}
			table.Row(repo, tag, id, formatBytes(img.Size))
		}
		table.Flush()
	}

	// All images
	p.Println("")
	p.Println("All Images:")
	all, err := clients.Engine.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing images: %w", err)
	}

	table := ui.NewTable(p.Out, "REPOSITORY", "TAG", "ID", "SIZE")
	for _, img := range all {
		repo := "<none>"
		tag := "<none>"
		if len(img.RepoTags) > 0 {
			parts := splitRepoTag(img.RepoTags[0])
			repo = parts[0]
			tag = parts[1]
		}
		id := img.ID
		if len(id) > 19 {
			id = id[:19]
		}
		table.Row(repo, tag, id, formatBytes(img.Size))
	}
	table.Flush()

	// Used by containers
	p.Println("")
	p.Println("Used by containers:")
	containers, err := clients.Engine.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	seen := map[string]bool{}
	for _, c := range containers {
		if !seen[c.Image] {
			p.Println(fmt.Sprintf("  %s", c.Image))
			seen[c.Image] = true
		}
	}

	return nil
}

// RunDangling removes dangling images only.
func RunDangling(ctx context.Context, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Removing Dangling Images")

	dangling, err := clients.Engine.ImageList(ctx, image.ListOptions{
		Filters: filters.NewArgs(filters.Arg("dangling", "true")),
	})
	if err != nil {
		return fmt.Errorf("listing dangling images: %w", err)
	}

	if len(dangling) == 0 {
		p.Info("No dangling images found")
		return nil
	}

	p.Info(fmt.Sprintf("Found %d dangling images", len(dangling)))

	report, err := clients.Engine.ImagesPrune(ctx, filters.NewArgs(
		filters.Arg("dangling", "true"),
	))
	if err != nil {
		return fmt.Errorf("pruning dangling images: %w", err)
	}

	p.Success(fmt.Sprintf("Dangling images removed (reclaimed %s)", formatBytes(int64(report.SpaceReclaimed))))
	return nil
}

// RunPruneImages removes all unused images.
func RunPruneImages(ctx context.Context, clients *dkr.Clients, p *ui.Printer, skipConfirm bool) error {
	p.Header("Removing All Unused Images")

	p.Warning("This will remove ALL images not used by containers")
	if !ui.ConfirmYesNo("Are you sure?", skipConfirm) {
		p.Info("Operation cancelled")
		return nil
	}

	report, err := clients.Engine.ImagesPrune(ctx, filters.NewArgs(
		filters.Arg("dangling", "false"),
	))
	if err != nil {
		return fmt.Errorf("pruning images: %w", err)
	}

	p.Success(fmt.Sprintf("All unused images removed (reclaimed %s)", formatBytes(int64(report.SpaceReclaimed))))
	return nil
}

// RunPruneOld removes images older than the given number of days.
func RunPruneOld(ctx context.Context, clients *dkr.Clients, p *ui.Printer, days int, skipConfirm bool) error {
	p.Header(fmt.Sprintf("Removing Images Older Than %d Days", days))

	p.Warning(fmt.Sprintf("This will remove images created more than %d days ago", days))
	if !ui.ConfirmYesNo("Are you sure?", skipConfirm) {
		p.Info("Operation cancelled")
		return nil
	}

	hours := days * 24
	report, err := clients.Engine.ImagesPrune(ctx, filters.NewArgs(
		filters.Arg("dangling", "false"),
		filters.Arg("until", fmt.Sprintf("%dh", hours)),
	))
	if err != nil {
		return fmt.Errorf("pruning old images: %w", err)
	}

	p.Success(fmt.Sprintf("Old images removed (reclaimed %s)", formatBytes(int64(report.SpaceReclaimed))))
	return nil
}
