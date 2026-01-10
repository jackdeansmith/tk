package model

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// ErrInvalidID is returned when an ID cannot be parsed.
	ErrInvalidID = errors.New("invalid ID format")

	// taskIDRegex matches task IDs like BY-07, by-7, BY-007
	taskIDRegex = regexp.MustCompile(`^([A-Za-z]{2,3})-(\d+)$`)

	// waitIDRegex matches wait IDs like BY-03W, by-3w
	waitIDRegex = regexp.MustCompile(`^([A-Za-z]{2,3})-(\d+)[Ww]$`)

	// anyIDRegex matches both task and wait IDs
	anyIDRegex = regexp.MustCompile(`^([A-Za-z]{2,3})-(\d+)([Ww])?$`)
)

// ParseTaskID parses a task ID string and returns the prefix and number.
// Accepts various formats: BY-07, by-7, BY-007 all parse to prefix="BY", num=7.
// Returns ErrInvalidID if the format is invalid.
func ParseTaskID(s string) (prefix string, num int, err error) {
	matches := taskIDRegex.FindStringSubmatch(s)
	if matches == nil {
		return "", 0, fmt.Errorf("%w: %q is not a valid task ID", ErrInvalidID, s)
	}

	prefix = strings.ToUpper(matches[1])
	num, err = strconv.Atoi(matches[2])
	if err != nil || num <= 0 {
		return "", 0, fmt.Errorf("%w: %q has invalid number", ErrInvalidID, s)
	}

	return prefix, num, nil
}

// ParseWaitID parses a wait ID string and returns the prefix and number.
// Accepts various formats: BY-03W, by-3w, BY-003W all parse to prefix="BY", num=3.
// Returns ErrInvalidID if the format is invalid.
func ParseWaitID(s string) (prefix string, num int, err error) {
	matches := waitIDRegex.FindStringSubmatch(s)
	if matches == nil {
		return "", 0, fmt.Errorf("%w: %q is not a valid wait ID", ErrInvalidID, s)
	}

	prefix = strings.ToUpper(matches[1])
	num, err = strconv.Atoi(matches[2])
	if err != nil || num <= 0 {
		return "", 0, fmt.Errorf("%w: %q has invalid number", ErrInvalidID, s)
	}

	return prefix, num, nil
}

// ParseAnyID parses either a task or wait ID and returns the prefix, number,
// and whether it's a wait.
// Returns ErrInvalidID if the format is invalid.
func ParseAnyID(s string) (prefix string, num int, isWait bool, err error) {
	matches := anyIDRegex.FindStringSubmatch(s)
	if matches == nil {
		return "", 0, false, fmt.Errorf("%w: %q is not a valid ID", ErrInvalidID, s)
	}

	prefix = strings.ToUpper(matches[1])
	num, err = strconv.Atoi(matches[2])
	if err != nil || num <= 0 {
		return "", 0, false, fmt.Errorf("%w: %q has invalid number", ErrInvalidID, s)
	}

	isWait = matches[3] != ""
	return prefix, num, isWait, nil
}

// FormatTaskID formats a task ID with appropriate zero-padding.
// The maxNum parameter determines the padding width:
// - maxNum < 100: 2 digits (BY-01...BY-99)
// - maxNum >= 100 && < 1000: 3 digits (BY-001...BY-999)
// - etc.
func FormatTaskID(prefix string, num int, maxNum int) string {
	width := digitWidth(maxNum)
	return fmt.Sprintf("%s-%0*d", strings.ToUpper(prefix), width, num)
}

// FormatWaitID formats a wait ID with appropriate zero-padding.
// Uses the same padding rules as FormatTaskID, with W suffix.
func FormatWaitID(prefix string, num int, maxNum int) string {
	width := digitWidth(maxNum)
	return fmt.Sprintf("%s-%0*dW", strings.ToUpper(prefix), width, num)
}

// NormalizeID normalizes an ID to uppercase canonical form.
// The maxNum parameter is used for zero-padding. If maxNum is 0,
// the original padding is preserved but the ID is uppercased.
func NormalizeID(s string, maxNum int) string {
	prefix, num, isWait, err := ParseAnyID(s)
	if err != nil {
		// Return uppercased original if parsing fails
		return strings.ToUpper(s)
	}

	// If maxNum is 0, use a default that preserves 2-digit padding
	if maxNum == 0 {
		maxNum = 99
	}

	if isWait {
		return FormatWaitID(prefix, num, maxNum)
	}
	return FormatTaskID(prefix, num, maxNum)
}

// digitWidth returns the number of digits needed to display maxNum.
// Minimum width is 2.
func digitWidth(maxNum int) int {
	if maxNum < 100 {
		return 2
	}
	if maxNum < 1000 {
		return 3
	}
	if maxNum < 10000 {
		return 4
	}
	// Fallback for very large numbers (>= 10000)
	// Count digits in maxNum
	width := 0
	for n := maxNum; n > 0; n /= 10 {
		width++
	}
	return width
}

// ExtractPrefix extracts the project prefix from a task or wait ID.
// Returns empty string if the ID is invalid.
func ExtractPrefix(id string) string {
	prefix, _, _, err := ParseAnyID(id)
	if err != nil {
		return ""
	}
	return prefix
}

// ExtractNumber extracts the numeric part from a task or wait ID.
// Returns 0 if the ID is invalid.
func ExtractNumber(id string) int {
	_, num, _, err := ParseAnyID(id)
	if err != nil {
		return 0
	}
	return num
}

// IsWaitID returns true if the ID represents a wait.
func IsWaitID(id string) bool {
	_, _, isWait, err := ParseAnyID(id)
	return err == nil && isWait
}

// IsTaskID returns true if the ID represents a task (not a wait).
func IsTaskID(id string) bool {
	_, _, isWait, err := ParseAnyID(id)
	return err == nil && !isWait
}
