package model

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadProject loads a project file from the given path.
func LoadProject(path string) (*ProjectFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read project file %s: %w", path, err)
	}

	var pf ProjectFile
	if err := yaml.Unmarshal(data, &pf); err != nil {
		return nil, fmt.Errorf("failed to parse project file %s: %w", path, err)
	}

	return &pf, nil
}

// SaveProject saves a project file to the given path.
// Tasks and waits are sorted by numeric ID.
// Null fields are omitted.
// Multi-line strings use block scalar style.
func SaveProject(path string, p *ProjectFile) error {
	// Sort tasks and waits by numeric ID before saving
	sortTasks(p.Tasks)
	sortWaits(p.Waits)

	// Build the YAML document with proper formatting
	node, err := buildProjectNode(p)
	if err != nil {
		return fmt.Errorf("failed to build YAML: %w", err)
	}

	// Encode to YAML
	data, err := yaml.Marshal(node)
	if err != nil {
		return fmt.Errorf("failed to encode project: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write project file %s: %w", path, err)
	}

	return nil
}

// sortTasks sorts tasks by their numeric ID.
func sortTasks(tasks []Task) {
	sort.Slice(tasks, func(i, j int) bool {
		return ExtractNumber(tasks[i].ID) < ExtractNumber(tasks[j].ID)
	})
}

// sortWaits sorts waits by their numeric ID.
func sortWaits(waits []Wait) {
	sort.Slice(waits, func(i, j int) bool {
		return ExtractNumber(waits[i].ID) < ExtractNumber(waits[j].ID)
	})
}

// buildProjectNode creates a yaml.Node tree for a ProjectFile with proper formatting.
func buildProjectNode(p *ProjectFile) (*yaml.Node, error) {
	doc := &yaml.Node{
		Kind: yaml.MappingNode,
	}

	// Project metadata fields
	addStringField(doc, "id", p.ID)
	addStringField(doc, "prefix", p.Prefix)
	addStringField(doc, "name", p.Name)
	if p.Description != "" {
		addStringField(doc, "description", p.Description)
	}
	addStringField(doc, "status", string(p.Status))
	addIntField(doc, "next_id", p.NextID)
	addTimeField(doc, "created", p.Created)

	// Tasks
	if len(p.Tasks) > 0 {
		tasksNode := &yaml.Node{Kind: yaml.SequenceNode}
		for _, task := range p.Tasks {
			taskNode, err := buildTaskNode(&task)
			if err != nil {
				return nil, err
			}
			tasksNode.Content = append(tasksNode.Content, taskNode)
		}
		doc.Content = append(doc.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "tasks"},
			tasksNode,
		)
	}

	// Waits
	if len(p.Waits) > 0 {
		waitsNode := &yaml.Node{Kind: yaml.SequenceNode}
		for _, wait := range p.Waits {
			waitNode, err := buildWaitNode(&wait)
			if err != nil {
				return nil, err
			}
			waitsNode.Content = append(waitsNode.Content, waitNode)
		}
		doc.Content = append(doc.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "waits"},
			waitsNode,
		)
	}

	return doc, nil
}

// buildTaskNode creates a yaml.Node for a Task.
func buildTaskNode(t *Task) (*yaml.Node, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	addStringField(node, "id", t.ID)
	addStringField(node, "title", t.Title)
	addStringField(node, "status", string(t.Status))
	addIntField(node, "priority", t.Priority)

	if len(t.BlockedBy) > 0 {
		addStringSliceField(node, "blocked_by", t.BlockedBy)
	}
	if len(t.Tags) > 0 {
		addStringSliceField(node, "tags", t.Tags)
	}
	if t.Notes != "" {
		addMultilineStringField(node, "notes", t.Notes)
	}
	if t.Assignee != "" {
		addStringField(node, "assignee", t.Assignee)
	}
	if t.DueDate != nil {
		addDateField(node, "due_date", *t.DueDate)
	}
	if t.AutoComplete {
		addBoolField(node, "auto_complete", t.AutoComplete)
	}

	addTimeField(node, "created", t.Created)
	addTimeField(node, "updated", t.Updated)

	if t.DoneAt != nil {
		addTimeField(node, "done_at", *t.DoneAt)
	}
	if t.DroppedAt != nil {
		addTimeField(node, "dropped_at", *t.DroppedAt)
	}
	if t.DropReason != "" {
		addStringField(node, "drop_reason", t.DropReason)
	}

	return node, nil
}

// buildWaitNode creates a yaml.Node for a Wait.
func buildWaitNode(w *Wait) (*yaml.Node, error) {
	node := &yaml.Node{Kind: yaml.MappingNode}

	addStringField(node, "id", w.ID)
	if w.Title != "" {
		addStringField(node, "title", w.Title)
	}
	addStringField(node, "status", string(w.Status))

	// Resolution criteria
	rcNode := &yaml.Node{Kind: yaml.MappingNode}
	addStringField(rcNode, "type", string(w.ResolutionCriteria.Type))
	if w.ResolutionCriteria.Question != "" {
		addStringField(rcNode, "question", w.ResolutionCriteria.Question)
	}
	if w.ResolutionCriteria.After != nil {
		addTimeField(rcNode, "after", *w.ResolutionCriteria.After)
	}
	if w.ResolutionCriteria.CheckAfter != nil {
		addTimeField(rcNode, "check_after", *w.ResolutionCriteria.CheckAfter)
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "resolution_criteria"},
		rcNode,
	)

	if len(w.BlockedBy) > 0 {
		addStringSliceField(node, "blocked_by", w.BlockedBy)
	}
	if w.Notes != "" {
		addMultilineStringField(node, "notes", w.Notes)
	}
	if w.Resolution != "" {
		addStringField(node, "resolution", w.Resolution)
	}

	addTimeField(node, "created", w.Created)

	if w.DoneAt != nil {
		addTimeField(node, "done_at", *w.DoneAt)
	}
	if w.DroppedAt != nil {
		addTimeField(node, "dropped_at", *w.DroppedAt)
	}
	if w.DropReason != "" {
		addStringField(node, "drop_reason", w.DropReason)
	}

	return node, nil
}

// Helper functions for building yaml.Node

func addStringField(node *yaml.Node, key, value string) {
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

func addIntField(node *yaml.Node, key string, value int) {
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%d", value), Tag: "!!int"},
	)
}

func addBoolField(node *yaml.Node, key string, value bool) {
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprintf("%t", value), Tag: "!!bool"},
	)
}

func addTimeField(node *yaml.Node, key string, t time.Time) {
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: t.Format(time.RFC3339)},
	)
}

func addDateField(node *yaml.Node, key string, t time.Time) {
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: t.Format("2006-01-02")},
	)
}

func addStringSliceField(node *yaml.Node, key string, values []string) {
	seqNode := &yaml.Node{Kind: yaml.SequenceNode, Style: yaml.FlowStyle}
	for _, v := range values {
		seqNode.Content = append(seqNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: v},
		)
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		seqNode,
	)
}

func addMultilineStringField(node *yaml.Node, key, value string) {
	// Use literal block scalar style for multi-line strings
	style := yaml.LiteralStyle
	if !containsNewline(value) {
		style = 0 // Use default style for single-line
	}
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value, Style: style},
	)
}

func containsNewline(s string) bool {
	return strings.Contains(s, "\n")
}
