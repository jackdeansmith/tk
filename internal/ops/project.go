// Package ops provides business logic for modifying tk data.
package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
)

// ProjectChanges represents fields that can be updated on a project.
type ProjectChanges struct {
	Name        *string
	Description *string
	Status      *model.ProjectStatus
}

// CreateProject creates a new project with the given parameters.
func CreateProject(s *storage.Storage, id, prefix, name, description string) error {
	// Normalize prefix to uppercase
	prefix = strings.ToUpper(prefix)

	// Check if prefix is already in use
	if s.ProjectExists(prefix) {
		return fmt.Errorf("project with prefix %q already exists", prefix)
	}

	// Validate prefix format (2-3 uppercase letters)
	if len(prefix) < 2 || len(prefix) > 3 {
		return fmt.Errorf("prefix must be 2-3 characters, got %q", prefix)
	}
	for _, c := range prefix {
		if c < 'A' || c > 'Z' {
			return fmt.Errorf("prefix must contain only letters, got %q", prefix)
		}
	}

	// Derive ID from prefix if not provided
	if id == "" {
		id = strings.ToLower(prefix)
	}
	id = strings.ToLower(id)

	// Check if ID is already in use
	if _, err := s.LoadProjectByID(id); err == nil {
		return fmt.Errorf("project with ID %q already exists", id)
	}

	// Create the project
	now := time.Now()
	project := &model.ProjectFile{
		Project: model.Project{
			ID:          id,
			Prefix:      prefix,
			Name:        name,
			Description: description,
			Status:      model.ProjectStatusActive,
			NextID:      1,
			Created:     now,
		},
	}

	return s.SaveProject(project)
}

// EditProject updates project metadata.
func EditProject(s *storage.Storage, prefix string, changes ProjectChanges) error {
	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	if changes.Name != nil {
		pf.Name = *changes.Name
	}
	if changes.Description != nil {
		pf.Description = *changes.Description
	}
	if changes.Status != nil {
		pf.Status = *changes.Status
	}

	return s.SaveProject(pf)
}

// DeleteProject removes a project and all its tasks/waits.
// If force is false, returns an error if the project has any open tasks or waits.
func DeleteProject(s *storage.Storage, prefix string, force bool) error {
	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	if !force {
		// Check for open tasks
		for _, t := range pf.Tasks {
			if t.Status == model.TaskStatusOpen {
				return fmt.Errorf("project has open tasks (use --force to delete anyway)")
			}
		}
		// Check for open waits
		for _, w := range pf.Waits {
			if w.Status == model.WaitStatusOpen {
				return fmt.Errorf("project has open waits (use --force to delete anyway)")
			}
		}
	}

	return s.DeleteProject(prefix)
}

// ChangeProjectPrefix changes a project's prefix and updates all task/wait IDs.
func ChangeProjectPrefix(s *storage.Storage, oldPrefix, newPrefix string) error {
	oldPrefix = strings.ToUpper(oldPrefix)
	newPrefix = strings.ToUpper(newPrefix)

	if oldPrefix == newPrefix {
		return fmt.Errorf("new prefix is the same as old prefix")
	}

	// Validate new prefix format
	if len(newPrefix) < 2 || len(newPrefix) > 3 {
		return fmt.Errorf("prefix must be 2-3 characters, got %q", newPrefix)
	}
	for _, c := range newPrefix {
		if c < 'A' || c > 'Z' {
			return fmt.Errorf("prefix must contain only letters, got %q", newPrefix)
		}
	}

	// Check if new prefix is in use
	if s.ProjectExists(newPrefix) {
		return fmt.Errorf("project with prefix %q already exists", newPrefix)
	}

	// Load the project
	pf, err := s.LoadProject(oldPrefix)
	if err != nil {
		return err
	}

	// Build a mapping of old IDs to new IDs
	idMap := make(map[string]string)

	// Update task IDs
	maxID := pf.NextID - 1
	for i := range pf.Tasks {
		oldID := pf.Tasks[i].ID
		_, num, _, _ := model.ParseAnyID(oldID)
		newID := model.FormatTaskID(newPrefix, num, maxID)
		idMap[oldID] = newID
		pf.Tasks[i].ID = newID
	}

	// Update wait IDs
	for i := range pf.Waits {
		oldID := pf.Waits[i].ID
		_, num, _, _ := model.ParseAnyID(oldID)
		newID := model.FormatWaitID(newPrefix, num, maxID)
		idMap[oldID] = newID
		pf.Waits[i].ID = newID
	}

	// Update all blocked_by references
	for i := range pf.Tasks {
		pf.Tasks[i].BlockedBy = updateBlockerRefs(pf.Tasks[i].BlockedBy, idMap)
	}
	for i := range pf.Waits {
		pf.Waits[i].BlockedBy = updateBlockerRefs(pf.Waits[i].BlockedBy, idMap)
	}

	// Update the project prefix
	pf.Prefix = newPrefix

	// Save with new prefix (creates new file)
	if err := s.SaveProject(pf); err != nil {
		return err
	}

	// Delete old project file
	return s.DeleteProject(oldPrefix)
}

// updateBlockerRefs updates blocker references using the provided ID mapping.
func updateBlockerRefs(blockedBy []string, idMap map[string]string) []string {
	if len(blockedBy) == 0 {
		return blockedBy
	}

	result := make([]string, len(blockedBy))
	for i, id := range blockedBy {
		if newID, ok := idMap[id]; ok {
			result[i] = newID
		} else {
			result[i] = id
		}
	}
	return result
}
