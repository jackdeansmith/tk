package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorGray   = "\033[90m"
)

// colorEnabled tracks whether color output is enabled.
// It is set based on terminal detection but can be overridden.
var colorEnabled = true

func init() {
	// Disable colors if stdout is not a terminal
	colorEnabled = IsTerminal(os.Stdout)
}

// SetColorEnabled allows overriding the color output setting.
func SetColorEnabled(enabled bool) {
	colorEnabled = enabled
}

// ColorEnabled returns whether color output is currently enabled.
func ColorEnabled() bool {
	return colorEnabled
}

// IsTerminal returns true if w is a terminal.
func IsTerminal(w io.Writer) bool {
	if f, ok := w.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// Green returns s wrapped in green ANSI codes if colors are enabled.
func Green(s string) string {
	if !colorEnabled {
		return s
	}
	return colorGreen + s + colorReset
}

// Red returns s wrapped in red ANSI codes if colors are enabled.
func Red(s string) string {
	if !colorEnabled {
		return s
	}
	return colorRed + s + colorReset
}

// Yellow returns s wrapped in yellow ANSI codes if colors are enabled.
func Yellow(s string) string {
	if !colorEnabled {
		return s
	}
	return colorYellow + s + colorReset
}

// Gray returns s wrapped in gray ANSI codes if colors are enabled.
func Gray(s string) string {
	if !colorEnabled {
		return s
	}
	return colorGray + s + colorReset
}

// Table formats columnar output with automatic column width calculation.
type Table struct {
	rows      [][]string
	colWidths []int
}

// NewTable creates a new empty table.
func NewTable() *Table {
	return &Table{}
}

// AddRow adds a row to the table.
func (t *Table) AddRow(cols ...string) {
	// Expand colWidths if needed
	for len(t.colWidths) < len(cols) {
		t.colWidths = append(t.colWidths, 0)
	}

	// Update column widths based on visible width (excluding ANSI codes)
	for i, col := range cols {
		width := visibleWidth(col)
		if width > t.colWidths[i] {
			t.colWidths[i] = width
		}
	}

	t.rows = append(t.rows, cols)
}

// Render writes the table to w with columns separated by two spaces.
func (t *Table) Render(w io.Writer) {
	for _, row := range t.rows {
		var parts []string
		for i, col := range row {
			if i < len(t.colWidths)-1 {
				// Pad all columns except the last
				padding := t.colWidths[i] - visibleWidth(col)
				parts = append(parts, col+strings.Repeat(" ", padding))
			} else {
				// Last column doesn't need padding
				parts = append(parts, col)
			}
		}
		fmt.Fprintln(w, strings.Join(parts, "  "))
	}
}

// visibleWidth returns the visible width of s, excluding ANSI escape codes.
func visibleWidth(s string) int {
	width := 0
	inEscape := false

	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		width++
	}

	return width
}
