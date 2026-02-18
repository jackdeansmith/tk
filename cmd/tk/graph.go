package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Generate dependency graph",
	Long: `Generate a DOT format dependency graph.

Output can be piped to graphviz to generate images:
  tk graph | dot -Tpng -o deps.png
  tk graph -p backyard | dot -Tsvg -o backyard.svg

Styling:
- Tasks are boxes, waits are diamonds
- Ready: green, Blocked: red, Waiting: yellow
- Done: gray, Dropped: strikethrough
- Wait dependencies use dashed lines`,
	RunE: runGraph,
}

var graphProject string

func init() {
	graphCmd.Flags().StringVarP(&graphProject, "project", "p", "", "limit to project (prefix or ID)")

	// Register completion function
	graphCmd.RegisterFlagCompletionFunc("project", completeProjectIDs)

	rootCmd.AddCommand(graphCmd)
}

func runGraph(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	ops.AutoCheck(s)

	var projects []*model.ProjectFile
	if graphProject != "" {
		pf, err := ops.ResolveProject(s, graphProject)
		if err != nil {
			return err
		}
		projects = append(projects, pf)
	} else {
		var err error
		projects, err = ops.LoadActiveProjects(s, false)
		if err != nil {
			return err
		}
	}

	// Generate DOT output
	fmt.Println("digraph tk {")
	fmt.Println("  rankdir=LR;")
	fmt.Println("  node [shape=box];")
	fmt.Println()

	now := time.Now()

	for _, pf := range projects {
		blockerStates := ops.ComputeBlockerStates(pf)

		// Output task nodes
		for _, t := range pf.Tasks {
			state := model.ComputeTaskState(&t, blockerStates)
			nodeAttrs := taskNodeAttrs(&t, state)
			fmt.Printf("  %q %s;\n", t.ID, nodeAttrs)
		}

		// Output wait nodes
		for _, w := range pf.Waits {
			state := model.ComputeWaitState(&w, blockerStates, now)
			nodeAttrs := waitNodeAttrs(&w, state)
			fmt.Printf("  %q %s;\n", w.ID, nodeAttrs)
		}

		fmt.Println()

		// Output edges
		for _, t := range pf.Tasks {
			for _, blockerID := range t.BlockedBy {
				edgeStyle := ""
				if model.IsWaitID(blockerID) {
					edgeStyle = " [style=dashed]"
				}
				fmt.Printf("  %q -> %q%s;\n", blockerID, t.ID, edgeStyle)
			}
		}

		for _, w := range pf.Waits {
			for _, blockerID := range w.BlockedBy {
				edgeStyle := " [style=dashed]"
				fmt.Printf("  %q -> %q%s;\n", blockerID, w.ID, edgeStyle)
			}
		}
	}

	fmt.Println("}")
	return nil
}

func taskNodeAttrs(t *model.Task, state model.TaskState) string {
	// Escape title for DOT label
	label := escapeLabel(t.Title)
	if len(label) > 30 {
		label = label[:27] + "..."
	}

	var attrs []string
	attrs = append(attrs, fmt.Sprintf("label=%q", t.ID+"\\n"+label))

	// Style based on state
	switch state {
	case model.TaskStateReady:
		attrs = append(attrs, "style=filled", "fillcolor=palegreen")
	case model.TaskStateBlocked:
		attrs = append(attrs, "style=filled", "fillcolor=lightcoral")
	case model.TaskStateWaiting:
		attrs = append(attrs, "style=filled", "fillcolor=khaki")
	case model.TaskStateDone:
		attrs = append(attrs, "style=filled", "fillcolor=lightgray")
	case model.TaskStateDropped:
		attrs = append(attrs, "style=\"filled,dashed\"", "fillcolor=lightgray", "fontcolor=gray")
	}

	return "[" + strings.Join(attrs, " ") + "]"
}

func waitNodeAttrs(w *model.Wait, state model.WaitState) string {
	// Use display text for label
	label := escapeLabel(w.DisplayText())
	if len(label) > 30 {
		label = label[:27] + "..."
	}

	var attrs []string
	attrs = append(attrs, "shape=diamond")
	attrs = append(attrs, fmt.Sprintf("label=%q", w.ID+"\\n"+label))

	// Style based on state
	switch state {
	case model.WaitStateActionable:
		attrs = append(attrs, "style=filled", "fillcolor=palegreen")
	case model.WaitStatePending:
		attrs = append(attrs, "style=filled", "fillcolor=khaki")
	case model.WaitStateDormant:
		attrs = append(attrs, "style=filled", "fillcolor=lightcoral")
	case model.WaitStateDone:
		attrs = append(attrs, "style=filled", "fillcolor=lightgray")
	case model.WaitStateDropped:
		attrs = append(attrs, "style=\"filled,dashed\"", "fillcolor=lightgray", "fontcolor=gray")
	}

	return "[" + strings.Join(attrs, " ") + "]"
}

func escapeLabel(s string) string {
	// Escape special DOT characters
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
