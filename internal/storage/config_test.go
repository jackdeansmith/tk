package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Run("no .tkconfig.yaml returns defaults", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		cfg, err := s.LoadConfig()
		require.NoError(t, err)

		assert.Equal(t, DefaultAutoCheck, cfg.AutoCheck)
		assert.Equal(t, DefaultDefaultProject, cfg.DefaultProject)
		assert.Equal(t, DefaultDefaultPriority, cfg.DefaultPriority)
	})

	t.Run("full .tkconfig.yaml loads all values", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create config file
		configContent := `autocheck: true
default_project: backyard
default_priority: 1
`
		configPath := filepath.Join(dir, ".tkconfig.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := s.LoadConfig()
		require.NoError(t, err)

		assert.True(t, cfg.AutoCheck)
		assert.Equal(t, "backyard", cfg.DefaultProject)
		assert.Equal(t, 1, cfg.DefaultPriority)
	})

	t.Run("partial .tkconfig.yaml merges with defaults", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create partial config file (only autocheck)
		configContent := `autocheck: true
`
		configPath := filepath.Join(dir, ".tkconfig.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := s.LoadConfig()
		require.NoError(t, err)

		assert.True(t, cfg.AutoCheck)
		assert.Equal(t, DefaultDefaultProject, cfg.DefaultProject)   // default
		assert.Equal(t, DefaultDefaultPriority, cfg.DefaultPriority) // default
	})

	t.Run("partial config with only default_project", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		configContent := `default_project: myproject
`
		configPath := filepath.Join(dir, ".tkconfig.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := s.LoadConfig()
		require.NoError(t, err)

		assert.Equal(t, DefaultAutoCheck, cfg.AutoCheck)
		assert.Equal(t, "myproject", cfg.DefaultProject)
		assert.Equal(t, DefaultDefaultPriority, cfg.DefaultPriority)
	})

	t.Run("partial config with only default_priority", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		configContent := `default_priority: 2
`
		configPath := filepath.Join(dir, ".tkconfig.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		cfg, err := s.LoadConfig()
		require.NoError(t, err)

		assert.Equal(t, DefaultAutoCheck, cfg.AutoCheck)
		assert.Equal(t, DefaultDefaultProject, cfg.DefaultProject)
		assert.Equal(t, 2, cfg.DefaultPriority)
	})

	t.Run("invalid YAML returns error with filename", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create invalid config file
		configContent := `autocheck: [invalid yaml
this is not valid
`
		configPath := filepath.Join(dir, ".tkconfig.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		_, err = s.LoadConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), ".tkconfig.yaml")
	})

	t.Run("empty .tkconfig.yaml returns defaults", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		// Create empty config file
		configPath := filepath.Join(dir, ".tkconfig.yaml")
		err = os.WriteFile(configPath, []byte(""), 0644)
		require.NoError(t, err)

		cfg, err := s.LoadConfig()
		require.NoError(t, err)

		assert.Equal(t, DefaultAutoCheck, cfg.AutoCheck)
		assert.Equal(t, DefaultDefaultProject, cfg.DefaultProject)
		assert.Equal(t, DefaultDefaultPriority, cfg.DefaultPriority)
	})
}

func TestDefaultConfig(t *testing.T) {
	t.Run("returns expected defaults", func(t *testing.T) {
		cfg := DefaultConfig()

		assert.False(t, cfg.AutoCheck)
		assert.Equal(t, "default", cfg.DefaultProject)
		assert.Equal(t, 3, cfg.DefaultPriority)
	})
}

func TestConfigPath(t *testing.T) {
	t.Run("returns correct path", func(t *testing.T) {
		dir := t.TempDir()
		s, err := Init(dir, "", "")
		require.NoError(t, err)

		expected := filepath.Join(dir, ".tkconfig.yaml")
		assert.Equal(t, expected, s.ConfigPath())
	})
}
