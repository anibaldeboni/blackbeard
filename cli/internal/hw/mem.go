package hw

import (
	"fmt"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/shirou/gopsutil/v4/mem"
)

// RunMem shows RAM and swap usage.
func RunMem(p *ui.Printer) error {
	p.Header("Memory")

	vm, err := mem.VirtualMemory()
	if err != nil {
		return fmt.Errorf("reading memory: %w", err)
	}

	p.Printf("RAM:   %s used / %s total (%.1f%%)\n",
		formatBytesHW(vm.Used), formatBytesHW(vm.Total), vm.UsedPercent)
	p.Printf("Avail: %s\n", formatBytesHW(vm.Available))
	p.Printf("Buffers/Cache: %s / %s\n",
		formatBytesHW(vm.Buffers), formatBytesHW(vm.Cached))

	sw, err := mem.SwapMemory()
	if err == nil {
		if sw.Total > 0 {
			p.Printf("Swap:  %s used / %s total (%.1f%%)\n",
				formatBytesHW(sw.Used), formatBytesHW(sw.Total), sw.UsedPercent)
		} else {
			p.Println("Swap:  disabled")
		}
	}

	return nil
}

// formatBytesHW formats bytes in human-readable form (1024-based).
func formatBytesHW(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
