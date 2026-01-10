// Package model defines the core data structures for tk.
package model

import "time"

// ProjectStatus represents the status of a project.
type ProjectStatus string

const (
	ProjectStatusActive ProjectStatus = "active"
	ProjectStatusPaused ProjectStatus = "paused"
	ProjectStatusDone   ProjectStatus = "done"
)

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskStatusOpen    TaskStatus = "open"
	TaskStatusDone    TaskStatus = "done"
	TaskStatusDropped TaskStatus = "dropped"
)

// WaitStatus represents the status of a wait (same as TaskStatus).
type WaitStatus string

const (
	WaitStatusOpen    WaitStatus = "open"
	WaitStatusDone    WaitStatus = "done"
	WaitStatusDropped WaitStatus = "dropped"
)

// ResolutionType represents the type of resolution criteria for a wait.
type ResolutionType string

const (
	ResolutionTypeTime   ResolutionType = "time"
	ResolutionTypeManual ResolutionType = "manual"
)

// ResolutionCriteria defines how a wait can be resolved.
type ResolutionCriteria struct {
	Type       ResolutionType `yaml:"type"`
	Question   string         `yaml:"question,omitempty"`    // For manual waits
	After      *time.Time     `yaml:"after,omitempty"`       // For time waits
	CheckAfter *time.Time     `yaml:"check_after,omitempty"` // For manual waits (optional)
}

// Project represents a container for related tasks.
type Project struct {
	ID          string        `yaml:"id"`
	Prefix      string        `yaml:"prefix"`
	Name        string        `yaml:"name"`
	Description string        `yaml:"description,omitempty"`
	Status      ProjectStatus `yaml:"status"`
	NextID      int           `yaml:"next_id"`
	Created     time.Time     `yaml:"created"`
}

// Task represents a unit of work that can be completed.
type Task struct {
	ID           string     `yaml:"id"`
	Title        string     `yaml:"title"`
	Status       TaskStatus `yaml:"status"`
	Priority     int        `yaml:"priority"`
	BlockedBy    []string   `yaml:"blocked_by,omitempty"`
	Tags         []string   `yaml:"tags,omitempty"`
	Notes        string     `yaml:"notes,omitempty"`
	Assignee     string     `yaml:"assignee,omitempty"`
	DueDate      *time.Time `yaml:"due_date,omitempty"`
	AutoComplete bool       `yaml:"auto_complete,omitempty"`
	Created      time.Time  `yaml:"created"`
	Updated      time.Time  `yaml:"updated"`
	DoneAt       *time.Time `yaml:"done_at,omitempty"`
	DroppedAt    *time.Time `yaml:"dropped_at,omitempty"`
	DropReason   string     `yaml:"drop_reason,omitempty"`
}

// Wait represents an external condition that blocks one or more tasks.
type Wait struct {
	ID                 string             `yaml:"id"`
	Title              string             `yaml:"title,omitempty"`
	Status             WaitStatus         `yaml:"status"`
	ResolutionCriteria ResolutionCriteria `yaml:"resolution_criteria"`
	BlockedBy          []string           `yaml:"blocked_by,omitempty"`
	Notes              string             `yaml:"notes,omitempty"`
	Resolution         string             `yaml:"resolution,omitempty"`
	Created            time.Time          `yaml:"created"`
	DoneAt             *time.Time         `yaml:"done_at,omitempty"`
	DroppedAt          *time.Time         `yaml:"dropped_at,omitempty"`
	DropReason         string             `yaml:"drop_reason,omitempty"`
}

// ProjectFile represents a complete project file with all tasks and waits.
type ProjectFile struct {
	Project `yaml:",inline"`
	Tasks   []Task `yaml:"tasks,omitempty"`
	Waits   []Wait `yaml:"waits,omitempty"`
}

// DisplayText returns the text to display for a wait in list views.
// If title exists, use it. Otherwise, for time waits show "Until {date}",
// for manual waits show the question.
func (w *Wait) DisplayText() string {
	if w.Title != "" {
		return w.Title
	}
	if w.ResolutionCriteria.Type == ResolutionTypeTime && w.ResolutionCriteria.After != nil {
		return "Until " + w.ResolutionCriteria.After.Format("2006-01-02")
	}
	if w.ResolutionCriteria.Type == ResolutionTypeManual {
		return w.ResolutionCriteria.Question
	}
	return ""
}
