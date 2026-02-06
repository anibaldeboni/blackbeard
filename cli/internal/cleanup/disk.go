package cleanup

import (
	"context"
	"fmt"

	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types"
)

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// RunDisk shows Docker disk usage.
func RunDisk(ctx context.Context, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Docker Disk Usage")

	du, err := clients.Engine.DiskUsage(ctx, types.DiskUsageOptions{})
	if err != nil {
		return fmt.Errorf("getting disk usage: %w", err)
	}

	// Images
	p.Println("")
	p.Info("Images:")
	table := ui.NewTable(p.Out, "REPOSITORY", "TAG", "SIZE")
	var totalImageSize int64
	for _, img := range du.Images {
		repo := "<none>"
		tag := "<none>"
		if len(img.RepoTags) > 0 {
			parts := splitRepoTag(img.RepoTags[0])
			repo = parts[0]
			tag = parts[1]
		}
		table.Row(repo, tag, formatBytes(img.Size))
		totalImageSize += img.Size
	}
	table.Flush()
	p.Printf("Total: %s (%d images)\n", formatBytes(totalImageSize), len(du.Images))

	// Containers
	p.Println("")
	p.Info("Containers:")
	table = ui.NewTable(p.Out, "NAME", "IMAGE", "SIZE (RW)")
	var totalContainerSize int64
	for _, c := range du.Containers {
		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0][1:]
		}
		table.Row(name, c.Image, formatBytes(c.SizeRw))
		totalContainerSize += c.SizeRw
	}
	table.Flush()
	p.Printf("Total: %s (%d containers)\n", formatBytes(totalContainerSize), len(du.Containers))

	// Volumes
	p.Println("")
	p.Info("Volumes:")
	table = ui.NewTable(p.Out, "NAME", "SIZE")
	var totalVolumeSize int64
	for _, v := range du.Volumes {
		table.Row(v.Name, formatBytes(v.UsageData.Size))
		totalVolumeSize += v.UsageData.Size
	}
	table.Flush()
	p.Printf("Total: %s (%d volumes)\n", formatBytes(totalVolumeSize), len(du.Volumes))

	// Build Cache
	p.Println("")
	p.Info("Build Cache:")
	var totalCacheSize int64
	for _, bc := range du.BuildCache {
		totalCacheSize += bc.Size
	}
	p.Printf("Total: %s (%d entries)\n", formatBytes(totalCacheSize), len(du.BuildCache))

	return nil
}

func splitRepoTag(repoTag string) [2]string {
	for i := len(repoTag) - 1; i >= 0; i-- {
		if repoTag[i] == ':' {
			return [2]string{repoTag[:i], repoTag[i+1:]}
		}
	}
	return [2]string{repoTag, "latest"}
}
