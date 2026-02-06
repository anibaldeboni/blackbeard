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

// Header prints a blue boxed header.
func (p *Printer) Header(msg string) {
	blue := color.New(color.FgBlue)
	blue.Fprintln(p.Out, "================================")
	blue.Fprintln(p.Out, msg)
	blue.Fprintln(p.Out, "================================")
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
