package cli

import (
	"bytes"
	"os"
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
