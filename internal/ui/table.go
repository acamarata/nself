package ui

import (
	"fmt"
	"strings"
)

// Table renders data as a box-drawing table to stdout.
type Table struct {
	Headers []string
	Rows    [][]string
	Widths  []int // auto-calculated from data
}

// NewTable creates a new table with the given column headers.
func NewTable(headers ...string) *Table {
	return &Table{
		Headers: headers,
		Rows:    nil,
		Widths:  nil,
	}
}

// AddRow appends a row of values to the table.
func (t *Table) AddRow(values ...string) {
	t.Rows = append(t.Rows, values)
}

// Render outputs the table with box-drawing characters to stdout.
func (t *Table) Render() {
	t.calcWidths()

	// Top border: ┌─────┬─────┐
	t.printBorder("┌", "┬", "┐")

	// Header row
	t.printRow(t.Headers)

	// Header separator: ├─────┼─────┤
	t.printBorder("├", "┼", "┤")

	// Data rows
	for _, row := range t.Rows {
		t.printRow(row)
	}

	// Bottom border: └─────┴─────┘
	t.printBorder("└", "┴", "┘")
}

// calcWidths computes column widths as max(header, longest value) + 2 padding.
func (t *Table) calcWidths() {
	cols := len(t.Headers)
	t.Widths = make([]int, cols)

	for i, h := range t.Headers {
		if len(h) > t.Widths[i] {
			t.Widths[i] = len(h)
		}
	}

	for _, row := range t.Rows {
		for i := 0; i < cols && i < len(row); i++ {
			if len(row[i]) > t.Widths[i] {
				t.Widths[i] = len(row[i])
			}
		}
	}

	// Add 2 for padding (1 space each side)
	for i := range t.Widths {
		t.Widths[i] += 2
	}
}

// printBorder prints a horizontal border line with the given corner/junction characters.
func (t *Table) printBorder(left, mid, right string) {
	var b strings.Builder
	b.WriteString(left)
	for i, w := range t.Widths {
		b.WriteString(strings.Repeat("─", w))
		if i < len(t.Widths)-1 {
			b.WriteString(mid)
		}
	}
	b.WriteString(right)
	fmt.Println(b.String())
}

// printRow prints a single data row with │ separators and padded values.
func (t *Table) printRow(values []string) {
	var b strings.Builder
	b.WriteString("│")
	for i, w := range t.Widths {
		val := ""
		if i < len(values) {
			val = values[i]
		}
		// Left-align with 1 space padding on each side
		b.WriteString(" ")
		fmt.Fprintf(&b, "%-*s", w-2, val)
		b.WriteString(" │")
	}
	fmt.Println(b.String())
}
