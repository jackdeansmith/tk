package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchCommand(t *testing.T) {
	commands := []string{"list", "log", "add", "done", "drop"}

	tests := []struct {
		name      string
		prefix    string
		want      string
		wantError bool
		errorMsg  string
	}{
		{
			name:   "exact match",
			prefix: "list",
			want:   "list",
		},
		{
			name:   "exact match case insensitive",
			prefix: "LIST",
			want:   "list",
		},
		{
			name:   "unique prefix li matches list",
			prefix: "li",
			want:   "list",
		},
		{
			name:   "unique prefix ad matches add",
			prefix: "ad",
			want:   "add",
		},
		{
			name:   "unique prefix don matches done",
			prefix: "don",
			want:   "done",
		},
		{
			name:   "unique prefix dro matches drop",
			prefix: "dro",
			want:   "drop",
		},
		{
			name:      "ambiguous prefix l matches list and log",
			prefix:    "l",
			wantError: true,
			errorMsg:  "ambiguous command",
		},
		{
			name:      "ambiguous prefix lo matches log only but d matches done and drop",
			prefix:    "d",
			wantError: true,
			errorMsg:  "ambiguous command",
		},
		{
			name:      "no match xyz",
			prefix:    "xyz",
			wantError: true,
			errorMsg:  "unknown command",
		},
		{
			name:      "empty prefix is ambiguous",
			prefix:    "",
			wantError: true,
			errorMsg:  "ambiguous command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MatchCommand(tt.prefix, commands)

			if tt.wantError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestMatchCommandEmptyCommands(t *testing.T) {
	_, err := MatchCommand("list", []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestMatchCommandSingleCommand(t *testing.T) {
	got, err := MatchCommand("l", []string{"list"})
	require.NoError(t, err)
	assert.Equal(t, "list", got)
}
