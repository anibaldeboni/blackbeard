package ui

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// Printer handles all terminal output with consistent formatting.
type Printer struct {
	Out     io.Writer
	noColor bool
}

// NewPrinter creates a new Printer. If noColor is true, disables color output.
func NewPrinter(noColor bool) *Printer {
	if noColor {
		color.NoColor = true
	}
	return &Printer{
		Out:     os.Stdout,
		noColor: noColor,
	}
}

// Header prints a modern styled header with gradient effect.
func (p *Printer) Header(msg string) {
	cyan := color.New(color.FgCyan, color.Bold)
	blue := color.New(color.FgBlue, color.Bold)

	// Calculate box width based on message length
	width := max(len(msg)+4, 50)

	// Top border with gradient effect
	cyan.Fprint(p.Out, "\n╔")
	for i := 0; i < width-2; i++ {
		if i < width/3 {
			cyan.Fprint(p.Out, "═")
		} else if i < 2*width/3 {
			blue.Fprint(p.Out, "═")
		} else {
			color.New(color.FgHiBlue).Fprint(p.Out, "═")
		}
	}
	cyan.Fprintln(p.Out, "╗")

	// Message with padding
	padding := (width - len(msg) - 2) / 2
	cyan.Fprint(p.Out, "║")
	fmt.Fprint(p.Out, " ")
	for i := 0; i < padding-1; i++ {
		fmt.Fprint(p.Out, " ")
	}
	color.New(color.FgHiWhite, color.Bold).Fprint(p.Out, msg)
	for i := 0; i < width-len(msg)-padding-2; i++ {
		fmt.Fprint(p.Out, " ")
	}
	cyan.Fprintln(p.Out, "║")

	// Bottom border
	cyan.Fprint(p.Out, "╚")
	for i := 0; i < width-2; i++ {
		cyan.Fprint(p.Out, "═")
	}
	cyan.Fprintln(p.Out, "╝")
}

// Success prints a green checkmark message.
func (p *Printer) Success(msg string) {
	color.New(color.FgGreen).Fprintf(p.Out, "✓ %s\n", msg)
}

// Warning prints a yellow warning message.
func (p *Printer) Warning(msg string) {
	color.New(color.FgYellow).Fprintf(p.Out, "⚠ %s\n", msg)
}

// Error prints a red error message.
func (p *Printer) Error(msg string) {
	color.New(color.FgRed).Fprintf(p.Out, "✗ %s\n", msg)
}

// Info prints a blue info message.
func (p *Printer) Info(msg string) {
	color.New(color.FgBlue).Fprintf(p.Out, "ℹ %s\n", msg)
}

// Println prints a plain message.
func (p *Printer) Println(msg string) {
	fmt.Fprintln(p.Out, msg)
}

// Printf prints a formatted message.
func (p *Printer) Printf(format string, args ...any) {
	fmt.Fprintf(p.Out, format, args...)
}
