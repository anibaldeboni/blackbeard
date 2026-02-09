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

type Position int

const (
	Top Position = iota
	Bottom
)

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

// Header prints a modern styled header with multiple content lines.
func (p *Printer) Header(msgs ...string) {
	if len(msgs) == 0 {
		return
	}

	width := p.calculateHeaderWidth(msgs...)
	p.printHeaderLine(width, Top)

	for _, msg := range msgs {
		p.printHeaderContent(msg, width)
	}

	p.printHeaderLine(width, Bottom)
	p.Println("")
}

// calculateHeaderWidth calculates the minimum width needed for all messages.
func (p *Printer) calculateHeaderWidth(msgs ...string) int {
	maxLen := 0
	for _, msg := range msgs {
		if len(msg) > maxLen {
			maxLen = len(msg)
		}
	}
	// Add padding: 2 for borders + 4 for internal spacing
	width := max(maxLen+6, 50)
	return width
}

// printHeaderLine prints the top or bottom border of the header.
func (p *Printer) printHeaderLine(width int, style Position) {
	cyan := color.New(color.FgCyan, color.Bold)
	blue := color.New(color.FgBlue, color.Bold)
	hiBlue := color.New(color.FgHiBlue, color.Bold)

	if style == Top {
		cyan.Fprint(p.Out, "\n╔")
	} else {
		cyan.Fprint(p.Out, "╚")
	}

	// Print border with gradient effect
	for i := 0; i < width-2; i++ {
		if i < width/3 {
			cyan.Fprint(p.Out, "═")
		} else if i < 2*width/3 {
			blue.Fprint(p.Out, "═")
		} else {
			hiBlue.Fprint(p.Out, "═")
		}
	}

	if style == Top {
		cyan.Fprintln(p.Out, "╗")
	} else {
		cyan.Fprintln(p.Out, "╝")
	}
}

// printHeaderContent prints a centered content line within the header box.
func (p *Printer) printHeaderContent(msg string, width int) {
	cyan := color.New(color.FgCyan, color.Bold)
	white := color.New(color.FgHiWhite, color.Bold)

	leftPadding := (width - len(msg) - 2) / 2
	rightPadding := width - len(msg) - leftPadding - 3

	cyan.Fprint(p.Out, "║")
	fmt.Fprint(p.Out, " ")
	for range leftPadding {
		fmt.Fprint(p.Out, " ")
	}
	white.Fprint(p.Out, msg)
	for range rightPadding {
		fmt.Fprint(p.Out, " ")
	}
	cyan.Fprintln(p.Out, "║")
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
