package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEditor(t *testing.T) {
	// Save original values
	origVisual := os.Getenv("VISUAL")
	origEditor := os.Getenv("EDITOR")
	defer func() {
		os.Setenv("VISUAL", origVisual)
		os.Setenv("EDITOR", origEditor)
	}()

	// Test VISUAL takes precedence
	os.Setenv("VISUAL", "code --wait")
	os.Setenv("EDITOR", "vim")
	assert.Equal(t, "code --wait", getEditor())

	// Test EDITOR is used when VISUAL is empty
	os.Setenv("VISUAL", "")
	os.Setenv("EDITOR", "vim")
	assert.Equal(t, "vim", getEditor())

	// Test empty when both are unset
	os.Setenv("VISUAL", "")
	os.Setenv("EDITOR", "")
	assert.Equal(t, "", getEditor())
}

func TestEditInEditorNoEditor(t *testing.T) {
	// Save original values
	origVisual := os.Getenv("VISUAL")
	origEditor := os.Getenv("EDITOR")
	defer func() {
		os.Setenv("VISUAL", origVisual)
		os.Setenv("EDITOR", origEditor)
	}()

	os.Setenv("VISUAL", "")
	os.Setenv("EDITOR", "")

	_, err := EditInEditor([]byte("test"), ".yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "EDITOR not set")
}

func TestEditInEditorWithTrueCommand(t *testing.T) {
	// Save original values
	origVisual := os.Getenv("VISUAL")
	origEditor := os.Getenv("EDITOR")
	defer func() {
		os.Setenv("VISUAL", origVisual)
		os.Setenv("EDITOR", origEditor)
	}()

	// Use 'true' command which exists and exits with 0
	// This tests the basic flow without actually editing
	os.Setenv("VISUAL", "")
	os.Setenv("EDITOR", "true")

	content := []byte("test content")
	result, err := EditInEditor(content, ".yaml")
	require.NoError(t, err)
	// Content should be unchanged since 'true' doesn't modify the file
	assert.Equal(t, content, result)
}

func TestEditInEditorNonZeroExit(t *testing.T) {
	// Save original values
	origVisual := os.Getenv("VISUAL")
	origEditor := os.Getenv("EDITOR")
	defer func() {
		os.Setenv("VISUAL", origVisual)
		os.Setenv("EDITOR", origEditor)
	}()

	// Use 'false' command which exits with 1
	os.Setenv("VISUAL", "")
	os.Setenv("EDITOR", "false")

	_, err := EditInEditor([]byte("test"), ".yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "editor exited with status")
}

func TestEditInEditorContentModified(t *testing.T) {
	// Save original values
	origVisual := os.Getenv("VISUAL")
	origEditor := os.Getenv("EDITOR")
	defer func() {
		os.Setenv("VISUAL", origVisual)
		os.Setenv("EDITOR", origEditor)
	}()

	// Create a test script that modifies the file
	script, err := os.CreateTemp("", "test-editor-*.sh")
	require.NoError(t, err)
	defer os.Remove(script.Name())

	// Write a simple script that appends to the file
	_, err = script.WriteString("#!/bin/sh\necho 'modified' > \"$1\"\n")
	require.NoError(t, err)
	script.Close()
	os.Chmod(script.Name(), 0755)

	os.Setenv("VISUAL", "")
	os.Setenv("EDITOR", script.Name())

	content := []byte("original")
	result, err := EditInEditor(content, ".yaml")
	require.NoError(t, err)
	assert.Equal(t, "modified\n", string(result))
}

func TestRunEditorEmptyCommand(t *testing.T) {
	err := runEditor("", "/tmp/test.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty editor command")
}

func TestRunEditorNonExistentCommand(t *testing.T) {
	err := runEditor("nonexistent-editor-command-12345", "/tmp/test.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to run editor")
}
