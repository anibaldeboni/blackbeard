package hw

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/fatih/color"
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

	// Initial read to establish baseline
	prevInterrupts := ReadVPUInterrupts()
	printGPUDashboard(p, nil, intervalSec)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			// Read current interrupts and calculate delta
			currentInterrupts := ReadVPUInterrupts()
			deltas := CalculateVPUDelta(prevInterrupts, currentInterrupts, intervalSec)
			prevInterrupts = currentInterrupts

			// Clear screen for dashboard effect
			fmt.Print("\033[2J\033[H")
			printGPUDashboard(p, deltas, intervalSec)
		}
	}
}

func printGPUDashboard(p *ui.Printer, vpuDeltas []VPUInterruptDelta, intervalSec int) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")

	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Printf("║         Orange Pi 3B - GPU Mali RK3566 Monitor                 ║\n")
	fmt.Printf("║         %s                                       ║\n", timestamp)
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	info := ReadGPUInfo()
	gpuTemp := ReadGPUTempDirect()

	// GPU Status Box
	fmt.Println("┌─ GPU Status ─────────────────────────────────────────────────────┐")

	if info.MaxFreq > 0 {
		freqColor := getFreqColor(info.FreqPct)
		p.Printf("│ Frequency:    %s / %d MHz (%d%%)%s│\n",
			freqColor.Sprintf("%3d MHz", info.CurrentFreq/1000000),
			info.MaxFreq/1000000, info.FreqPct,
			strings.Repeat(" ", 28-len(fmt.Sprintf("%d", info.MaxFreq/1000000))-len(fmt.Sprintf("%d", info.FreqPct))))

		if info.TargetFreq > 0 {
			p.Printf("│ Target Freq:  %d MHz%s│\n",
				info.TargetFreq/1000000,
				strings.Repeat(" ", 48-len(fmt.Sprintf("%d", info.TargetFreq/1000000))))
		}

		p.Printf("│ Governor:     %-20s%s│\n", info.Governor, strings.Repeat(" ", 29))
	}

	if gpuTemp.Valid {
		p.Printf("│ Temperature:  %s%s│\n",
			FormatTemp(gpuTemp),
			strings.Repeat(" ", 49))
	}

	if info.PowerState != "" {
		powerColor := getPowerStateColor(info.PowerState)
		p.Printf("│ Power State:  %s%s│\n",
			powerColor.Sprintf("%-10s", info.PowerState),
			strings.Repeat(" ", 45))
	}

	fmt.Println("└──────────────────────────────────────────────────────────────────┘")
	fmt.Println()

	// Available Frequencies
	if len(info.AvailableFreqs) > 0 {
		fmt.Println("Available Frequencies (MHz):")
		p.Printf("  ")
		for i, freq := range info.AvailableFreqs {
			if i > 0 {
				p.Printf(",  ")
			}
			freqMHz := freq / 1000000
			if freq == info.CurrentFreq {
				getFreqColor(info.FreqPct).Printf("[%d]", freqMHz)
			} else {
				p.Printf("%d", freqMHz)
			}
		}
		p.Printf("\n")
	}
	fmt.Println()

	// VPU/RGA Info
	vpuInfo := ReadVPUInfo()

	// Calculate overall VPU activity
	var maxRate float64
	if vpuDeltas != nil {
		for _, delta := range vpuDeltas {
			if delta.RatePerSec > maxRate {
				maxRate = delta.RatePerSec
			}
		}
	}

	vpuColor, vpuStatus := getVPUActivityColor(maxRate)

	fmt.Println("┌─ VPU Status ─────────────────────────────────────────────────────┐")
	p.Printf("│ Activity:     %s%s│\n",
		vpuColor.Sprintf("%-10s", vpuStatus),
		strings.Repeat(" ", 46))

	if vpuDeltas != nil && len(vpuDeltas) > 0 {
		p.Printf("│ Interrupts/s:%s│\n", strings.Repeat(" ", 49))
		for _, delta := range vpuDeltas {
			intColor, _ := getVPUActivityColor(delta.RatePerSec)
			p.Printf("│   %-12s %s (+%d)%s│\n",
				delta.Name,
				intColor.Sprintf("%.1f/s", delta.RatePerSec),
				delta.Delta,
				strings.Repeat(" ", 33-len(fmt.Sprintf("%.1f/s", delta.RatePerSec))-len(fmt.Sprintf("%d", delta.Delta))))
		}
	} else {
		p.Printf("│ (calculating...)%s│\n", strings.Repeat(" ", 49))
	}

	// VPU Clocks
	if len(vpuInfo.Clocks) > 0 {
		p.Printf("│ Clocks:%s│\n", strings.Repeat(" ", 57))
		for name, freq := range vpuInfo.Clocks {
			freqMHz := freq / 1000000
			if freqMHz > 0 {
				color.New(color.FgGreen).Printf("│   %-18s %4d MHz%s│\n",
					name, freqMHz, strings.Repeat(" ", 36))
			} else {
				p.Printf("│   %-18s %4d MHz%s│\n",
					name, freqMHz, strings.Repeat(" ", 36))
			}
		}
	}

	fmt.Println("└──────────────────────────────────────────────────────────────────┘")
}
