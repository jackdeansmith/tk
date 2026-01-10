package model

import "time"

// TaskState represents the derived state of a task.
type TaskState string

const (
	TaskStateReady   TaskState = "ready"
	TaskStateBlocked TaskState = "blocked"
	TaskStateWaiting TaskState = "waiting"
	TaskStateDone    TaskState = "done"
	TaskStateDropped TaskState = "dropped"
)

// WaitState represents the derived state of a wait.
type WaitState string

const (
	WaitStateDormant    WaitState = "dormant"
	WaitStateActionable WaitState = "actionable"
	WaitStatePending    WaitState = "pending"
	WaitStateDone       WaitState = "done"
	WaitStateDropped    WaitState = "dropped"
)

// BlockerStatus represents whether a blocker is resolved (done/dropped) or not.
// True means the blocker is resolved and doesn't block.
// False means the blocker is still open and blocks.
type BlockerStatus map[string]bool

// ComputeTaskState returns the derived state for a task.
// blockerStates maps blocker IDs to their resolved status:
// - true: blocker is done or dropped (doesn't block)
// - false: blocker is open (blocks)
//
// When a task has both task blockers and wait blockers that are open,
// TaskStateBlocked is returned (blocked takes precedence over waiting).
func ComputeTaskState(t *Task, blockerStates BlockerStatus) TaskState {
	// Terminal states take precedence
	if t.Status == TaskStatusDone {
		return TaskStateDone
	}
	if t.Status == TaskStatusDropped {
		return TaskStateDropped
	}

	// Task must be open - check blockers
	if len(t.BlockedBy) == 0 {
		return TaskStateReady
	}

	hasOpenTaskBlocker := false
	hasOpenWaitBlocker := false

	for _, blockerID := range t.BlockedBy {
		resolved, exists := blockerStates[blockerID]
		if !exists {
			// If we don't have status info, assume it's still blocking
			resolved = false
		}

		if !resolved {
			// This blocker is still open
			if IsWaitID(blockerID) {
				hasOpenWaitBlocker = true
			} else {
				hasOpenTaskBlocker = true
			}
		}
	}

	// Blocked takes precedence over waiting
	if hasOpenTaskBlocker {
		return TaskStateBlocked
	}
	if hasOpenWaitBlocker {
		return TaskStateWaiting
	}

	// All blockers are resolved
	return TaskStateReady
}

// ComputeWaitState returns the derived state for a wait.
// blockerStates maps blocker IDs to their resolved status:
// - true: blocker is done or dropped (doesn't block)
// - false: blocker is open (blocks)
// now is the current time for evaluating time-based conditions.
//
// State logic:
// - dormant: has unresolved blockers
// - actionable: manual wait with no blockers AND (check_after is nil OR has passed)
// - pending: time wait waiting to auto-resolve, OR manual wait before check_after
// - done/dropped: terminal states
func ComputeWaitState(w *Wait, blockerStates BlockerStatus, now time.Time) WaitState {
	// Terminal states take precedence
	if w.Status == WaitStatusDone {
		return WaitStateDone
	}
	if w.Status == WaitStatusDropped {
		return WaitStateDropped
	}

	// Wait must be open - check blockers
	for _, blockerID := range w.BlockedBy {
		resolved, exists := blockerStates[blockerID]
		if !exists {
			resolved = false
		}
		if !resolved {
			return WaitStateDormant
		}
	}

	// All blockers resolved (or no blockers) - determine state based on type
	switch w.ResolutionCriteria.Type {
	case ResolutionTypeTime:
		// Time waits are always pending until tk check resolves them
		// (even if after time has passed - that's tk check's job)
		return WaitStatePending

	case ResolutionTypeManual:
		// Manual wait: actionable if check_after is nil or has passed
		if w.ResolutionCriteria.CheckAfter == nil {
			return WaitStateActionable
		}
		if now.After(*w.ResolutionCriteria.CheckAfter) || now.Equal(*w.ResolutionCriteria.CheckAfter) {
			return WaitStateActionable
		}
		return WaitStatePending

	default:
		// Unknown type - treat as pending
		return WaitStatePending
	}
}

// IsTaskBlocked returns true if the task has any open task blockers.
// Useful for filtering with --blocked flag.
func IsTaskBlocked(t *Task, blockerStates BlockerStatus) bool {
	if t.Status != TaskStatusOpen {
		return false
	}
	for _, blockerID := range t.BlockedBy {
		if IsTaskID(blockerID) {
			resolved, exists := blockerStates[blockerID]
			if !exists || !resolved {
				return true
			}
		}
	}
	return false
}

// IsTaskWaiting returns true if the task has any open wait blockers.
// Useful for filtering with --waiting flag.
func IsTaskWaiting(t *Task, blockerStates BlockerStatus) bool {
	if t.Status != TaskStatusOpen {
		return false
	}
	for _, blockerID := range t.BlockedBy {
		if IsWaitID(blockerID) {
			resolved, exists := blockerStates[blockerID]
			if !exists || !resolved {
				return true
			}
		}
	}
	return false
}

// IsTaskReady returns true if the task is open and all blockers are resolved.
func IsTaskReady(t *Task, blockerStates BlockerStatus) bool {
	if t.Status != TaskStatusOpen {
		return false
	}
	for _, blockerID := range t.BlockedBy {
		resolved, exists := blockerStates[blockerID]
		if !exists || !resolved {
			return false
		}
	}
	return true
}
