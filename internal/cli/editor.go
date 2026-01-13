package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// EditInEditor opens content in $EDITOR and returns modified content.
// The suffix is used for the temporary file (e.g., ".yaml" for syntax highlighting).
// Returns error if EDITOR/VISUAL not set or editor exits non-zero.
func EditInEditor(content []byte, suffix string) ([]byte, error) {
	editor := getEditor()
	if editor == "" {
		return nil, fmt.Errorf("EDITOR not set. Set it or use --flags instead of -i")
	}

	// Create temp file with suffix for syntax highlighting
	tmpFile, err := os.CreateTemp("", "tk-*"+suffix)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write content to temp file
	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Run editor
	if err := runEditor(editor, tmpPath); err != nil {
		return nil, err
	}

	// Read modified content
	result, err := os.ReadFile(tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read edited file: %w", err)
	}

	return result, nil
}

// getEditor returns the editor command from environment.
// Checks VISUAL first (for graphical editors), then EDITOR.
func getEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	return os.Getenv("EDITOR")
}

// runEditor executes the editor with the given file path.
func runEditor(editor, path string) error {
	// Split editor into command and args (e.g., "code --wait")
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("empty editor command")
	}

	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("editor exited with status %d", exitErr.ExitCode())
		}
		return fmt.Errorf("failed to run editor: %w", err)
	}

	return nil
}
