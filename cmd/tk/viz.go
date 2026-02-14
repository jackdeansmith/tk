package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/ops"
	"github.com/jacksmith/tk/internal/storage"
	"github.com/spf13/cobra"
)

//go:embed viz.html
var vizTemplate string

var vizCmd = &cobra.Command{
	Use:   "viz",
	Short: "Open interactive dependency graph in browser",
	Long: `Open an interactive, force-directed dependency graph in your browser.

Nodes represent tasks (circles) and waits (diamonds), colored by state:
  Ready/Actionable: green, Blocked/Dormant: red, Waiting/Pending: amber
  Done: gray, Dropped: dark gray

Interactions:
  - Drag nodes to rearrange
  - Scroll to zoom, drag background to pan
  - Hover a node to highlight connections and see details
  - Click a node to pin/unpin its position

By default, done and dropped items are hidden. Use --include-done to show them.`,
	RunE: runViz,
}

var (
	vizProject     string
	vizIncludeDone bool
)

func init() {
	vizCmd.Flags().StringVarP(&vizProject, "project", "p", "", "limit to project (prefix or ID)")
	vizCmd.Flags().BoolVar(&vizIncludeDone, "include-done", false, "include done/dropped items")
	vizCmd.RegisterFlagCompletionFunc("project", completeProjectIDs)
	rootCmd.AddCommand(vizCmd)
}

type vizNode struct {
	ID       string   `json:"id"`
	Label    string   `json:"label"`
	Type     string   `json:"type"`
	State    string   `json:"state"`
	Priority int      `json:"priority,omitempty"`
	Project  string   `json:"project"`
	Tags     []string `json:"tags,omitempty"`
}

type vizEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Dashed bool   `json:"dashed"`
}

type vizData struct {
	Nodes []vizNode `json:"nodes"`
	Edges []vizEdge `json:"edges"`
}

func runViz(cmd *cobra.Command, args []string) error {
	s, err := storage.Open(".")
	if err != nil {
		return err
	}

	cfg, err := s.LoadConfig()
	if err != nil {
		return err
	}
	if cfg.AutoCheck {
		_, _ = ops.RunCheck(s)
	}

	var projects []*model.ProjectFile

	if vizProject != "" {
		pf, err := s.LoadProject(vizProject)
		if err != nil {
			pf, err = s.LoadProjectByID(vizProject)
			if err != nil {
				return fmt.Errorf("project %q not found", vizProject)
			}
		}
		projects = append(projects, pf)
	} else {
		prefixes, err := s.ListProjects()
		if err != nil {
			return err
		}
		for _, prefix := range prefixes {
			pf, err := s.LoadProject(prefix)
			if err != nil {
				continue
			}
			if pf.Status == model.ProjectStatusActive {
				projects = append(projects, pf)
			}
		}
	}

	now := time.Now()
	gd := vizData{}
	nodeSet := make(map[string]bool)

	for _, pf := range projects {
		blockerStates := computeBlockerStates(pf)

		for _, t := range pf.Tasks {
			state := model.ComputeTaskState(&t, blockerStates)
			if !vizIncludeDone && (state == model.TaskStateDone || state == model.TaskStateDropped) {
				continue
			}
			nodeSet[t.ID] = true
			gd.Nodes = append(gd.Nodes, vizNode{
				ID:       t.ID,
				Label:    t.Title,
				Type:     "task",
				State:    string(state),
				Priority: t.Priority,
				Project:  pf.Prefix,
				Tags:     t.Tags,
			})
		}

		for _, w := range pf.Waits {
			state := model.ComputeWaitState(&w, blockerStates, now)
			if !vizIncludeDone && (state == model.WaitStateDone || state == model.WaitStateDropped) {
				continue
			}
			nodeSet[w.ID] = true
			gd.Nodes = append(gd.Nodes, vizNode{
				ID:      w.ID,
				Label:   w.DisplayText(),
				Type:    "wait",
				State:   string(state),
				Project: pf.Prefix,
			})
		}

		for _, t := range pf.Tasks {
			if !nodeSet[t.ID] {
				continue
			}
			for _, blockerID := range t.BlockedBy {
				if !nodeSet[blockerID] {
					continue
				}
				gd.Edges = append(gd.Edges, vizEdge{
					Source: blockerID,
					Target: t.ID,
					Dashed: model.IsWaitID(blockerID),
				})
			}
		}

		for _, w := range pf.Waits {
			if !nodeSet[w.ID] {
				continue
			}
			for _, blockerID := range w.BlockedBy {
				if !nodeSet[blockerID] {
					continue
				}
				gd.Edges = append(gd.Edges, vizEdge{
					Source: blockerID,
					Target: w.ID,
					Dashed: true,
				})
			}
		}
	}

	if len(gd.Nodes) == 0 {
		fmt.Println("No items to visualize.")
		return nil
	}

	jsonBytes, err := json.Marshal(gd)
	if err != nil {
		return fmt.Errorf("marshaling graph data: %w", err)
	}

	html := strings.Replace(vizTemplate, "/*GRAPH_DATA*/", string(jsonBytes), 1)

	tmpFile, err := os.CreateTemp("", "tk-viz-*.html")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(html); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	tmpFile.Close()

	opener := "open"
	if runtime.GOOS == "linux" {
		opener = "xdg-open"
	}

	if err := exec.Command(opener, tmpFile.Name()).Start(); err != nil {
		return fmt.Errorf("opening browser: %w (file written to %s)", err, tmpFile.Name())
	}

	fmt.Printf("Opened %s in browser.\n", tmpFile.Name())
	return nil
}
