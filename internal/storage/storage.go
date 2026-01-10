// Package storage provides file system operations for .tk/ directories.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"gopkg.in/yaml.v3"
)

const (
	// tkDir is the name of the tk directory.
	tkDir = ".tk"
	// projectsDir is the subdirectory for project files.
	projectsDir = "projects"
	// configFile is the name of the config file within .tk/.
	configFile = "config.yaml"
)

// StorageConfig contains settings stored in .tk/config.yaml.
type StorageConfig struct {
	Version int `yaml:"version"`
}

// Storage provides access to a .tk/ directory.
type Storage struct {
	root string // path to directory containing .tk/
}

// Open returns a Storage for the given directory.
// Returns error if .tk/ does not exist.
func Open(dir string) (*Storage, error) {
	tkPath := filepath.Join(dir, tkDir)
	info, err := os.Stat(tkPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(".tk/ directory not found in %s", dir)
		}
		return nil, fmt.Errorf("failed to access .tk/: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf(".tk is not a directory")
	}

	return &Storage{root: dir}, nil
}

// Init creates .tk/ directory with a default project.
// Returns error if .tk/ already exists.
func Init(dir string, projectName string, prefix string) (*Storage, error) {
	tkPath := filepath.Join(dir, tkDir)

	// Check if .tk/ already exists
	if _, err := os.Stat(tkPath); err == nil {
		return nil, fmt.Errorf(".tk/ directory already exists in %s", dir)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to check for .tk/: %w", err)
	}

	// Set defaults if not provided
	if projectName == "" {
		projectName = "Default"
	}
	if prefix == "" {
		prefix = "DF"
	}

	// Normalize prefix to uppercase
	prefix = strings.ToUpper(prefix)

	// Create directory structure
	projectsPath := filepath.Join(tkPath, projectsDir)
	if err := os.MkdirAll(projectsPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .tk/projects/: %w", err)
	}

	// Create config.yaml
	cfg := StorageConfig{Version: 1}
	cfgData, err := yaml.Marshal(&cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}
	cfgPath := filepath.Join(tkPath, configFile)
	if err := os.WriteFile(cfgPath, cfgData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write config.yaml: %w", err)
	}

	// Create default project
	now := time.Now()
	project := &model.ProjectFile{
		Project: model.Project{
			ID:      "default",
			Prefix:  prefix,
			Name:    projectName,
			Status:  model.ProjectStatusActive,
			NextID:  1,
			Created: now,
		},
	}

	s := &Storage{root: dir}
	if err := s.SaveProject(project); err != nil {
		// Clean up on failure
		os.RemoveAll(tkPath)
		return nil, fmt.Errorf("failed to create default project: %w", err)
	}

	return s, nil
}

// Root returns the root directory containing .tk/.
func (s *Storage) Root() string {
	return s.root
}

// TkPath returns the path to the .tk/ directory.
func (s *Storage) TkPath() string {
	return filepath.Join(s.root, tkDir)
}

// projectPath returns the path to a project file by prefix.
func (s *Storage) projectPath(prefix string) string {
	return filepath.Join(s.root, tkDir, projectsDir, strings.ToUpper(prefix)+".yaml")
}

// LoadProject loads a project by prefix (e.g., "BY").
// Prefix lookup is case-insensitive.
func (s *Storage) LoadProject(prefix string) (*model.ProjectFile, error) {
	path := s.projectPath(prefix)

	// Check if file exists first to give a clearer error message
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("project with prefix %q not found", strings.ToUpper(prefix))
		}
		return nil, fmt.Errorf("failed to access project file: %w", err)
	}

	pf, err := model.LoadProject(path)
	if err != nil {
		return nil, err
	}
	return pf, nil
}

// LoadProjectByID loads a project by its ID (e.g., "backyard").
// This requires scanning all project files to find a match.
func (s *Storage) LoadProjectByID(id string) (*model.ProjectFile, error) {
	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, err
	}

	id = strings.ToLower(id)
	for _, prefix := range prefixes {
		pf, err := s.LoadProject(prefix)
		if err != nil {
			continue // Skip files that can't be loaded
		}
		if strings.ToLower(pf.ID) == id {
			return pf, nil
		}
	}

	return nil, fmt.Errorf("project with id %q not found", id)
}

// SaveProject saves a project file.
// The file is saved to .tk/projects/{PREFIX}.yaml where PREFIX is uppercase.
func (s *Storage) SaveProject(p *model.ProjectFile) error {
	path := s.projectPath(p.Prefix)
	return model.SaveProject(path, p)
}

// ListProjects returns all project prefixes.
// Prefixes are returned in uppercase.
func (s *Storage) ListProjects() ([]string, error) {
	projectsPath := filepath.Join(s.root, tkDir, projectsDir)
	entries, err := os.ReadDir(projectsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read projects directory: %w", err)
	}

	var prefixes []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		prefix := strings.TrimSuffix(name, ".yaml")
		prefixes = append(prefixes, prefix)
	}

	return prefixes, nil
}

// DeleteProject removes a project file.
// Prefix lookup is case-insensitive.
func (s *Storage) DeleteProject(prefix string) error {
	path := s.projectPath(prefix)
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("project with prefix %q not found", strings.ToUpper(prefix))
		}
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// ProjectExists checks if a prefix is in use.
// Prefix lookup is case-insensitive.
func (s *Storage) ProjectExists(prefix string) bool {
	path := s.projectPath(prefix)
	_, err := os.Stat(path)
	return err == nil
}
