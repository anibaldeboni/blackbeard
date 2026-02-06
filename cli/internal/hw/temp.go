package hw

import (
	"strings"

	"github.com/anibalnet/blackbeard/cli/internal/ui"
	"github.com/fatih/color"
	"github.com/shirou/gopsutil/v4/sensors"
)

// TempReading holds a temperature reading.
type TempReading struct {
	Label   string
	Celsius float64
	Valid   bool
}

// ReadTemps reads all temperature sensors via gopsutil.
func ReadTemps() map[string]TempReading {
	result := make(map[string]TempReading)

	temps, err := sensors.SensorsTemperatures()
	if err != nil {
		return result
	}

	for _, t := range temps {
		if t.Temperature <= 0 {
			continue
		}
		label := classifyTempSensor(t.SensorKey)
		if label == "" {
			continue
		}
		if _, exists := result[label]; !exists {
			result[label] = TempReading{
				Label:   label,
				Celsius: t.Temperature,
				Valid:   true,
			}
		}
	}

	return result
}

// classifyTempSensor maps gopsutil sensor keys to labels.
// On RK3566 thermal zones are typically named with cpu/soc/gpu prefixes.
func classifyTempSensor(key string) string {
	lower := strings.ToLower(key)
	switch {
	case strings.Contains(lower, "cpu") || strings.Contains(lower, "soc"):
		return "CPU"
	case strings.Contains(lower, "gpu"):
		return "GPU"
	default:
		return ""
	}
}

// FormatTemp returns a color-coded temperature string.
func FormatTemp(t TempReading) string {
	if !t.Valid {
		return "N/A"
	}
	c := getTempColor(t.Celsius)
	return c.Sprintf("%.1f", t.Celsius) + "\u00b0C"
}

func getTempColor(celsius float64) *color.Color {
	switch {
	case celsius < 45:
		return color.New(color.FgGreen)
	case celsius < 60:
		return color.New(color.FgYellow)
	case celsius < 75:
		return color.New(color.FgHiRed)
	case celsius < 85:
		return color.New(color.FgRed)
	default:
		return color.New(color.FgRed, color.Bold)
	}
}

// RunTemp shows temperature for the specified target.
func RunTemp(p *ui.Printer, target string) error {
	temps := ReadTemps()

	get := func(label string) TempReading {
		if t, ok := temps[label]; ok {
			return t
		}
		return TempReading{Label: label, Valid: false}
	}

	switch target {
	case "cpu":
		p.Printf("CPU: %s\n", FormatTemp(get("CPU")))
	case "gpu":
		p.Printf("GPU: %s\n", FormatTemp(get("GPU")))
	default:
		p.Printf("CPU: %s | GPU: %s\n", FormatTemp(get("CPU")), FormatTemp(get("GPU")))
	}
	return nil
}

// RunFullStatus shows comprehensive hardware status.
func RunFullStatus(p *ui.Printer) error {
	RunInfo(p)
	p.Println("")

	p.Header("Temperature")
	RunTemp(p, "all")
	p.Println("")

	RunCPU(p)
	p.Println("")

	RunMem(p)
	p.Println("")

	RunDisk(p, nil)
	p.Println("")

	RunNet(p)
	p.Println("")

	return RunGPUStatus(p)
}
