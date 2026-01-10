package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTaskID(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPrefix string
		wantNum    int
		wantErr    bool
	}{
		{
			name:       "standard format",
			input:      "BY-07",
			wantPrefix: "BY",
			wantNum:    7,
		},
		{
			name:       "lowercase",
			input:      "by-7",
			wantPrefix: "BY",
			wantNum:    7,
		},
		{
			name:       "mixed case",
			input:      "By-07",
			wantPrefix: "BY",
			wantNum:    7,
		},
		{
			name:       "leading zeros",
			input:      "BY-007",
			wantPrefix: "BY",
			wantNum:    7,
		},
		{
			name:       "three letter prefix",
			input:      "ABC-123",
			wantPrefix: "ABC",
			wantNum:    123,
		},
		{
			name:       "large number",
			input:      "BY-999",
			wantPrefix: "BY",
			wantNum:    999,
		},
		// Error cases
		{
			name:    "invalid - no dash",
			input:   "BY07",
			wantErr: true,
		},
		{
			name:    "invalid - wait suffix",
			input:   "BY-07W",
			wantErr: true,
		},
		{
			name:    "invalid - no prefix",
			input:   "-07",
			wantErr: true,
		},
		{
			name:    "invalid - no number",
			input:   "BY-",
			wantErr: true,
		},
		{
			name:    "invalid - single letter prefix",
			input:   "B-07",
			wantErr: true,
		},
		{
			name:    "invalid - four letter prefix",
			input:   "ABCD-07",
			wantErr: true,
		},
		{
			name:    "invalid - just text",
			input:   "INVALID",
			wantErr: true,
		},
		{
			name:    "invalid - zero",
			input:   "BY-0",
			wantErr: true,
		},
		{
			name:    "invalid - negative (parsed as text)",
			input:   "BY--1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, num, err := ParseTaskID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidID)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPrefix, prefix)
			assert.Equal(t, tt.wantNum, num)
		})
	}
}

func TestParseWaitID(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPrefix string
		wantNum    int
		wantErr    bool
	}{
		{
			name:       "standard format",
			input:      "BY-03W",
			wantPrefix: "BY",
			wantNum:    3,
		},
		{
			name:       "lowercase",
			input:      "by-3w",
			wantPrefix: "BY",
			wantNum:    3,
		},
		{
			name:       "mixed case",
			input:      "By-03w",
			wantPrefix: "BY",
			wantNum:    3,
		},
		{
			name:       "leading zeros",
			input:      "BY-003W",
			wantPrefix: "BY",
			wantNum:    3,
		},
		{
			name:       "three letter prefix",
			input:      "ABC-123W",
			wantPrefix: "ABC",
			wantNum:    123,
		},
		// Error cases
		{
			name:    "invalid - no W suffix",
			input:   "BY-03",
			wantErr: true,
		},
		{
			name:    "invalid - W in wrong place",
			input:   "BY-W03",
			wantErr: true,
		},
		{
			name:    "invalid - just W",
			input:   "BY-W",
			wantErr: true,
		},
		{
			name:    "invalid - no prefix",
			input:   "-03W",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, num, err := ParseWaitID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.ErrorIs(t, err, ErrInvalidID)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPrefix, prefix)
			assert.Equal(t, tt.wantNum, num)
		})
	}
}

func TestParseAnyID(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantPrefix string
		wantNum    int
		wantIsWait bool
		wantErr    bool
	}{
		{
			name:       "task ID",
			input:      "BY-07",
			wantPrefix: "BY",
			wantNum:    7,
			wantIsWait: false,
		},
		{
			name:       "wait ID",
			input:      "BY-03W",
			wantPrefix: "BY",
			wantNum:    3,
			wantIsWait: true,
		},
		{
			name:       "lowercase task",
			input:      "by-7",
			wantPrefix: "BY",
			wantNum:    7,
			wantIsWait: false,
		},
		{
			name:       "lowercase wait",
			input:      "by-3w",
			wantPrefix: "BY",
			wantNum:    3,
			wantIsWait: true,
		},
		// Error cases
		{
			name:    "invalid format",
			input:   "INVALID",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, num, isWait, err := ParseAnyID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPrefix, prefix)
			assert.Equal(t, tt.wantNum, num)
			assert.Equal(t, tt.wantIsWait, isWait)
		})
	}
}

func TestFormatTaskID(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		num    int
		maxNum int
		want   string
	}{
		{
			name:   "2 digits when max < 100",
			prefix: "BY",
			num:    7,
			maxNum: 99,
			want:   "BY-07",
		},
		{
			name:   "2 digits at boundary",
			prefix: "BY",
			num:    99,
			maxNum: 99,
			want:   "BY-99",
		},
		{
			name:   "3 digits when max >= 100",
			prefix: "BY",
			num:    7,
			maxNum: 100,
			want:   "BY-007",
		},
		{
			name:   "3 digits at 100",
			prefix: "BY",
			num:    100,
			maxNum: 100,
			want:   "BY-100",
		},
		{
			name:   "3 digits mid range",
			prefix: "BY",
			num:    42,
			maxNum: 500,
			want:   "BY-042",
		},
		{
			name:   "4 digits when max >= 1000",
			prefix: "BY",
			num:    7,
			maxNum: 1000,
			want:   "BY-0007",
		},
		{
			name:   "lowercase prefix gets uppercased",
			prefix: "by",
			num:    7,
			maxNum: 99,
			want:   "BY-07",
		},
		{
			name:   "three letter prefix",
			prefix: "ABC",
			num:    42,
			maxNum: 99,
			want:   "ABC-42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTaskID(tt.prefix, tt.num, tt.maxNum)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatWaitID(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		num    int
		maxNum int
		want   string
	}{
		{
			name:   "2 digits when max < 100",
			prefix: "BY",
			num:    3,
			maxNum: 99,
			want:   "BY-03W",
		},
		{
			name:   "3 digits when max >= 100",
			prefix: "BY",
			num:    3,
			maxNum: 100,
			want:   "BY-003W",
		},
		{
			name:   "lowercase prefix gets uppercased",
			prefix: "by",
			num:    3,
			maxNum: 99,
			want:   "BY-03W",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatWaitID(tt.prefix, tt.num, tt.maxNum)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeID(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxNum int
		want   string
	}{
		{
			name:   "already normalized task",
			input:  "BY-07",
			maxNum: 99,
			want:   "BY-07",
		},
		{
			name:   "lowercase task",
			input:  "by-7",
			maxNum: 99,
			want:   "BY-07",
		},
		{
			name:   "already normalized wait",
			input:  "BY-03W",
			maxNum: 99,
			want:   "BY-03W",
		},
		{
			name:   "lowercase wait",
			input:  "by-3w",
			maxNum: 99,
			want:   "BY-03W",
		},
		{
			name:   "add padding for larger max",
			input:  "BY-7",
			maxNum: 100,
			want:   "BY-007",
		},
		{
			name:   "invalid ID returns uppercased original",
			input:  "invalid",
			maxNum: 99,
			want:   "INVALID",
		},
		{
			name:   "zero maxNum uses default",
			input:  "by-7",
			maxNum: 0,
			want:   "BY-07",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeID(tt.input, tt.maxNum)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractPrefix(t *testing.T) {
	assert.Equal(t, "BY", ExtractPrefix("BY-07"))
	assert.Equal(t, "BY", ExtractPrefix("by-7"))
	assert.Equal(t, "BY", ExtractPrefix("BY-03W"))
	assert.Equal(t, "", ExtractPrefix("invalid"))
}

func TestExtractNumber(t *testing.T) {
	assert.Equal(t, 7, ExtractNumber("BY-07"))
	assert.Equal(t, 7, ExtractNumber("by-7"))
	assert.Equal(t, 3, ExtractNumber("BY-03W"))
	assert.Equal(t, 0, ExtractNumber("invalid"))
}

func TestIsWaitID(t *testing.T) {
	assert.True(t, IsWaitID("BY-03W"))
	assert.True(t, IsWaitID("by-3w"))
	assert.False(t, IsWaitID("BY-07"))
	assert.False(t, IsWaitID("by-7"))
	assert.False(t, IsWaitID("invalid"))
}

func TestIsTaskID(t *testing.T) {
	assert.True(t, IsTaskID("BY-07"))
	assert.True(t, IsTaskID("by-7"))
	assert.False(t, IsTaskID("BY-03W"))
	assert.False(t, IsTaskID("by-3w"))
	assert.False(t, IsTaskID("invalid"))
}

func TestDigitWidth_LargeNumbers(t *testing.T) {
	// Test the fallback path for very large numbers (>= 10000)
	tests := []struct {
		name   string
		prefix string
		num    int
		maxNum int
		want   string
	}{
		{
			name:   "5 digits when max >= 10000",
			prefix: "BY",
			num:    7,
			maxNum: 10000,
			want:   "BY-00007",
		},
		{
			name:   "5 digits at 10000",
			prefix: "BY",
			num:    10000,
			maxNum: 10000,
			want:   "BY-10000",
		},
		{
			name:   "6 digits when max >= 100000",
			prefix: "BY",
			num:    42,
			maxNum: 100000,
			want:   "BY-000042",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTaskID(tt.prefix, tt.num, tt.maxNum)
			assert.Equal(t, tt.want, got)
		})
	}
}
