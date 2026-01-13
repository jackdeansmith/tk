// Package cli provides CLI infrastructure for tk.
package cli

import (
	"fmt"
	"strings"
)

// MatchCommand finds a unique command from a prefix.
// Returns the matched command or an error if ambiguous or no match.
func MatchCommand(prefix string, commands []string) (string, error) {
	prefix = strings.ToLower(prefix)

	// First check for exact match
	for _, cmd := range commands {
		if strings.ToLower(cmd) == prefix {
			return cmd, nil
		}
	}

	// Check for prefix match
	var matches []string
	for _, cmd := range commands {
		if strings.HasPrefix(strings.ToLower(cmd), prefix) {
			matches = append(matches, cmd)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("unknown command %q", prefix)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous command %q matches: %s", prefix, strings.Join(matches, ", "))
	}
}
