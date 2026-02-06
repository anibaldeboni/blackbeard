package hw

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
)

// RunTempMonitor monitors temperature continuously until interrupted.
func RunTempMonitor(ctx context.Context, p *ui.Printer, intervalSec int) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	p.Info(fmt.Sprintf("Monitoring temperature every %ds (Ctrl+C to stop)", intervalSec))
	p.Println("")

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	// Print once immediately
	printTempLine(p)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			printTempLine(p)
		}
	}
}

func printTempLine(p *ui.Printer) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	temps := ReadTemps()

	cpu := temps["CPU"]
	if !cpu.Valid {
		cpu = TempReading{Label: "CPU", Valid: false}
	}

	gpu := temps["GPU"]
	if !gpu.Valid {
		gpu = TempReading{Label: "GPU", Valid: false}
	}

	p.Printf("[%s] CPU: %s | GPU: %s\n", timestamp, FormatTemp(cpu), FormatTemp(gpu))
}

// RunGPUMonitor monitors GPU/VPU continuously until interrupted.
func RunGPUMonitor(ctx context.Context, p *ui.Printer, intervalSec int) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	p.Info(fmt.Sprintf("Monitoring GPU every %ds (Ctrl+C to stop)", intervalSec))
	p.Println("")

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	printGPULine(p)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Clear screen for dashboard effect
			fmt.Print("\033[2J\033[H")
			fmt.Println("╔══════════════════════════════════════════════════════════════╗")
			fmt.Println("║           Orange Pi 3B - GPU/VPU Monitor                    ║")
			fmt.Println("╚══════════════════════════════════════════════════════════════╝")
			fmt.Println()
			printGPULine(p)
		}
	}
}

func printGPULine(p *ui.Printer) {
	curData, err := os.ReadFile(GPUFreqPath + "/cur_freq")
	if err == nil {
		maxData, _ := os.ReadFile(GPUFreqPath + "/max_freq")
		cur, _ := strconv.Atoi(strings.TrimSpace(string(curData)))
		max, _ := strconv.Atoi(strings.TrimSpace(string(maxData)))
		if max > 0 {
			pct := cur * 100 / max
			p.Printf("GPU Mali:  %3d MHz / %d MHz (%d%%)\n", cur/1000000, max/1000000, pct)
		}
	}

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

	p.Println("")
	p.Println("VPU Interrupts:")
	irqData, err := os.ReadFile("/proc/interrupts")
	if err == nil {
		for _, line := range strings.Split(string(irqData), "\n") {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "hantro") || strings.Contains(lower, "fdea") || strings.Contains(lower, "fdee") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					p.Printf("  %-10s %s\n", fields[len(fields)-1], fields[1])
				}
			}
		}
	} else {
		p.Warning("VPU interrupt info not available")
	}
}
