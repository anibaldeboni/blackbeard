package hw

import (
	"os"
	"strconv"
	"strings"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/fatih/color"
)

const (
	CPUTempPath = "/sys/devices/virtual/thermal/thermal_zone0/temp"
	GPUTempPath = "/sys/devices/virtual/thermal/thermal_zone1/temp"
)

// TempReading holds a temperature reading.
type TempReading struct {
	Label   string
	Celsius float64
	Valid   bool
}

// ReadTemp reads temperature from a sysfs thermal zone path.
func ReadTemp(sysfsPath, label string) TempReading {
	data, err := os.ReadFile(sysfsPath)
	if err != nil {
		return TempReading{Label: label, Valid: false}
	}

	raw := strings.TrimSpace(string(data))
	millidegrees, err := strconv.Atoi(raw)
	if err != nil {
		return TempReading{Label: label, Valid: false}
	}

	celsius := float64(millidegrees) / 1000.0
	return TempReading{Label: label, Celsius: celsius, Valid: true}
}

// FormatTemp returns a color-coded temperature string.
func FormatTemp(t TempReading) string {
	if !t.Valid {
		return "N/A"
	}

	c := getTempColor(t.Celsius)
	return c.Sprintf("%.1f", t.Celsius) + "°C"
}

func getTempColor(celsius float64) *color.Color {
	switch {
	case celsius < 45:
		return color.New(color.FgGreen)
	case celsius < 60:
		return color.New(color.FgYellow)
	case celsius < 75:
		return color.New(color.FgHiRed) // Orange-ish
	case celsius < 85:
		return color.New(color.FgRed)
	default:
		return color.New(color.FgRed, color.Bold)
	}
}

// RunTemp shows temperature for the specified target.
func RunTemp(p *ui.Printer, target string) error {
	switch target {
	case "cpu":
		t := ReadTemp(CPUTempPath, "CPU")
		p.Printf("CPU: %s\n", FormatTemp(t))
	case "gpu":
		t := ReadTemp(GPUTempPath, "GPU")
		p.Printf("GPU: %s\n", FormatTemp(t))
	default: // "all"
		cpu := ReadTemp(CPUTempPath, "CPU")
		gpu := ReadTemp(GPUTempPath, "GPU")
		p.Printf("CPU: %s | GPU: %s\n", FormatTemp(cpu), FormatTemp(gpu))
	}
	return nil
}

// RunFullStatus shows full hardware status.
func RunFullStatus(p *ui.Printer) error {
	p.Header("Hardware Status")

	p.Println("Temperature:")
	RunTemp(p, "all")
	p.Println("")

	return RunGPUStatus(p)
}
