package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/graph"
	"github.com/jacksmith/tk/internal/model"
)

// ResolveProject resolves a project reference (prefix or ID) to a loaded ProjectFile.
// If ref is empty, uses the default project from config.
func ResolveProject(s Store, ref string) (*model.ProjectFile, error) {
	if ref == "" {
		cfg, err := s.LoadConfig()
		if err != nil {
			return nil, err
		}
		if cfg.DefaultProject == "" {
			return nil, fmt.Errorf("no project specified and no default_project in config")
		}
		pf, err := s.LoadProjectByID(cfg.DefaultProject)
		if err != nil {
			return nil, fmt.Errorf("default project %q not found", cfg.DefaultProject)
		}
		return pf, nil
	}

	pf, err := s.LoadProject(ref)
	if err != nil {
		pf, err = s.LoadProjectByID(ref)
		if err != nil {
			return nil, fmt.Errorf("project %q not found", ref)
		}
	}
	return pf, nil
}

// AutoCheck runs auto-resolution if configured, silently ignoring errors.
func AutoCheck(s Store) {
	cfg, err := s.LoadConfig()
	if err != nil {
		return
	}
	if cfg.AutoCheck {
		_, _ = RunCheck(s)
	}
}

// LoadActiveProjects returns all project files for active projects.
// If includeAll is true, includes paused and done projects too.
func LoadActiveProjects(s Store, includeAll bool) ([]*model.ProjectFile, error) {
	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, err
	}
	var projects []*model.ProjectFile
	for _, prefix := range prefixes {
		pf, err := s.LoadProject(prefix)
		if err != nil {
			continue
		}
		if !includeAll && pf.Status != model.ProjectStatusActive {
			continue
		}
		projects = append(projects, pf)
	}
	return projects, nil
}

// TaskFilter specifies filtering criteria for listing tasks.
type TaskFilter struct {
	Project  string           // Limit to a specific project (prefix or ID). Empty = all active.
	State    *model.TaskState // Filter by derived state. Nil = open tasks only.
	All      bool             // Show all tasks regardless of status.
	Priority int              // Filter by priority (0 = any).
	Tags     []string         // Require all specified tags (AND logic).
	Overdue  bool             // Only tasks with due date in the past.
}

// TaskResult is a single task with its computed state.
type TaskResult struct {
	Task    model.Task
	State   model.TaskState
	Project string // project prefix
}

// ListTasks returns tasks matching the given filter across projects.
func ListTasks(s Store, filter TaskFilter) ([]TaskResult, error) {
	projects, err := resolveProjectsForFilter(s, filter.Project, filter.All)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var results []TaskResult

	for _, pf := range projects {
		blockerStates := ComputeBlockerStates(pf)
		for _, t := range pf.Tasks {
			state := model.ComputeTaskState(&t, blockerStates)
			if !matchesTaskFilter(&t, state, blockerStates, filter, now) {
				continue
			}
			results = append(results, TaskResult{
				Task:    t,
				State:   state,
				Project: pf.Prefix,
			})
		}
	}

	return results, nil
}

func matchesTaskFilter(t *model.Task, state model.TaskState, blockerStates model.BlockerStatus, f TaskFilter, now time.Time) bool {
	// Status/state filter
	if f.State != nil {
		if state != *f.State {
			return false
		}
	} else if !f.All {
		// Default: show only open tasks
		if t.Status != model.TaskStatusOpen {
			return false
		}
	}

	// Priority filter
	if f.Priority > 0 && t.Priority != f.Priority {
		return false
	}

	// Tag filter (AND logic)
	if len(f.Tags) > 0 {
		taskTags := make(map[string]bool)
		for _, tag := range t.Tags {
			taskTags[strings.ToLower(tag)] = true
		}
		for _, requiredTag := range f.Tags {
			if !taskTags[strings.ToLower(requiredTag)] {
				return false
			}
		}
	}

	// Overdue filter
	if f.Overdue {
		if t.DueDate == nil || !t.DueDate.Before(now) {
			return false
		}
	}

	return true
}

// WaitFilter specifies filtering criteria for listing waits.
type WaitFilter struct {
	Project string           // Limit to a specific project (prefix or ID). Empty = all active.
	State   *model.WaitState // Filter by derived state. Nil = open waits only.
	All     bool             // Show all waits regardless of status.
}

// WaitResult is a single wait with its computed state.
type WaitResult struct {
	Wait    model.Wait
	State   model.WaitState
	Project string
}

// ListWaits returns waits matching the given filter across projects.
func ListWaits(s Store, filter WaitFilter) ([]WaitResult, error) {
	projects, err := resolveProjectsForFilter(s, filter.Project, filter.All)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var results []WaitResult

	for _, pf := range projects {
		blockerStates := ComputeBlockerStates(pf)
		for _, w := range pf.Waits {
			state := model.ComputeWaitState(&w, blockerStates, now)
			if !matchesWaitFilter(&w, state, filter) {
				continue
			}
			results = append(results, WaitResult{
				Wait:    w,
				State:   state,
				Project: pf.Prefix,
			})
		}
	}

	return results, nil
}

func matchesWaitFilter(w *model.Wait, state model.WaitState, f WaitFilter) bool {
	if f.State != nil {
		return state == *f.State
	}
	if f.All {
		return true
	}
	// Default: open waits only
	return w.Status == model.WaitStatusOpen
}

// ShowTask loads a single task by ID with its computed state.
func ShowTask(s Store, taskID string) (*TaskResult, *model.ProjectFile, error) {
	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return nil, nil, fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("project with prefix %q not found", prefix)
	}

	normalizedID := strings.ToUpper(taskID)
	for i := range pf.Tasks {
		if strings.ToUpper(pf.Tasks[i].ID) == normalizedID {
			blockerStates := ComputeBlockerStates(pf)
			state := model.ComputeTaskState(&pf.Tasks[i], blockerStates)
			return &TaskResult{Task: pf.Tasks[i], State: state, Project: pf.Prefix}, pf, nil
		}
		// Also try matching without leading zeros
		_, num, err2 := model.ParseTaskID(pf.Tasks[i].ID)
		if err2 == nil {
			_, searchNum, err3 := model.ParseTaskID(taskID)
			if err3 == nil && num == searchNum {
				blockerStates := ComputeBlockerStates(pf)
				state := model.ComputeTaskState(&pf.Tasks[i], blockerStates)
				return &TaskResult{Task: pf.Tasks[i], State: state, Project: pf.Prefix}, pf, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("task %s not found", taskID)
}

// ShowWait loads a single wait by ID with its computed state.
func ShowWait(s Store, waitID string) (*WaitResult, *model.ProjectFile, error) {
	prefix := model.ExtractPrefix(waitID)
	if prefix == "" {
		return nil, nil, fmt.Errorf("invalid wait ID: %s", waitID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, nil, fmt.Errorf("project with prefix %q not found", prefix)
	}

	now := time.Now()
	normalizedID := strings.ToUpper(waitID)
	for i := range pf.Waits {
		if strings.ToUpper(pf.Waits[i].ID) == normalizedID {
			blockerStates := ComputeBlockerStates(pf)
			state := model.ComputeWaitState(&pf.Waits[i], blockerStates, now)
			return &WaitResult{Wait: pf.Waits[i], State: state, Project: pf.Prefix}, pf, nil
		}
		// Also try matching without leading zeros
		_, num, err2 := model.ParseWaitID(pf.Waits[i].ID)
		if err2 == nil {
			_, searchNum, err3 := model.ParseWaitID(waitID)
			if err3 == nil && num == searchNum {
				blockerStates := ComputeBlockerStates(pf)
				state := model.ComputeWaitState(&pf.Waits[i], blockerStates, now)
				return &WaitResult{Wait: pf.Waits[i], State: state, Project: pf.Prefix}, pf, nil
			}
		}
	}

	return nil, nil, fmt.Errorf("wait %s not found", waitID)
}

// BlockerInfo describes a blocker item for display purposes.
type BlockerInfo struct {
	ID          string
	Status      string
	DisplayText string
}

// GetBlockerInfo returns display information for a blocker by ID.
func GetBlockerInfo(pf *model.ProjectFile, blockerID string) BlockerInfo {
	normalizedID := strings.ToUpper(blockerID)

	for _, t := range pf.Tasks {
		if strings.ToUpper(t.ID) == normalizedID {
			return BlockerInfo{ID: t.ID, Status: string(t.Status), DisplayText: t.Title}
		}
	}
	for _, w := range pf.Waits {
		if strings.ToUpper(w.ID) == normalizedID {
			return BlockerInfo{ID: w.ID, Status: string(w.Status), DisplayText: w.DisplayText()}
		}
	}

	return BlockerInfo{ID: blockerID, Status: "unknown", DisplayText: ""}
}

// GetBlockers returns the blockers for an item (task or wait) by ID.
func GetBlockers(s Store, id string) ([]string, error) {
	prefix := model.ExtractPrefix(id)
	if prefix == "" {
		return nil, fmt.Errorf("invalid ID format: %s", id)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	if model.IsWaitID(id) {
		for _, w := range pf.Waits {
			if strings.EqualFold(w.ID, id) {
				return w.BlockedBy, nil
			}
		}
	} else {
		for _, t := range pf.Tasks {
			if strings.EqualFold(t.ID, id) {
				return t.BlockedBy, nil
			}
		}
	}

	return nil, nil
}

// GetBlocking returns the items that an item is blocking (direct dependents).
func GetBlocking(s Store, id string) ([]string, error) {
	prefix := model.ExtractPrefix(id)
	if prefix == "" {
		return nil, fmt.Errorf("invalid ID format: %s", id)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return nil, err
	}

	g := graph.BuildGraph(pf)
	return g.Blocking(id), nil
}

// FindResult holds a search match.
type FindResult struct {
	Tasks []TaskResult
	Waits []WaitResult
}

// FindItems searches tasks and waits by keyword across projects.
func FindItems(s Store, query string, projectRef string) (*FindResult, error) {
	var projects []*model.ProjectFile

	if projectRef != "" {
		pf, err := ResolveProject(s, projectRef)
		if err != nil {
			return nil, err
		}
		projects = append(projects, pf)
	} else {
		var err error
		projects, err = LoadActiveProjects(s, false)
		if err != nil {
			return nil, err
		}
	}

	queryLower := strings.ToLower(query)
	now := time.Now()
	result := &FindResult{}

	for _, pf := range projects {
		blockerStates := ComputeBlockerStates(pf)

		for _, t := range pf.Tasks {
			if strings.Contains(strings.ToLower(t.Title), queryLower) ||
				strings.Contains(strings.ToLower(t.Notes), queryLower) {
				state := model.ComputeTaskState(&t, blockerStates)
				result.Tasks = append(result.Tasks, TaskResult{Task: t, State: state, Project: pf.Prefix})
			}
		}

		for _, w := range pf.Waits {
			if strings.Contains(strings.ToLower(w.Title), queryLower) ||
				strings.Contains(strings.ToLower(w.ResolutionCriteria.Question), queryLower) ||
				strings.Contains(strings.ToLower(w.Notes), queryLower) {
				state := model.ComputeWaitState(&w, blockerStates, now)
				result.Waits = append(result.Waits, WaitResult{Wait: w, State: state, Project: pf.Prefix})
			}
		}
	}

	return result, nil
}

// ProjectSummary holds computed counts for a project.
type ProjectSummary struct {
	Project     model.Project
	OpenCount   int
	ReadyCount  int
	BlockedCount int
	WaitingCount int
	DoneCount   int
	DroppedCount int
	OpenWaits   int
	DoneWaits   int
	DroppedWaits int
}

// GetProjectSummary computes task/wait summary for a project.
func GetProjectSummary(s Store, projectRef string) (*ProjectSummary, error) {
	pf, err := ResolveProject(s, projectRef)
	if err != nil {
		return nil, err
	}

	blockerStates := ComputeBlockerStates(pf)
	summary := &ProjectSummary{Project: pf.Project}

	for _, t := range pf.Tasks {
		switch t.Status {
		case model.TaskStatusDone:
			summary.DoneCount++
		case model.TaskStatusDropped:
			summary.DroppedCount++
		case model.TaskStatusOpen:
			summary.OpenCount++
			state := model.ComputeTaskState(&t, blockerStates)
			switch state {
			case model.TaskStateReady:
				summary.ReadyCount++
			case model.TaskStateBlocked:
				summary.BlockedCount++
			case model.TaskStateWaiting:
				summary.WaitingCount++
			}
		}
	}

	for _, w := range pf.Waits {
		switch w.Status {
		case model.WaitStatusOpen:
			summary.OpenWaits++
		case model.WaitStatusDone:
			summary.DoneWaits++
		case model.WaitStatusDropped:
			summary.DroppedWaits++
		}
	}

	return summary, nil
}

// ProjectInfo is a loaded project with its status, for listing.
type ProjectInfo struct {
	Prefix string
	Name   string
	Status model.ProjectStatus
}

// ListProjects returns all projects, optionally including non-active ones.
func ListProjectInfos(s Store, includeAll bool) ([]ProjectInfo, error) {
	prefixes, err := s.ListProjects()
	if err != nil {
		return nil, err
	}

	var results []ProjectInfo
	for _, prefix := range prefixes {
		pf, err := s.LoadProject(prefix)
		if err != nil {
			continue
		}
		if !includeAll && pf.Status != model.ProjectStatusActive {
			continue
		}
		results = append(results, ProjectInfo{
			Prefix: pf.Prefix,
			Name:   pf.Name,
			Status: pf.Status,
		})
	}
	return results, nil
}

// AddTag adds a tag to a task. Returns true if the tag was added, false if already present.
func AddTag(s Store, taskID string, tag string) (bool, error) {
	if err := ValidateTag(tag); err != nil {
		return false, err
	}

	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return false, fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return false, err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return false, fmt.Errorf("task %s not found", taskID)
	}

	// Check if tag already exists
	for _, t := range task.Tags {
		if strings.EqualFold(t, tag) {
			return false, nil
		}
	}

	newTags := append(task.Tags, tag)
	changes := TaskChanges{Tags: &newTags}
	if err := EditTask(s, taskID, changes); err != nil {
		return false, err
	}
	return true, nil
}

// RemoveTag removes a tag from a task. Returns true if removed, false if not found.
func RemoveTag(s Store, taskID string, tag string) (bool, error) {
	if err := ValidateTag(tag); err != nil {
		return false, err
	}

	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return false, fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return false, err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return false, fmt.Errorf("task %s not found", taskID)
	}

	found := false
	var newTags []string
	for _, t := range task.Tags {
		if strings.EqualFold(t, tag) {
			found = true
		} else {
			newTags = append(newTags, t)
		}
	}

	if !found {
		return false, nil
	}

	changes := TaskChanges{Tags: &newTags}
	if err := EditTask(s, taskID, changes); err != nil {
		return false, err
	}
	return true, nil
}

// AppendNote appends text to a task's notes field.
func AppendNote(s Store, taskID string, text string) error {
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("note text must not be empty")
	}

	prefix := model.ExtractPrefix(taskID)
	if prefix == "" {
		return fmt.Errorf("invalid task ID: %s", taskID)
	}

	pf, err := s.LoadProject(prefix)
	if err != nil {
		return err
	}

	task := findTask(pf, taskID)
	if task == nil {
		return fmt.Errorf("task %s not found", taskID)
	}

	var newNotes string
	if task.Notes == "" {
		newNotes = text
	} else {
		newNotes = task.Notes + "\n" + text
	}

	changes := TaskChanges{Notes: &newNotes}
	return EditTask(s, taskID, changes)
}

// resolveProjectsForFilter loads the projects applicable to a query filter.
func resolveProjectsForFilter(s Store, projectRef string, includeAll bool) ([]*model.ProjectFile, error) {
	if projectRef != "" {
		pf, err := ResolveProject(s, projectRef)
		if err != nil {
			return nil, err
		}
		return []*model.ProjectFile{pf}, nil
	}
	return LoadActiveProjects(s, includeAll)
}
