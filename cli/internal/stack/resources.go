package stack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/anibalnet/blackbeard/cli/internal/config"
	dkr "github.com/anibalnet/blackbeard/cli/internal/docker"
	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
)

// containerStats holds decoded Docker stats for a single container.
type containerStats struct {
	CPUPercent float64
	MemUsage   uint64
	MemLimit   uint64
	MemPercent float64
}

func decodeStats(body io.ReadCloser) (*containerStats, error) {
	defer body.Close()

	var stats struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
			OnlineCPUs     int    `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemCPUUsage uint64 `json:"system_cpu_usage"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage uint64 `json:"usage"`
			Limit uint64 `json:"limit"`
		} `json:"memory_stats"`
	}

	if err := json.NewDecoder(body).Decode(&stats); err != nil {
		return nil, err
	}

	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemCPUUsage - stats.PreCPUStats.SystemCPUUsage)

	cpuPercent := 0.0
	if systemDelta > 0 && cpuDelta > 0 {
		cpuPercent = (cpuDelta / systemDelta) * float64(stats.CPUStats.OnlineCPUs) * 100.0
	}

	memPercent := 0.0
	if stats.MemoryStats.Limit > 0 {
		memPercent = float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit) * 100.0
	}

	return &containerStats{
		CPUPercent: cpuPercent,
		MemUsage:   stats.MemoryStats.Usage,
		MemLimit:   stats.MemoryStats.Limit,
		MemPercent: memPercent,
	}, nil
}

func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// RunResources shows resource usage for stack containers.
func RunResources(ctx context.Context, cfg *config.Config, clients *dkr.Clients, p *ui.Printer) error {
	p.Header("Resource Usage")

	containers, err := clients.Engine.ContainerList(ctx, container.ListOptions{
		Filters: filters.NewArgs(
			filters.Arg("label", "com.docker.compose.project="+config.ProjectName),
		),
	})
	if err != nil {
		return fmt.Errorf("listing containers: %w", err)
	}

	if len(containers) == 0 {
		p.Warning("No running containers found")
		return nil
	}

	table := ui.NewTable(p.Out, "NAME", "CPU %", "MEM USAGE / LIMIT", "MEM %")

	for _, c := range containers {
		statsBody, err := clients.Engine.ContainerStats(ctx, c.ID, false)
		if err != nil {
			continue
		}

		stats, err := decodeStats(statsBody.Body)
		if err != nil {
			continue
		}

		name := ""
		if len(c.Names) > 0 {
			name = c.Names[0][1:] // Remove leading /
		}

		table.Row(
			name,
			fmt.Sprintf("%.2f%%", stats.CPUPercent),
			fmt.Sprintf("%s / %s", formatBytes(stats.MemUsage), formatBytes(stats.MemLimit)),
			fmt.Sprintf("%.2f%%", stats.MemPercent),
		)
	}

	table.Flush()
	return nil
}
