# tk - Task Tracker Specification

A personal task management system with first-class support for external blockers.

## Inspiration

tk is inspired by [beads](https://github.com/steveyegge/beads), a lightweight issue tracker with first-class dependency support. beads pioneered several ideas that tk builds on:

- Text-based storage for git-friendly diffs
- Dependencies as a core primitive (not an afterthought)
- CLI designed for both humans and AI agents
- Simple data model that stays out of your way
- A git-like workflow feel for issue tracking

**What tk takes from beads:**
- The dependency-first mental model where blockers are explicit relationships
- Text-based storage format for clean diffs and easy inspection
- CLI-first design that works equally well for humans and scripts

**What tk deliberately avoids:**
- beads has a massive surface area with many commands and concepts; tk stays minimal
- beads includes gastown-specific concepts and terminology; tk is domain-agnostic
- beads has agent orchestration layers; tk has no automation beyond auto-complete
- beads uses SQLite caching and daemon processes; tk reads/writes flat files directly

tk diverges from beads in a few key ways:
- **First-class "waits"** for external blockers (packages arriving, time passing) separate from task dependencies
- **Project-centric** rather than repo-centric - designed for personal life management, not just software development
- **Stripped down** - no molecules, gates, agents, or workflow automation; just tasks, waits, and dependencies
- **Radically simple architecture** - no databases, no caching, no daemons; just YAML files

## Design Principles

1. **Tasks live in projects** - Every task belongs to exactly one project
2. **External blockers are explicit** - "Waiting for package" is different from "blocked by other task"
3. **Git-friendly storage** - YAML files, sorted by ID, clean diffs
4. **CLI-first** - Simple commands that work well for humans and AI agents
5. **Keep everything** - Nothing is deleted, status changes are permanent record
6. **Protect against accidents** - Destructive operations require explicit flags
7. **Start simple** - No premature optimization or architectural complexity

## Implementation

- **Language**: Go (single binary distribution)
- **Storage location**: `.tk/` directory in current working directory (no tree walk for v1)
- **Config file**: `.tkconfig.yaml` sibling to `.tk/` directory (user-created, never auto-generated)
- **Timezone**: Local system timezone for all dates/times
- **File writes**: Direct writes, no atomic operations
- **Concurrency**: No locking (personal tool assumption)

### Design Decision: No Tree Walk (v1)

Unlike git, tk does not walk up the directory tree to find `.tk/`. This simplifies the implementation and makes behavior predictable. This decision is worth revisiting in future versions.

## Data Model

### Project

A container for related tasks. Each project has a short prefix used in task IDs.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Lowercase identifier (e.g., `backyard`) |
| `prefix` | string | yes | 2-3 uppercase letters for task IDs (e.g., `BY`) |
| `name` | string | yes | Human-readable name |
| `description` | string | no | What this project is about |
| `status` | enum | yes | `active`, `paused`, `done` |
| `next_id` | int | yes | Counter for generating task/wait IDs |
| `created` | datetime | yes | When project was created |

**Validation rules:**
- `id`: lowercase alphanumeric + dash, 1-50 characters
- `prefix`: 2-3 uppercase letters, must be globally unique across all projects

**Project status effects:**
- `active`: Normal state, tasks appear in queries
- `paused`: Project and all tasks hidden from queries (on ice)
- `done`: Project complete, hidden from default queries

**Constraints:**
- At least one project must exist (cannot delete last project)
- The "Default" project created by `tk init` can be renamed but not deleted
- Cannot add tasks or waits to paused or done projects
- Prefix must be unique - error if attempting to create or change to an existing prefix

Example:
```yaml
id: backyard
prefix: BY
name: Backyard Redo
description: Replace turf with gravel, replant beds, make space pleasant
status: active
next_id: 24
created: 2025-12-02T10:30:00
```

### Task

A unit of work that can be completed.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | `{prefix}-{number}` (e.g., `BY-07`) |
| `title` | string | yes | Short description of the task |
| `status` | enum | yes | `open`, `done`, `dropped` |
| `priority` | int | yes | 1-4 (1=urgent, 2=high, 3=medium, 4=backlog) |
| `blocked_by` | [string] | no | Task IDs or Wait IDs that must complete first |
| `tags` | [string] | no | Lowercase tags for cross-project queries |
| `notes` | string | no | Freeform notes (markdown supported) |
| `assignee` | string | no | Freeform assignee name |
| `due_date` | date | no | Optional soft deadline (past dates allowed) |
| `auto_complete` | bool | no | Auto-complete when all blockers done |
| `created` | datetime | yes | When task was created |
| `updated` | datetime | yes | When task was last modified |
| `done_at` | datetime | no | When status changed to `done` |
| `dropped_at` | datetime | no | When status changed to `dropped` |
| `drop_reason` | string | no | Why the task was dropped (optional) |

**ID format:**
- Dynamic zero-padding: 2 digits until 100, then 3 digits, etc. (BY-01...BY-99, BY-100...BY-999)
- IDs stored internally as integers
- Commands accept any case and any number of leading zeros (by-007, BY-7, BY-07 all match BY-07)
- IDs are normalized to uppercase for storage and display

**Dependencies:**
- `blocked_by` can contain both task IDs (e.g., `BY-05`) and wait IDs (e.g., `BY-03W`)
- All blockers must reference items in the same project
- Circular dependencies are prevented at creation time (full graph cycle detection across tasks and waits)
- Editing blocked_by validates only for cycles; removing incomplete blockers is allowed

Example:
```yaml
id: BY-07
title: Lay landscape fabric
status: open
priority: 2
blocked_by:
  - BY-05
  - BY-03W
tags:
  - hardscape
  - weekend
notes: |
  Use 4ft wide fabric, ~90 linear feet.
  Don't forget landscape staples.
assignee: null
due_date: null
auto_complete: false
created: 2025-12-02T10:30:00
updated: 2025-12-06T14:22:00
done_at: null
dropped_at: null
drop_reason: null
```

### Wait

An external condition that blocks one or more tasks. Unlike task dependencies,
waits represent things outside your control that you're waiting to happen.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | `{prefix}-{number}W` (e.g., `BY-03W`) |
| `title` | string | no | Short label (derived from resolution_criteria if absent) |
| `status` | enum | yes | `open`, `done`, `dropped` (same as tasks) |
| `resolution_criteria` | object | yes | How this wait resolves (see below) |
| `blocked_by` | [string] | no | Task IDs or Wait IDs that must complete before wait becomes active |
| `notes` | string | no | Additional context (markdown supported) |
| `resolution` | string | no | How it resolved (freeform, for done waits) |
| `created` | datetime | yes | When wait was created |
| `done_at` | datetime | no | When status changed to `done` |
| `dropped_at` | datetime | no | When status changed to `dropped` |
| `drop_reason` | string | no | Why the wait was dropped (optional) |

**Resolution Criteria:**

Waits have one of two resolution types:

| Type | Fields | Description |
|------|--------|-------------|
| `time` | `after: datetime` | Auto-resolves when datetime passes |
| `manual` | `question: string`, `check_after: datetime?` | Human answers question to resolve |

- `type: time` waits auto-resolve the instant `after` passes (checked by `tk check`)
- `type: manual` waits require human verification via `tk wait resolve`
- For manual waits, `check_after` controls when the question surfaces in `tk waits --actionable`. If null, the wait is immediately actionable once unblocked.
- Date-only values for `after` or `check_after` are interpreted as end of day (23:59:59)

**Display Logic:**
- If `title` exists, use it for list views
- Else if `type: time`, display "Until {after date}"
- Else if `type: manual`, display the question (truncated if needed)

**Blocked Waits:**
- Waits can be `blocked_by` both tasks and other waits, making them dormant until those blockers complete
- A dormant wait does not surface in `tk waits --actionable`
- This enables planning chains like: task → wait → wait → task

**Constraints:**
- Waits are project-scoped (same numbering as tasks, with W suffix)
- Waits and their blockers must be in the same project
- Cannot resolve dormant waits (must complete blocking items first)

Example (manual wait, immediately actionable):
```yaml
id: BY-03W
title: null
status: open
resolution_criteria:
  type: manual
  question: "Did the landscape fabric arrive from Home Depot?"
  check_after: null
blocked_by: []
notes: "Ordered standard shipping, tracking #12345"
resolution: null
created: 2025-12-06T14:22:00
done_at: null
dropped_at: null
drop_reason: null
```

Example (manual wait, dormant until task completes):
```yaml
id: EL-08W
title: "Prototype PCBs"
status: open
resolution_criteria:
  type: manual
  question: "Did the prototype PCBs arrive?"
  check_after: null
blocked_by:
  - EL-05
notes: null
resolution: null
created: 2025-12-20T09:00:00
done_at: null
dropped_at: null
drop_reason: null
```

Example (time-based wait):
```yaml
id: BY-07W
title: null
status: open
resolution_criteria:
  type: time
  after: 2026-01-15T23:59:59
blocked_by: []
notes: null
resolution: null
created: 2025-12-20T09:00:00
done_at: null
dropped_at: null
drop_reason: null
```

## Derived States

### Task States

A task's **effective state** is computed from its fields:

| State | Condition |
|-------|-----------|
| `ready` | status=open AND all `blocked_by` items are done |
| `blocked` | status=open AND any `blocked_by` task is not done |
| `waiting` | status=open AND any `blocked_by` wait is not done |
| `done` | status=done |
| `dropped` | status=dropped |

A task can be both `blocked` and `waiting` simultaneously.

### Wait States

A wait's **effective state** is computed from its fields:

| State | Condition |
|-------|-----------|
| `dormant` | status=open AND any `blocked_by` item is not done |
| `actionable` | status=open AND all `blocked_by` items are done AND (type=manual AND (check_after is null OR check_after has passed)) |
| `pending` | status=open AND all `blocked_by` items are done AND (type=time OR (type=manual AND check_after has not passed)) |
| `done` | status=done |
| `dropped` | status=dropped |

**Notes:**
- `dormant` waits are blocked by incomplete items and do not surface in queries
- `actionable` waits are manual waits ready for the user to answer
- `pending` waits are either time waits waiting to auto-resolve, or manual waits before their check_after time
- Time waits transition directly from `pending` to `done` when their `after` time passes
- `dropped` waits no longer block any items depending on them

## Storage Format

All data stored as YAML - one file per project, containing all tasks and waits for that project.

```
.tk/
  config.yaml           # global settings (version, etc.)
  projects/
    BY.yaml             # project "backyard" (prefix BY)
    EL.yaml             # project "electronics" (prefix EL)

.tkconfig.yaml          # user configuration (sibling to .tk/, user-created)
```

**File naming:** Project files are named by their prefix (e.g., `BY.yaml` for prefix "BY"). This enables O(1) lookup when resolving task IDs - see `BY-07`, open `BY.yaml`.

### config.yaml

```yaml
version: 1
```

### .tkconfig.yaml

User-created configuration file (never auto-generated by tk):

```yaml
# Auto-run `tk check` on read commands (list, waits, show, ready, waiting, graph, find)
autocheck: false

# Default project for `tk add` when -p not specified
default_project: default

# Default priority for new tasks (1-4)
default_priority: 3
```

### Project File Format

Each project file contains the project metadata followed by tasks and waits as sorted lists.

**Sort order:** Tasks and waits are sorted by numeric ID within each list (BY-01, BY-02, ..., BY-10, BY-11, ...).

```yaml
# .tk/projects/BY.yaml

# Project metadata
id: backyard
prefix: BY
name: Backyard Redo
description: Replace turf with gravel, replant beds
status: active
next_id: 24
created: 2025-12-02T10:30:00

# Tasks as a sorted list
tasks:
  - id: BY-01
    title: Get paper bags
    status: done
    priority: 2
    tags: [shopping]
    created: 2025-12-02T10:30:00
    updated: 2025-12-02T10:30:00
    done_at: 2025-12-02T11:00:00

  - id: BY-02
    title: Fill bags with weeds
    status: open
    priority: 2
    blocked_by: [BY-01]
    notes: |
      Make sure bags are sturdy.
      Don't overfill them.
    created: 2025-12-02T10:35:00
    updated: 2025-12-02T10:35:00

# Waits as a sorted list
waits:
  - id: BY-03W
    status: open
    resolution_criteria:
      type: manual
      question: Did the landscape fabric arrive from Home Depot?
    notes: Ordered standard shipping, tracking #12345
    created: 2025-12-06T14:22:00
```

**Why YAML per project?**
- Clean diffs: editing a task shows only changed fields, not entire lines
- Dependency traversal: all tasks/waits loaded in one read for in-memory graph operations
- Multi-line support: notes and descriptions render naturally
- Self-contained: each project file has everything needed to understand that project

**Why prefix-based filenames?**
- Instant lookup: task ID `BY-07` maps directly to file `BY.yaml`
- Filesystem enforces uniqueness: can't create duplicate prefixes
- Matches mental model: tasks are namespaced by prefix

### Data Integrity

- If YAML files contain syntax errors, commands fail with clear error message pointing to the problematic file
- Use `tk validate` to check data integrity (schema validation, reference checks)
- Use `tk validate --fix` to auto-repair orphan references (references to non-existent tasks/waits)

## CLI Reference

### Global Options

```
--help, -h       Show help
--version        Show version
```

### Command Matching

Commands support unique prefix matching (like git):
- `tk ad` matches `tk add`
- `tk li` matches `tk list`
- `tk pro` matches `tk projects` (but `tk p` is ambiguous)

### Initialization

```bash
# Create .tk/ directory with default project
tk init
tk init --name="My Tasks" --prefix=MT    # custom default project
```

Fails if `.tk/` already exists. Never creates `.tkconfig.yaml` (that's user-managed).

### System Commands

```bash
# Check data integrity (report only by default)
tk validate
tk validate --fix    # auto-repair orphan references

# Run auto-resolution for time-based waits
tk check

# Generate shell completion script (with dynamic ID completion)
tk completion bash
tk completion zsh
tk completion fish
```

### Projects

```bash
# List projects
tk projects
tk projects --all              # include paused/done

# Create project
tk project new backyard --prefix=BY --name="Backyard Redo"

# Show project summary (counts only)
tk project backyard            # e.g., "5 open (2 ready, 3 blocked), 10 done, 2 waits"

# Edit project (flag-based for agents, -i for interactive)
tk project edit backyard --name="New Name"
tk project edit backyard --status=paused
tk project edit backyard --prefix=NW     # triggers ID migration (fails if prefix in use)
tk project edit backyard -i              # open in $EDITOR (YAML format, abort on error)

# Delete project (cascades - removes all tasks/waits)
tk project delete backyard --force

# Export project as plain text (human-readable, not for import)
tk dump backyard
```

### Tasks

```bash
# Add task (uses default_project from .tkconfig if -p not specified)
tk add "Dig test hole"
tk add "Dig test hole" --project=backyard
tk add "Dig test hole" -p backyard --priority=1 --tag=weekend
tk add "Dig test hole" -p backyard --blocked-by=BY-05,BY-03W

# Output: BY-08 Dig test hole

# List tasks (sorted by ID)
tk list                        # all open tasks in active projects
tk list -p backyard            # one project
tk list --ready                # unblocked and not waiting
tk list --blocked              # has unfinished task dependencies
tk list --waiting              # has open waits
tk list --done                 # completed tasks
tk list --dropped              # dropped tasks
tk list --all                  # everything including done/dropped
tk list --priority=1           # filter by priority
tk list --p1                   # shorthand for --priority=1
tk list --tag=errand           # filter by tag
tk list --tag=call --tag=home  # AND: must have all tags
tk list --overdue              # tasks with due_date < now (time-aware)

# Output columns: ID, status indicator, priority, title, tags

# Show task details
tk show BY-07
# Shows: ID, title, status, priority, tags, notes, assignee, due_date
# For blocked_by: shows ID + [status] + title (e.g., "BY-05 [done] Dig test hole")

# Edit task (flag-based for agents, -i for interactive)
tk edit BY-07 --title="New title"
tk edit BY-07 --priority=2
tk edit BY-07 --notes="Additional context"
tk edit BY-07 --assignee="John"
tk edit BY-07 --due-date=2026-02-15
tk edit BY-07 --auto-complete=true
tk edit BY-07 --tags=weekend,hardscape      # replaces all tags
tk edit BY-07 --add-tag=urgent              # adds tag
tk edit BY-07 --remove-tag=weekend          # removes tag
tk edit BY-07 --blocked-by=BY-05,BY-06      # replaces blockers (tasks or waits)
tk edit BY-07 --add-blocked-by=BY-08        # adds blocker
tk edit BY-07 --remove-blocked-by=BY-05     # removes blocker
tk edit BY-07 -i                            # open in $EDITOR (YAML format, abort on error)

# Tag shortcuts
tk tag BY-07 weekend           # add tag
tk untag BY-07 weekend         # remove tag

# Complete task
tk done BY-07
tk done BY-07 BY-08 BY-09      # batch: multiple IDs (best effort - completes what it can)
tk done BY-07 --force          # removes unfulfilled blockers from task, then completes

# Output on success: shows unblocked tasks, activated waits, auto-completed tasks

# Drop task
tk drop BY-07
tk drop BY-07 --reason="No longer needed"
tk drop BY-07 --drop-deps      # also drop all dependent tasks and blocked waits
tk drop BY-07 --remove-deps    # remove this task from dependents' blocked_by

# Reopen task (works on done or dropped)
tk reopen BY-07
# Clears drop_reason if present; does not affect downstream tasks

# Defer task (creates time wait and links it)
# Error if task already has open waits
tk defer BY-07 --days=4                # defer for 4 days (end of that day)
tk defer BY-07 --until=2026-01-20      # defer until specific date (end of day)

# Move task to different project
# Fails if ANY task or wait references BY-07 (in either direction)
tk move BY-07 --to=household

# Dependencies between tasks (--by works for both tasks and waits)
tk block BY-07 --by=BY-05          # BY-07 blocked by BY-05
tk block BY-07 --by=BY-03W         # BY-07 blocked by wait BY-03W
tk unblock BY-07 --from=BY-05      # remove dependency

# Query dependencies (shows both tasks and waits)
tk blocked-by BY-07                # what is blocking BY-07?
tk blocking BY-05                  # what is BY-05 blocking?
```

### Waits

```bash
# Create manual wait (--question implies type=manual)
tk wait add -p backyard --question="Did the landscape fabric arrive?"
tk wait add "Fabric delivery" -p backyard --question="Did the landscape fabric arrive?"
tk wait add -p backyard --question="Did the PCBs arrive?" --check-after=2026-01-10
tk wait add -p backyard --question="Did the PCBs arrive?" --blocked-by=EL-05

# Output: BY-03W Did the landscape fabric arrive?

# Create time wait (--after implies type=time)
tk wait add -p backyard --after=2026-01-15                # end of day
tk wait add "After Jan 15" -p backyard --after=2026-01-15T14:00:00

# List waits (sorted by ID)
tk waits                       # all open waits in active projects (auto-runs tk check)
tk waits -p backyard           # one project
tk waits --actionable          # manual waits ready for user to answer (auto-runs tk check)
tk waits --dormant             # waits blocked by incomplete items
tk waits --done                # completed waits
tk waits --dropped             # dropped waits
tk waits --all                 # all waits including done/dropped

# Output columns: ID, display text (title or question or "Until {date}")

# Edit wait
tk wait edit BY-03W --title="New title"
tk wait edit BY-03W --question="Updated question?"
tk wait edit BY-03W --check-after=2026-01-20T12:00:00
tk wait edit BY-03W --after=2026-01-20T12:00:00      # for time waits
tk wait edit BY-03W --notes="Tracking number: 123456"
tk wait edit BY-03W --blocked-by=BY-05,BY-06         # replaces blockers (tasks or waits)
tk wait edit BY-03W --add-blocked-by=BY-07           # adds blocker
tk wait edit BY-03W --remove-blocked-by=BY-05        # removes blocker
tk wait edit BY-03W -i                               # open in $EDITOR (YAML format)

# Resolve manual wait (immediate, no confirmation prompt)
tk wait resolve BY-03W
tk wait resolve BY-03W --as="Arrived damaged, returning"

# Resolving a time wait early: allowed, updates 'after' to current time, prints explanation
tk wait resolve BY-07W

# Error: cannot resolve dormant waits

# Defer wait (push back dates)
tk wait defer BY-03W --days=4                # manual: pushes check_after; time: pushes after
tk wait defer BY-03W --until=2026-01-20

# Drop wait
tk wait drop BY-03W
tk wait drop BY-03W --reason="No longer needed"
tk wait drop BY-03W --drop-deps              # also drop dependent tasks
tk wait drop BY-03W --remove-deps            # unlink from dependent tasks
```

### Queries

```bash
# Find tasks and waits (case-insensitive substring match in title/question and notes)
tk find "gravel"               # search all active projects
tk find "gravel" -p backyard   # within project

# Output clearly labels tasks vs waits in results

# What's actionable right now?
tk ready                       # alias for: tk list --ready

# What waits need attention?
tk waiting                     # alias for: tk waits --actionable
```

### Visualization

```bash
# Generate DOT graph of dependencies
tk graph                       # all projects
tk graph -p backyard           # one project

# Full styling:
# - Tasks as boxes, waits as diamonds
# - Colors for states: ready (green), blocked (red), waiting (yellow), done (gray), dropped (strikethrough)
# - Dashed lines for wait dependencies

# Output suitable for graphviz: tk graph | dot -Tpng -o deps.png
```

## Completion Behavior

### Auto-Complete

When any task is completed (for any reason), check all tasks that depend on it. If a dependent task has `auto_complete: true` AND all its blockers are now done, that task auto-completes. This cascades transitively through the dependency graph.

### Completing a Task (`tk done BY-07`)

- If task has incomplete blockers: error, suggest `--force`
- With `--force`: removes unfulfilled blockers from the task's `blocked_by` list, then completes the task
- On success: shows what tasks were unblocked, what waits became active, and what tasks were auto-completed
  - Example: "BY-07 done. Unblocked: BY-08, BY-09. Waits now active: BY-03W. Auto-completed: BY-10"
- Waits that were blocked by this task become active (dormant → actionable/pending)

### Batch Completion (`tk done BY-07 BY-08 BY-09`)

- Best effort: completes all tasks that can be completed
- Reports errors for tasks that couldn't be completed (e.g., had blockers)
- Continues processing remaining tasks after errors

### Dropping a Task (`tk drop BY-07`)

- If task has dependent tasks OR blocked waits: error, suggest `--drop-deps` or `--remove-deps`
- With `--drop-deps`: drops all dependent tasks AND blocked waits recursively
- With `--remove-deps`: removes BY-07 from all downstream blocked_by lists
  - Dependent tasks become unblocked
  - Blocked waits become active (dormant → actionable/pending)
- `--reason` is optional

### Dropping a Wait (`tk wait drop BY-03W`)

- If wait has items depending on it: error, suggest `--drop-deps` or `--remove-deps`
- With `--drop-deps`: drops all tasks and waits depending on this wait recursively
- With `--remove-deps`: removes BY-03W from all downstream blocked_by lists
  - Those items may become ready if they have no other blockers
- `--reason` is optional

### Reopening (`tk reopen BY-07`)

- Works on both done and dropped tasks/waits
- Sets status back to open
- Clears drop_reason, dropped_at, done_at
- Does NOT affect downstream tasks (they remain in whatever state they're in)

## Error Handling

- Errors go to stderr with non-zero exit code
- Error messages are human-readable and actionable

**Error message styles:**
- Not found: `error: task BY-999 not found`
- Cycle detected: `error: BY-07 cannot depend on BY-05 (would create cycle: BY-05 → BY-03 → BY-07)`
- Project not active: `error: cannot add task to paused project 'backyard'`
- $EDITOR not set: `error: EDITOR not set. Set it or use --flags instead of -i`

## Output Formatting

- Colored output auto-detected (color if TTY, plain if piped)
- `tk list` shows flat list with status indicators [blocked], [waiting], etc.
- Dates displayed in ISO 8601 format (2026-01-15T14:30:00)
- Running `tk` with no subcommand shows help

## Future Considerations

Not in v1, but worth thinking about:

- **REST API** - HTTP server mirroring CLI commands
- **Export/Import** - Backup and restore functionality
- **JSON output** - `--json` flag for machine-readable output
- **Recurring waits** - "Check on X every week"
- **Subtasks** - Explicit hierarchical task breakdown (currently modeled via blocking)
- **Time tracking** - Log time spent on tasks
- **Attachments** - Links to files, images
- **Sync protocol** - Conflict resolution for multi-device use
- **Tree walk** - Walk up directory tree to find `.tk/` like git
- **Archive command** - Move completed projects out of main storage
- **Advanced search** - Fuzzy matching, full-text search with ranking
- **Accessibility** - Screen reader support, high contrast mode
- **Bulk import** - Create multiple tasks from text file
- **Task logs** - Chronological activity log per task
- **Due soon filter** - `--due-before=DATE` for filtering by due date
- **Summary dashboard** - `tk status` showing counts across all projects
