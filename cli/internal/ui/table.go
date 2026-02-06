package ui

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Table provides simple tabular output formatting.
type Table struct {
	w       *tabwriter.Writer
	headers []string
}

// NewTable creates a new table writer with the given headers.
func NewTable(out io.Writer, headers ...string) *Table {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	t := &Table{w: w, headers: headers}
	if len(headers) > 0 {
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}
	return t
}

// Row adds a row to the table.
func (t *Table) Row(values ...string) {
	fmt.Fprintln(t.w, strings.Join(values, "\t"))
}

// Flush writes the buffered table to the output.
func (t *Table) Flush() {
	t.w.Flush()
}
