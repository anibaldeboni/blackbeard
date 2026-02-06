package hw

import (
	"fmt"
	"time"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/shirou/gopsutil/v4/cpu"
)

// RunCPU shows CPU model, core count, and per-core usage.
func RunCPU(p *ui.Printer) error {
	p.Header("CPU")

	infos, err := cpu.Info()
	if err == nil && len(infos) > 0 {
		p.Printf("Model:  %s\n", infos[0].ModelName)
		p.Printf("Cores:  %d\n", len(infos))
	}

	perCorePcts, err := cpu.Percent(time.Second, true)
	if err != nil {
		return fmt.Errorf("reading CPU usage: %w", err)
	}

	// Derive overall from per-core
	var total float64
	for _, pct := range perCorePcts {
		total += pct
	}
	overall := total / float64(len(perCorePcts))
	p.Printf("Usage:  %.1f%%\n", overall)

	p.Println("")
	table := ui.NewTable(p.Out, "CORE", "USAGE")
	for i, pct := range perCorePcts {
		table.Row(fmt.Sprintf("core-%d", i), fmt.Sprintf("%.1f%%", pct))
	}
	table.Flush()

	return nil
}
