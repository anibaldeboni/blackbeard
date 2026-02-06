package hw

import (
	"fmt"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/fatih/color"
	"github.com/shirou/gopsutil/v4/disk"
)

// DefaultDiskPaths are checked when no args are given.
var DefaultDiskPaths = []string{"/", "/media/STORAGE"}

// RunDisk shows disk usage for the given paths (or defaults).
func RunDisk(p *ui.Printer, paths []string) error {
	p.Header("Disk Usage")

	if len(paths) == 0 {
		paths = DefaultDiskPaths
	}

	table := ui.NewTable(p.Out, "MOUNT", "USED", "TOTAL", "AVAIL", "USE%")

	for _, path := range paths {
		usage, err := disk.Usage(path)
		if err != nil {
			p.Warning(fmt.Sprintf("%s: %s", path, err))
			continue
		}
		table.Row(
			path,
			formatBytesHW(usage.Used),
			formatBytesHW(usage.Total),
			formatBytesHW(usage.Free),
			getDiskPctString(usage.UsedPercent),
		)
	}
	table.Flush()
	return nil
}

func getDiskPctString(pct float64) string {
	s := fmt.Sprintf("%.1f%%", pct)
	switch {
	case pct >= 90:
		return color.New(color.FgRed, color.Bold).Sprint(s)
	case pct >= 80:
		return color.New(color.FgRed).Sprint(s)
	case pct >= 70:
		return color.New(color.FgYellow).Sprint(s)
	default:
		return color.New(color.FgGreen).Sprint(s)
	}
}
