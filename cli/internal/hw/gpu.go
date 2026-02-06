package hw

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/fatih/color"
)

const (
	GPUFreqPath    = "/sys/class/devfreq/fde60000.gpu"
	GPUPowerPath   = "/sys/devices/platform/fde60000.gpu/power"
	GPUThermalZone = "/sys/class/thermal/thermal_zone1"
)

// GPUInfo holds GPU monitoring data.
type GPUInfo struct {
	CurrentFreq    int64
	TargetFreq     int64
	MinFreq        int64
	MaxFreq        int64
	Governor       string
	AvailableFreqs []int64
	PowerState     string
	FreqPct        int
	TransStat      string
}

// VPUInfo holds VPU monitoring data.
type VPUInfo struct {
	Interrupts map[string]int64 // interrupt name -> count
	Clocks     map[string]int64 // clock name -> frequency (Hz)
	Active     bool             // true if VPU is processing
}

// VPUInterruptDelta holds interrupt delta information.
type VPUInterruptDelta struct {
	Name       string
	Count      int64
	Delta      int64
	RatePerSec float64
}

// ReadGPUInfo reads GPU metrics from sysfs.
func ReadGPUInfo() GPUInfo {
	info := GPUInfo{}

	// Read frequencies
	if data, err := os.ReadFile(GPUFreqPath + "/cur_freq"); err == nil {
		info.CurrentFreq, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}
	if data, err := os.ReadFile(GPUFreqPath + "/target_freq"); err == nil {
		info.TargetFreq, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}
	if data, err := os.ReadFile(GPUFreqPath + "/min_freq"); err == nil {
		info.MinFreq, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}
	if data, err := os.ReadFile(GPUFreqPath + "/max_freq"); err == nil {
		info.MaxFreq, _ = strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	}

	// Calculate frequency percentage
	if info.MaxFreq > 0 {
		info.FreqPct = int(info.CurrentFreq * 100 / info.MaxFreq)
	}

	// Read governor
	if data, err := os.ReadFile(GPUFreqPath + "/governor"); err == nil {
		info.Governor = strings.TrimSpace(string(data))
	}

	// Read available frequencies
	if data, err := os.ReadFile(GPUFreqPath + "/available_frequencies"); err == nil {
		freqStrs := strings.Fields(string(data))
		for _, fs := range freqStrs {
			if freq, err := strconv.ParseInt(fs, 10, 64); err == nil {
				info.AvailableFreqs = append(info.AvailableFreqs, freq)
			}
		}
	}

	// Read power state
	if data, err := os.ReadFile(GPUPowerPath + "/runtime_status"); err == nil {
		info.PowerState = strings.TrimSpace(string(data))
	}

	// Read trans_stat (raw for future use)
	if data, err := os.ReadFile(GPUFreqPath + "/trans_stat"); err == nil {
		info.TransStat = string(data)
	}

	return info
}

// ReadVPUInterrupts reads VPU interrupt counts from /proc/interrupts.
func ReadVPUInterrupts() map[string]int64 {
	result := make(map[string]int64)

	data, err := os.ReadFile("/proc/interrupts")
	if err != nil {
		return result
	}

	for _, line := range strings.Split(string(data), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "hantro") || strings.Contains(lower, "fdea") || strings.Contains(lower, "fdee") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				name := fields[len(fields)-1]
				count, _ := strconv.ParseInt(fields[1], 10, 64)
				result[name] = count
			}
		}
	}

	return result
}

// ReadVPUClocks reads VPU/RGA clock frequencies.
func ReadVPUClocks() map[string]int64 {
	result := make(map[string]int64)

	data, err := os.ReadFile("/sys/kernel/debug/clk/clk_summary")
	if err != nil {
		return result
	}

	for _, line := range strings.Split(string(data), "\n") {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "vpu") || strings.Contains(lower, "rga") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				freq, _ := strconv.ParseInt(fields[3], 10, 64)
				result[fields[0]] = freq
			}
		}
	}

	return result
}

// ReadVPUInfo reads VPU monitoring data.
func ReadVPUInfo() VPUInfo {
	info := VPUInfo{
		Interrupts: ReadVPUInterrupts(),
		Clocks:     ReadVPUClocks(),
	}

	// Determine if VPU is active based on interrupt counts
	// (actual activity is determined by delta in monitor)
	info.Active = len(info.Interrupts) > 0

	return info
}

// CalculateVPUDelta calculates interrupt deltas between two readings.
func CalculateVPUDelta(prev, current map[string]int64, intervalSec int) []VPUInterruptDelta {
	var deltas []VPUInterruptDelta

	for name, curCount := range current {
		prevCount := prev[name]
		delta := curCount - prevCount
		rate := 0.0
		if intervalSec > 0 && delta > 0 {
			rate = float64(delta) / float64(intervalSec)
		}

		deltas = append(deltas, VPUInterruptDelta{
			Name:       name,
			Count:      curCount,
			Delta:      delta,
			RatePerSec: rate,
		})
	}

	return deltas
}

// getFreqColor returns color based on frequency percentage.
func getFreqColor(pct int) *color.Color {
	switch {
	case pct < 50:
		return color.New(color.FgGreen)
	case pct < 75:
		return color.New(color.FgYellow)
	default:
		return color.New(color.FgRed)
	}
}

// getPowerStateColor returns color based on power state.
func getPowerStateColor(state string) *color.Color {
	if state == "active" {
		return color.New(color.FgGreen)
	}
	return color.New(color.FgYellow)
}

// getVPUActivityColor returns color and status based on interrupt rate.
func getVPUActivityColor(rate float64) (*color.Color, string) {
	switch {
	case rate == 0:
		return color.New(color.FgHiBlack), "IDLE"
	case rate < 10:
		return color.New(color.FgYellow), "LOW"
	case rate < 100:
		return color.New(color.FgGreen), "ACTIVE"
	default:
		return color.New(color.FgHiGreen, color.Bold), "HIGH"
	}
}

// RunGPUStatus shows GPU/VPU status.
func RunGPUStatus(p *ui.Printer) error {
	p.Header("GPU/VPU Status (RK3566)")

	// Read GPU info
	info := ReadGPUInfo()

	// GPU Mali details
	p.Println("GPU Mali:")
	if info.MaxFreq > 0 {
		freqColor := getFreqColor(info.FreqPct)
		p.Printf("  Frequency:  %s / %d MHz (%d%%)\n",
			freqColor.Sprintf("%3d MHz", info.CurrentFreq/1000000),
			info.MaxFreq/1000000, info.FreqPct)
		if info.TargetFreq > 0 {
			p.Printf("  Target:     %d MHz\n", info.TargetFreq/1000000)
		}
		p.Printf("  Range:      %d - %d MHz\n", info.MinFreq/1000000, info.MaxFreq/1000000)
		p.Printf("  Governor:   %s\n", info.Governor)
		if len(info.AvailableFreqs) > 0 {
			p.Printf("  Available:  ")
			for i, freq := range info.AvailableFreqs {
				if i > 0 {
					p.Printf(", ")
				}
				if freq == info.CurrentFreq {
					freqColor.Printf("%d*", freq/1000000)
				} else {
					p.Printf("%d", freq/1000000)
				}
			}
			p.Printf(" MHz\n")
		}
	} else {
		p.Warning(fmt.Sprintf("GPU frequency path not found: %s", GPUFreqPath))
	}

	// Temperature
	gpuTemp := ReadGPUTempDirect()
	if gpuTemp.Valid {
		p.Printf("  Temperature: %s\n", FormatTemp(gpuTemp))
	}

	// Power state
	if info.PowerState != "" {
		powerColor := getPowerStateColor(info.PowerState)
		p.Printf("  Power:      %s\n", powerColor.Sprint(info.PowerState))
	}

	// VPU/RGA Status
	p.Println("")
	p.Println("VPU/RGA Status:")

	vpuInfo := ReadVPUInfo()

	// VPU Clocks
	if len(vpuInfo.Clocks) > 0 {
		p.Println("  Clocks:")
		for name, freq := range vpuInfo.Clocks {
			freqMHz := freq / 1000000
			// Highlight if clock is active (> 0 MHz)
			if freqMHz > 0 {
				color.New(color.FgGreen).Printf("    %-20s %d MHz\n", name, freqMHz)
			} else {
				p.Printf("    %-20s %d MHz\n", name, freqMHz)
			}
		}
	} else {
		p.Warning("  VPU/RGA clock info not available")
	}

	// VPU Interrupts
	p.Println("")
	if len(vpuInfo.Interrupts) > 0 {
		p.Println("  Interrupts (total counts):")
		for name, count := range vpuInfo.Interrupts {
			p.Printf("    %-15s %d\n", name, count)
		}
		p.Println("")
		color.New(color.FgCyan).Println("  💡 Use 'gpu-monitor' to see VPU activity in real-time")
	} else {
		p.Warning("  VPU interrupt info not available")
	}

	return nil
}
