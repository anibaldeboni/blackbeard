package hw

import (
	"fmt"
	"time"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
)

// RunInfo shows system info, uptime, and load averages.
func RunInfo(p *ui.Printer) error {
	p.Header("System Info")

	info, err := host.Info()
	if err != nil {
		return fmt.Errorf("reading host info: %w", err)
	}

	p.Printf("Hostname:  %s\n", info.Hostname)
	p.Printf("OS:        %s %s\n", info.Platform, info.PlatformVersion)
	p.Printf("Kernel:    %s %s\n", info.OS, info.KernelVersion)
	p.Printf("Arch:      %s\n", info.KernelArch)
	p.Printf("Uptime:    %s\n", formatUptime(info.Uptime))

	avg, err := load.Avg()
	if err == nil {
		p.Printf("Load Avg:  %.2f  %.2f  %.2f  (1m / 5m / 15m)\n",
			avg.Load1, avg.Load5, avg.Load15)
	}

	return nil
}

func formatUptime(seconds uint64) string {
	d := time.Duration(seconds) * time.Second
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}
