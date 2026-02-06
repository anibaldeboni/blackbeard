package hw

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
)

const (
	GPUFreqPath = "/sys/class/devfreq/fde60000.gpu"
)

// RunGPUStatus shows GPU/VPU status.
func RunGPUStatus(p *ui.Printer) error {
	p.Header("GPU/VPU Status (RK3566)")

	// GPU Mali frequency
	curFreqData, err := os.ReadFile(GPUFreqPath + "/cur_freq")
	if err == nil {
		maxFreqData, _ := os.ReadFile(GPUFreqPath + "/max_freq")

		curFreq, _ := strconv.Atoi(strings.TrimSpace(string(curFreqData)))
		maxFreq, _ := strconv.Atoi(strings.TrimSpace(string(maxFreqData)))

		if maxFreq > 0 {
			pct := curFreq * 100 / maxFreq
			p.Printf("GPU Mali:  %3d MHz / %d MHz (%d%%)\n",
				curFreq/1000000, maxFreq/1000000, pct)
		}
	} else {
		p.Warning(fmt.Sprintf("GPU frequency path not found: %s", GPUFreqPath))
	}

	// VPU/RGA Clocks
	p.Println("")
	p.Println("VPU/RGA Clocks:")
	clkData, err := os.ReadFile("/sys/kernel/debug/clk/clk_summary")
	if err == nil {
		for _, line := range strings.Split(string(clkData), "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "vpu") || strings.Contains(lower, "rga") {
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					freq, _ := strconv.Atoi(fields[3])
					p.Printf("  %-20s %d MHz\n", fields[0], freq/1000000)
				}
			}
		}
	} else {
		p.Warning("VPU/RGA clock info not available")
	}

	// VPU Interrupts
	p.Println("")
	p.Println("VPU Interrupts:")
	irqData, err := os.ReadFile("/proc/interrupts")
	if err == nil {
		found := false
		for _, line := range strings.Split(string(irqData), "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "hantro") || strings.Contains(lower, "fdea") || strings.Contains(lower, "fdee") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					name := fields[len(fields)-1]
					count := fields[1]
					p.Printf("  %-10s %s\n", name, count)
					found = true
				}
			}
		}
		if !found {
			p.Warning("VPU interrupt info not available")
		}
	} else {
		p.Warning("VPU interrupt info not available")
	}

	return nil
}
