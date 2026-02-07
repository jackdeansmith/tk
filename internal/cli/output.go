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

// DefaultMaxTitleWidth is the default maximum visible width for title columns.
const DefaultMaxTitleWidth = 60

// Table formats columnar output with automatic column width calculation.
type Table struct {
	rows      [][]string
	colWidths []int
	maxWidths map[int]int // optional per-column max visible width
}

// NewTable creates a new empty table.
func NewTable() *Table {
	return &Table{}
}

// SetMaxWidth sets the maximum visible width for a column.
// Content exceeding the limit is truncated with an ellipsis ("...").
func (t *Table) SetMaxWidth(col, maxWidth int) {
	if t.maxWidths == nil {
		t.maxWidths = make(map[int]int)
	}
	t.maxWidths[col] = maxWidth
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
		// Cap the tracked width if a max is set for this column
		if maxW, ok := t.maxWidths[i]; ok && width > maxW {
			width = maxW
		}
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
			// Truncate if a max width is set for this column
			if maxW, ok := t.maxWidths[i]; ok {
				col = Truncate(col, maxW)
			}
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

// Truncate returns s truncated to maxWidth visible characters. If s exceeds
// maxWidth, it is cut and "..." is appended (counted within the limit).
// ANSI escape codes are preserved up to the truncation point with a reset appended.
// If maxWidth is less than 4, truncation still applies but the ellipsis may use all
// available space.
func Truncate(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if visibleWidth(s) <= maxWidth {
		return s
	}

	ellipsis := "..."
	// If the max width can't even fit the ellipsis, just hard-truncate
	// to maxWidth visible characters.
	if maxWidth < len(ellipsis) {
		var result strings.Builder
		visible := 0
		inEscape := false
		for _, r := range s {
			if r == '\033' {
				inEscape = true
				result.WriteRune(r)
				continue
			}
			if inEscape {
				result.WriteRune(r)
				if r == 'm' {
					inEscape = false
				}
				continue
			}
			if visible >= maxWidth {
				break
			}
			result.WriteRune(r)
			visible++
		}
		return result.String()
	}
	limit := maxWidth - len(ellipsis)

	var result strings.Builder
	visible := 0
	inEscape := false
	hasAnsi := false

	for _, r := range s {
		if r == '\033' {
			inEscape = true
			hasAnsi = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		if visible >= limit {
			break
		}
		result.WriteRune(r)
		visible++
	}

	result.WriteString(ellipsis)
	if hasAnsi {
		result.WriteString(colorReset)
	}
	return result.String()
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
