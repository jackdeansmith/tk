package model

import (
	"testing"
	"time"
)

func TestComputeTaskState(t *testing.T) {
	tests := []struct {
		name          string
		task          *Task
		blockerStates BlockerStatus
		want          TaskState
	}{
		{
			name: "done task returns done regardless of blockers",
			task: &Task{
				Status:    TaskStatusDone,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": false},
			want:          TaskStateDone,
		},
		{
			name: "dropped task returns dropped regardless of blockers",
			task: &Task{
				Status:    TaskStatusDropped,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": false},
			want:          TaskStateDropped,
		},
		{
			name: "open task with no blockers is ready",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: nil,
			},
			blockerStates: BlockerStatus{},
			want:          TaskStateReady,
		},
		{
			name: "open task with empty blockers is ready",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{},
			},
			blockerStates: BlockerStatus{},
			want:          TaskStateReady,
		},
		{
			name: "open task with done blocker is ready",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": true},
			want:          TaskStateReady,
		},
		{
			name: "open task with open task blocker is blocked",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": false},
			want:          TaskStateBlocked,
		},
		{
			name: "open task with open wait blocker is waiting",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01W"},
			},
			blockerStates: BlockerStatus{"BY-01W": false},
			want:          TaskStateWaiting,
		},
		{
			name: "open task with both open task and wait blockers returns blocked",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01", "BY-02W"},
			},
			blockerStates: BlockerStatus{
				"BY-01":  false,
				"BY-02W": false,
			},
			want: TaskStateBlocked,
		},
		{
			name: "open task with dropped blocker is ready",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": true}, // dropped = resolved = true
			want:          TaskStateReady,
		},
		{
			name: "open task with multiple blockers all done is ready",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01", "BY-02", "BY-03W"},
			},
			blockerStates: BlockerStatus{
				"BY-01":  true,
				"BY-02":  true,
				"BY-03W": true,
			},
			want: TaskStateReady,
		},
		{
			name: "open task with one open blocker among many is blocked",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01", "BY-02", "BY-03"},
			},
			blockerStates: BlockerStatus{
				"BY-01": true,
				"BY-02": false,
				"BY-03": true,
			},
			want: TaskStateBlocked,
		},
		{
			name: "open task with one open wait among many is waiting",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01", "BY-02W", "BY-03W"},
			},
			blockerStates: BlockerStatus{
				"BY-01":  true,
				"BY-02W": true,
				"BY-03W": false,
			},
			want: TaskStateWaiting,
		},
		{
			name: "open task with unknown blocker treats it as blocking",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{}, // BY-01 not in map
			want:          TaskStateBlocked,
		},
		{
			name: "open task with unknown wait blocker treats it as waiting",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01W"},
			},
			blockerStates: BlockerStatus{}, // BY-01W not in map
			want:          TaskStateWaiting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeTaskState(tt.task, tt.blockerStates)
			if got != tt.want {
				t.Errorf("ComputeTaskState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestComputeWaitState(t *testing.T) {
	now := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	pastTime := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	futureTime := time.Date(2026, 1, 20, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name          string
		wait          *Wait
		blockerStates BlockerStatus
		now           time.Time
		want          WaitState
	}{
		{
			name: "done wait returns done",
			wait: &Wait{
				Status: WaitStatusDone,
			},
			blockerStates: BlockerStatus{},
			now:           now,
			want:          WaitStateDone,
		},
		{
			name: "dropped wait returns dropped",
			wait: &Wait{
				Status: WaitStatusDropped,
			},
			blockerStates: BlockerStatus{},
			now:           now,
			want:          WaitStateDropped,
		},
		{
			name: "open wait with open blocker is dormant",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: []string{"BY-01"},
				ResolutionCriteria: ResolutionCriteria{
					Type:     ResolutionTypeManual,
					Question: "Test?",
				},
			},
			blockerStates: BlockerStatus{"BY-01": false},
			now:           now,
			want:          WaitStateDormant,
		},
		{
			name: "open manual wait with no blockers and no check_after is actionable",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: nil,
				ResolutionCriteria: ResolutionCriteria{
					Type:       ResolutionTypeManual,
					Question:   "Did the package arrive?",
					CheckAfter: nil,
				},
			},
			blockerStates: BlockerStatus{},
			now:           now,
			want:          WaitStateActionable,
		},
		{
			name: "open manual wait with check_after in future is pending",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: nil,
				ResolutionCriteria: ResolutionCriteria{
					Type:       ResolutionTypeManual,
					Question:   "Did the package arrive?",
					CheckAfter: &futureTime,
				},
			},
			blockerStates: BlockerStatus{},
			now:           now,
			want:          WaitStatePending,
		},
		{
			name: "open manual wait with check_after in past is actionable",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: nil,
				ResolutionCriteria: ResolutionCriteria{
					Type:       ResolutionTypeManual,
					Question:   "Did the package arrive?",
					CheckAfter: &pastTime,
				},
			},
			blockerStates: BlockerStatus{},
			now:           now,
			want:          WaitStateActionable,
		},
		{
			name: "open manual wait with check_after equal to now is actionable",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: nil,
				ResolutionCriteria: ResolutionCriteria{
					Type:       ResolutionTypeManual,
					Question:   "Did the package arrive?",
					CheckAfter: &now,
				},
			},
			blockerStates: BlockerStatus{},
			now:           now,
			want:          WaitStateActionable,
		},
		{
			name: "open time wait with after in future is pending",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: nil,
				ResolutionCriteria: ResolutionCriteria{
					Type:  ResolutionTypeTime,
					After: &futureTime,
				},
			},
			blockerStates: BlockerStatus{},
			now:           now,
			want:          WaitStatePending,
		},
		{
			name: "open time wait with after in past is still pending (tk check transitions)",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: nil,
				ResolutionCriteria: ResolutionCriteria{
					Type:  ResolutionTypeTime,
					After: &pastTime,
				},
			},
			blockerStates: BlockerStatus{},
			now:           now,
			want:          WaitStatePending,
		},
		{
			name: "open wait blocked by done blocker checks resolution criteria",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: []string{"BY-01"},
				ResolutionCriteria: ResolutionCriteria{
					Type:       ResolutionTypeManual,
					Question:   "Test?",
					CheckAfter: nil,
				},
			},
			blockerStates: BlockerStatus{"BY-01": true},
			now:           now,
			want:          WaitStateActionable,
		},
		{
			name: "open wait blocked by multiple blockers with one open is dormant",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: []string{"BY-01", "BY-02"},
				ResolutionCriteria: ResolutionCriteria{
					Type:     ResolutionTypeManual,
					Question: "Test?",
				},
			},
			blockerStates: BlockerStatus{
				"BY-01": true,
				"BY-02": false,
			},
			now:  now,
			want: WaitStateDormant,
		},
		{
			name: "open wait blocked by wait is dormant",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: []string{"BY-01W"},
				ResolutionCriteria: ResolutionCriteria{
					Type:     ResolutionTypeManual,
					Question: "Test?",
				},
			},
			blockerStates: BlockerStatus{"BY-01W": false},
			now:           now,
			want:          WaitStateDormant,
		},
		{
			name: "open wait with unknown blocker treats it as dormant",
			wait: &Wait{
				Status:    WaitStatusOpen,
				BlockedBy: []string{"BY-01"},
				ResolutionCriteria: ResolutionCriteria{
					Type:     ResolutionTypeManual,
					Question: "Test?",
				},
			},
			blockerStates: BlockerStatus{}, // BY-01 not in map
			now:           now,
			want:          WaitStateDormant,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeWaitState(tt.wait, tt.blockerStates, tt.now)
			if got != tt.want {
				t.Errorf("ComputeWaitState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTaskBlocked(t *testing.T) {
	tests := []struct {
		name          string
		task          *Task
		blockerStates BlockerStatus
		want          bool
	}{
		{
			name: "open task with open task blocker is blocked",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": false},
			want:          true,
		},
		{
			name: "open task with only wait blockers is not blocked",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01W"},
			},
			blockerStates: BlockerStatus{"BY-01W": false},
			want:          false,
		},
		{
			name: "done task is not blocked",
			task: &Task{
				Status:    TaskStatusDone,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": false},
			want:          false,
		},
		{
			name: "open task with resolved task blocker is not blocked",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": true},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTaskBlocked(tt.task, tt.blockerStates)
			if got != tt.want {
				t.Errorf("IsTaskBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTaskWaiting(t *testing.T) {
	tests := []struct {
		name          string
		task          *Task
		blockerStates BlockerStatus
		want          bool
	}{
		{
			name: "open task with open wait blocker is waiting",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01W"},
			},
			blockerStates: BlockerStatus{"BY-01W": false},
			want:          true,
		},
		{
			name: "open task with only task blockers is not waiting",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01"},
			},
			blockerStates: BlockerStatus{"BY-01": false},
			want:          false,
		},
		{
			name: "dropped task is not waiting",
			task: &Task{
				Status:    TaskStatusDropped,
				BlockedBy: []string{"BY-01W"},
			},
			blockerStates: BlockerStatus{"BY-01W": false},
			want:          false,
		},
		{
			name: "open task with resolved wait blocker is not waiting",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01W"},
			},
			blockerStates: BlockerStatus{"BY-01W": true},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTaskWaiting(tt.task, tt.blockerStates)
			if got != tt.want {
				t.Errorf("IsTaskWaiting() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTaskReady(t *testing.T) {
	tests := []struct {
		name          string
		task          *Task
		blockerStates BlockerStatus
		want          bool
	}{
		{
			name: "open task with no blockers is ready",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: nil,
			},
			blockerStates: BlockerStatus{},
			want:          true,
		},
		{
			name: "open task with all resolved blockers is ready",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01", "BY-02W"},
			},
			blockerStates: BlockerStatus{
				"BY-01":  true,
				"BY-02W": true,
			},
			want: true,
		},
		{
			name: "open task with any open blocker is not ready",
			task: &Task{
				Status:    TaskStatusOpen,
				BlockedBy: []string{"BY-01", "BY-02"},
			},
			blockerStates: BlockerStatus{
				"BY-01": true,
				"BY-02": false,
			},
			want: false,
		},
		{
			name: "done task is not ready",
			task: &Task{
				Status:    TaskStatusDone,
				BlockedBy: nil,
			},
			blockerStates: BlockerStatus{},
			want:          false,
		},
		{
			name: "dropped task is not ready",
			task: &Task{
				Status:    TaskStatusDropped,
				BlockedBy: nil,
			},
			blockerStates: BlockerStatus{},
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsTaskReady(tt.task, tt.blockerStates)
			if got != tt.want {
				t.Errorf("IsTaskReady() = %v, want %v", got, tt.want)
			}
		})
	}
}
