package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsTerminal(t *testing.T) {
	// When running tests, stdout is typically not a terminal
	// We test with a regular file which should not be a terminal
	f, err := os.CreateTemp("", "test")
	if err != nil {
		t.Skip("cannot create temp file")
	}
	defer os.Remove(f.Name())
	defer f.Close()

	assert.False(t, IsTerminal(f), "temp file should not be a terminal")

	// bytes.Buffer is not a terminal
	var buf bytes.Buffer
	assert.False(t, IsTerminal(&buf), "bytes.Buffer should not be a terminal")
}

func TestColorFunctions(t *testing.T) {
	// Test with colors enabled
	SetColorEnabled(true)

	assert.Equal(t, "\033[32mtest\033[0m", Green("test"))
	assert.Equal(t, "\033[31mtest\033[0m", Red("test"))
	assert.Equal(t, "\033[33mtest\033[0m", Yellow("test"))
	assert.Equal(t, "\033[90mtest\033[0m", Gray("test"))

	// Test with colors disabled
	SetColorEnabled(false)

	assert.Equal(t, "test", Green("test"))
	assert.Equal(t, "test", Red("test"))
	assert.Equal(t, "test", Yellow("test"))
	assert.Equal(t, "test", Gray("test"))

	// Restore default (for other tests)
	SetColorEnabled(true)
}

func TestColorEnabled(t *testing.T) {
	SetColorEnabled(true)
	assert.True(t, ColorEnabled())

	SetColorEnabled(false)
	assert.False(t, ColorEnabled())

	// Restore
	SetColorEnabled(true)
}

func TestTableEmpty(t *testing.T) {
	table := NewTable()
	var buf bytes.Buffer
	table.Render(&buf)
	assert.Equal(t, "", buf.String())
}

func TestTableSingleRow(t *testing.T) {
	table := NewTable()
	table.AddRow("one", "two", "three")

	var buf bytes.Buffer
	table.Render(&buf)
	assert.Equal(t, "one  two  three\n", buf.String())
}

func TestTableMultipleRows(t *testing.T) {
	table := NewTable()
	table.AddRow("a", "bb", "ccc")
	table.AddRow("dddd", "e", "ff")

	var buf bytes.Buffer
	table.Render(&buf)

	expected := "a     bb  ccc\n" +
		"dddd  e   ff\n"
	assert.Equal(t, expected, buf.String())
}

func TestTableColumnAlignment(t *testing.T) {
	table := NewTable()
	table.AddRow("BY-01", "[done]", "Get paper bags")
	table.AddRow("BY-02", "[blocked]", "Fill bags with weeds")
	table.AddRow("BY-100", "[ready]", "Order more gravel")

	var buf bytes.Buffer
	table.Render(&buf)

	lines := []string{
		"BY-01   [done]     Get paper bags",
		"BY-02   [blocked]  Fill bags with weeds",
		"BY-100  [ready]    Order more gravel",
	}
	expected := lines[0] + "\n" + lines[1] + "\n" + lines[2] + "\n"
	assert.Equal(t, expected, buf.String())
}

func TestTableWithColoredText(t *testing.T) {
	SetColorEnabled(true)
	defer SetColorEnabled(false)

	table := NewTable()
	table.AddRow("ID", Green("done"), "Task")
	table.AddRow("BY-01", Red("blocked"), "Another task")

	var buf bytes.Buffer
	table.Render(&buf)

	// Columns should still align correctly despite ANSI codes
	output := buf.String()
	assert.Contains(t, output, "done")
	assert.Contains(t, output, "blocked")
}

func TestVisibleWidth(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello", 5},
		{"", 0},
		{"\033[32mhello\033[0m", 5}, // green "hello"
		{"\033[31m\033[0m", 0},      // empty colored string
		{"a\033[32mb\033[0mc", 3},   // mixed colored/plain
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := visibleWidth(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTruncatePlainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{"no truncation needed", "hello", 10, "hello"},
		{"exact fit", "hello", 5, "hello"},
		{"truncated", "hello world", 8, "hello..."},
		{"very short max", "hello world", 3, "..."},
		{"max 1", "hello", 1, "h"},
		{"max 0", "hello", 0, ""},
		{"empty string", "", 10, ""},
		{"long title", strings.Repeat("x", 100), 20, strings.Repeat("x", 17) + "..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.input, tt.maxWidth)
			assert.Equal(t, tt.want, got)
			// visible width must not exceed maxWidth
			assert.LessOrEqual(t, visibleWidth(got), tt.maxWidth)
		})
	}
}

func TestTruncateWithANSI(t *testing.T) {
	SetColorEnabled(true)
	defer SetColorEnabled(false)

	// Green "hello world" - 11 visible chars, truncate to 8
	colored := Green("hello world")
	got := Truncate(colored, 8)
	assert.Equal(t, 8, visibleWidth(got))
	assert.Contains(t, got, "...")
	assert.True(t, strings.HasSuffix(got, colorReset), "should end with ANSI reset")
}

func TestTruncatePreservesShortColoredText(t *testing.T) {
	SetColorEnabled(true)
	defer SetColorEnabled(false)

	colored := Green("hi")
	got := Truncate(colored, 10)
	assert.Equal(t, colored, got, "should not truncate short colored text")
}

func TestTableSetMaxWidth(t *testing.T) {
	table := NewTable()
	table.SetMaxWidth(1, 10)

	table.AddRow("ID", strings.Repeat("x", 100), "end")
	table.AddRow("ID", "short", "end")

	var buf bytes.Buffer
	table.Render(&buf)

	output := buf.String()
	// The long column should be truncated
	assert.Contains(t, output, "...")
	// Column width should be capped at 10, not 100
	assert.NotContains(t, output, strings.Repeat("x", 100))
	// Both rows should have "end" as the last column
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Len(t, lines, 2)
	for _, line := range lines {
		assert.True(t, strings.HasSuffix(strings.TrimSpace(line), "end"))
	}
}

func TestTableMaxWidthAlignsProperly(t *testing.T) {
	table := NewTable()
	table.SetMaxWidth(2, 15)

	table.AddRow("BY-01", "[ready]", "Short title", "[bug]")
	table.AddRow("BY-02", "[ready]", "This title is way too long for the column", "[fix]")

	var buf bytes.Buffer
	table.Render(&buf)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	assert.Len(t, lines, 2)

	// Both lines should contain the tag at the end
	assert.True(t, strings.HasSuffix(strings.TrimSpace(lines[0]), "[bug]"))
	assert.True(t, strings.HasSuffix(strings.TrimSpace(lines[1]), "[fix]"))

	// The truncated title should end with "..."
	assert.Contains(t, lines[1], "...")
}

func TestTableNoMaxWidthUnchangedBehavior(t *testing.T) {
	// Without SetMaxWidth, the table should behave exactly as before
	table := NewTable()
	longTitle := strings.Repeat("z", 200)
	table.AddRow("ID", longTitle, "end")

	var buf bytes.Buffer
	table.Render(&buf)

	output := buf.String()
	assert.Contains(t, output, longTitle, "without max width, long text should not be truncated")
}

func TestTableUnevenRows(t *testing.T) {
	table := NewTable()
	table.AddRow("a", "b", "c")
	table.AddRow("d", "e") // fewer columns

	var buf bytes.Buffer
	table.Render(&buf)

	// Should handle gracefully without panicking
	output := buf.String()
	assert.Contains(t, output, "a")
	assert.Contains(t, output, "d")
}
