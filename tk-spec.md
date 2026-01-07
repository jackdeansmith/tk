# tk - Task Tracker Specification

A personal task management system with first-class support for external blockers.

## Inspiration

tk is inspired by [beads](https://github.com/steveyegge/beads), a lightweight issue tracker with first-class dependency support. beads pioneered several ideas that tk builds on:

- JSONL storage for git-friendly diffs
- Dependencies as core primitive (not an afterthought)
- CLI designed for both humans and AI agents
- Simple data model that stays out of your way

tk diverges from beads in a few key ways:
- **First-class "waits"** for external blockers (packages arriving, time passing) separate from task dependencies
- **Project-centric** rather than repo-centric - designed for personal life management, not just software development
- **Stripped down** - no molecules, gates, agents, or workflow automation; just tasks, waits, and dependencies

## Design Principles

1. **Tasks live in projects** - Every task belongs to exactly one project
2. **External blockers are explicit** - "Waiting for package" is different from "blocked by other task"
3. **Git-friendly storage** - JSONL files, sorted by ID, clean diffs
4. **CLI-first** - Simple commands that work well for humans and AI agents
5. **Keep everything** - Nothing is deleted, status changes are permanent record
6. **Protect against accidents** - Destructive operations require explicit flags

## Implementation

- **Language**: Go (single binary distribution)
- **Storage location**: `.tk/` directory in current working directory (no tree walk for v1)
- **Config file**: `.tkconfig.yaml` sibling to `.tk/` directory
- **Timezone**: Local system timezone for all dates/times

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
- `prefix`: 2-3 uppercase letters

**Project status effects:**
- `active`: Normal state, tasks appear in queries
- `paused`: Project and all tasks hidden from queries (on ice)
- `done`: Project complete, hidden from default queries

**Notes:**
- At least one project must exist (cannot delete last project)
- Project prefix can be changed; this triggers migration of all task/wait IDs
- The "Default" project created by `tk init` can be renamed but not deleted

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
| `project` | string | yes | Project ID this task belongs to |
| `status` | enum | yes | `open`, `done`, `dropped` |
| `priority` | int | yes | 1-4 (1=urgent, 2=high, 3=medium, 4=backlog) |
| `blocked_by` | [string] | no | Task IDs that must complete first |
| `waiting_on` | [string] | no | Wait IDs for external blockers |
| `tags` | [string] | no | Lowercase tags for cross-project queries |
| `notes` | string | no | Freeform notes (markdown supported) |
| `assignee` | string | no | Freeform assignee name |
| `due_date` | date | no | Optional soft deadline |
| `auto_complete` | bool | no | Auto-complete when all blockers done |
| `log` | [LogEntry] | no | Chronological activity log |
| `created` | datetime | yes | When task was created |
| `updated` | datetime | yes | When task was last modified |
| `done_at` | datetime | no | When status changed to `done` |
| `dropped_at` | datetime | no | When status changed to `dropped` |
| `drop_reason` | string | no | Why the task was dropped |

LogEntry:
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `date` | datetime | yes | When the log entry was created |
| `text` | string | yes | What happened (unlimited length) |

**ID format:**
- 2-digit zero-padding by default (BY-01, BY-02, ..., BY-99, BY-100)
- IDs stored internally as integers
- Commands accept any number of leading zeros (BY-007 matches BY-07)

**Dependencies:**
- `blocked_by` and `waiting_on` must reference items in the same project
- Circular dependencies are prevented at creation time
- Status changes automatically create log entries

Example:
```yaml
id: BY-07
title: Lay landscape fabric
project: backyard
status: open
priority: 2
blocked_by:
  - BY-05
waiting_on:
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
log:
  - date: 2025-12-02T10:30:00
    text: Created
  - date: 2025-12-06T14:22:00
    text: Ordered fabric from Home Depot, arriving next week
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
| `project` | string | yes | Project this wait belongs to |
| `status` | enum | yes | `open`, `done`, `dropped` (same as tasks) |
| `resolution_criteria` | object | yes | How this wait resolves (see below) |
| `blocked_by` | [string] | no | Task IDs that must complete before wait becomes active |
| `notes` | string | no | Additional context (markdown supported) |
| `resolution` | string | no | How it resolved (freeform, for done waits) |
| `created` | datetime | yes | When wait was created |
| `done_at` | datetime | no | When status changed to `done` |
| `dropped_at` | datetime | no | When status changed to `dropped` |
| `drop_reason` | string | no | Why the wait was dropped |

**Resolution Criteria:**

Waits have one of two resolution types:

| Type | Fields | Description |
|------|--------|-------------|
| `time` | `after: datetime` | Auto-resolves when datetime passes |
| `manual` | `question: string`, `check_after: datetime?` | Human answers question to resolve |

- `type: time` waits auto-resolve the instant `after` passes (checked by `tk check`)
- `type: manual` waits require human verification via `tk wait resolve`
- For manual waits, `check_after` controls when the question surfaces in `tk waits --actionable`. If null, the wait is immediately actionable once unblocked.

**Display Logic:**
- If `title` exists, use it for list views
- Else if `type: time`, display "Until {after date}"
- Else if `type: manual`, display the question (truncated if needed)

**Blocked Waits:**
- Waits can be `blocked_by` tasks, making them dormant until those tasks complete
- A dormant wait does not surface in `tk waits --actionable`
- This enables planning chains like: task → task → wait → task

**Notes:**
- Waits are project-scoped (same numbering as tasks, with W suffix)
- Waits and their blockers must be in the same project

Example (manual wait, immediately actionable):
```yaml
id: BY-03W
title: null
project: backyard
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
id: BY-08W
title: "Prototype PCBs"
project: electronics
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
project: backyard
status: open
resolution_criteria:
  type: time
  after: 2026-01-15T00:00:00
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
| `ready` | status=open AND all `blocked_by` tasks are done AND all `waiting_on` waits are done |
| `blocked` | status=open AND any `blocked_by` task is not done |
| `waiting` | status=open AND any `waiting_on` wait is not done |
| `done` | status=done |
| `dropped` | status=dropped |

A task can be both `blocked` and `waiting` simultaneously.

### Wait States

A wait's **effective state** is computed from its fields:

| State | Condition |
|-------|-----------|
| `dormant` | status=open AND any `blocked_by` task is not done |
| `actionable` | status=open AND all `blocked_by` tasks are done AND (type=manual AND (check_after is null OR check_after has passed)) |
| `pending` | status=open AND all `blocked_by` tasks are done AND (type=time OR (type=manual AND check_after has not passed)) |
| `done` | status=done |
| `dropped` | status=dropped |

**Notes:**
- `dormant` waits are blocked by incomplete tasks and do not surface in queries
- `actionable` waits are manual waits ready for the user to answer
- `pending` waits are either time waits waiting to auto-resolve, or manual waits before their check_after time
- Time waits transition directly from `pending` to `done` when their `after` time passes
- `dropped` waits no longer block any tasks waiting on them

## Storage Format

All data stored as JSONL (JSON Lines) - one JSON object per line, sorted by ID.

```
.tk/
  projects.jsonl    # project definitions
  tasks.jsonl       # all tasks
  waits.jsonl       # all waits
  config.json       # global settings (version, etc.)

.tkconfig.yaml      # user configuration (sibling to .tk/)
```

### config.json

```json
{
  "version": 1
}
```

### .tkconfig.yaml

```yaml
# Auto-run `tk check` on every command
autocheck: false

# Default project for `tk add` when -p not specified
default_project: default

# Default priority for new tasks (1-4)
default_priority: 3
```

### JSONL Format

Each file contains one JSON object per line, sorted by ID for stable diffs:

```jsonl
{"id":"BY-01","title":"Get paper bags","project":"backyard","status":"done",...}
{"id":"BY-02","title":"Fill bags with weeds","project":"backyard","status":"open",...}
{"id":"BY-03","title":"Dispose yard waste","project":"backyard","status":"open",...}
```

### Data Integrity

- If JSONL files contain invalid JSON, commands fail with clear error message pointing to the problematic file/line
- Use `tk validate` to check data integrity

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

Fails if `.tk/` already exists.

### System Commands

```bash
# Check data integrity
tk validate

# Run auto-resolution for time-based waits
tk check

# Generate shell completion script
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

# Show project summary
tk project backyard            # stats, recent activity

# Edit project (flag-based for agents, -i for interactive)
tk project edit backyard --name="New Name"
tk project edit backyard --status=paused
tk project edit backyard --prefix=NW     # triggers ID migration
tk project edit backyard -i              # open in $EDITOR (YAML format)

# Delete project (cascades - removes all tasks/waits)
tk project delete backyard --force

# Export project as plain text (full state)
tk dump backyard
```

### Tasks

```bash
# Add task (uses default_project from .tkconfig if -p not specified)
tk add "Dig test hole"
tk add "Dig test hole" --project=backyard
tk add "Dig test hole" -p backyard --priority=1 --tag=weekend

# List tasks
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
tk list --overdue              # tasks with passed due_date
tk list --limit=10             # limit results
tk list --offset=20            # skip first N results

# Show task details (summary by default, --verbose for full log)
tk show BY-07
tk show BY-07 --verbose

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
tk edit BY-07 --blocked-by=BY-05,BY-06      # replaces blockers
tk edit BY-07 --add-blocked-by=BY-08        # adds blocker
tk edit BY-07 --remove-blocked-by=BY-05     # removes blocker
tk edit BY-07 --waiting-on=BY-03W           # replaces waits
tk edit BY-07 -i                            # open in $EDITOR (YAML format)

# Tag shortcuts
tk tag BY-07 weekend           # add tag
tk untag BY-07 weekend         # remove tag

# Complete task
tk done BY-07
tk done BY-07 BY-08 BY-09      # batch: multiple IDs
tk done BY-07 --force          # force even if blockers remain (cascades)

# Drop task
tk drop BY-07 --reason="No longer needed"
tk drop BY-07 --drop-deps      # also drop all dependent tasks
tk drop BY-07 --remove-deps    # remove this task from dependents' blocked_by

# Reopen task (works on done or dropped)
tk reopen BY-07

# Defer task (creates time wait and links it)
tk defer BY-07 --days=4                # defer for 4 days
tk defer BY-07 --until=2026-01-20      # defer until specific date

# Add log entry
tk log BY-07 "Tested soil, mostly clean fill"

# Move task to different project (fails if task has deps or waits)
tk move BY-07 --to=household

# Dependencies between tasks
tk block BY-07 --by=BY-05          # BY-07 blocked by BY-05
tk unblock BY-07 --from=BY-05      # remove dependency

# Query dependencies
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

# Create time wait (--after implies type=time)
tk wait add -p backyard --after=2026-01-15
tk wait add "After Jan 15" -p backyard --after=2026-01-15T00:00:00

# Explicit type (--type=manual requires --question, --type=time requires --after)
tk wait add -p backyard --type=manual --question="Did the package arrive?"
tk wait add -p backyard --type=time --after=2026-01-15

# List waits
tk waits                       # all open waits in active projects
tk waits -p backyard           # one project
tk waits --actionable          # manual waits ready for user to answer
tk waits --dormant             # waits blocked by incomplete tasks
tk waits --done                # completed waits
tk waits --dropped             # dropped waits
tk waits --all                 # all waits including done/dropped

# Edit wait
tk wait edit BY-03W --title="New title"
tk wait edit BY-03W --question="Updated question?"
tk wait edit BY-03W --check-after=2026-01-20T12:00:00
tk wait edit BY-03W --after=2026-01-20T12:00:00      # for time waits
tk wait edit BY-03W --notes="Tracking number: 123456"
tk wait edit BY-03W --blocked-by=BY-05,BY-06         # replaces blockers
tk wait edit BY-03W --add-blocked-by=BY-07           # adds blocker
tk wait edit BY-03W --remove-blocked-by=BY-05        # removes blocker
tk wait edit BY-03W -i                               # open in $EDITOR (YAML format)

# Resolve manual wait
tk wait resolve BY-03W
tk wait resolve BY-03W --as="Arrived damaged, returning"

# Defer manual wait (push back check_after)
tk wait defer BY-03W --days=4
tk wait defer BY-03W --until=2026-01-20

# Drop wait
tk wait drop BY-03W --reason="No longer needed"
tk wait drop BY-03W --drop-deps --reason="Cancelling order"    # also drop dependent tasks
tk wait drop BY-03W --remove-deps --reason="Changed approach"  # unlink from dependent tasks

# Link task to wait
tk block BY-07 --on=BY-03W           # BY-07 waiting on BY-03W
tk unblock BY-07 --from=BY-03W       # remove wait dependency
```

### Queries

```bash
# Find tasks (case-insensitive substring match in title and notes)
tk find "gravel"               # search all active projects
tk find "gravel" -p backyard   # within project

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

# Output suitable for graphviz: tk graph | dot -Tpng -o deps.png
```

## Completion Behavior

When completing a task (`tk done BY-07`):
- If task has incomplete blockers or open waits: error, suggest `--force`
- With `--force`: removes unfulfilled waits/blockers from the task, completes entire subtree of dependent tasks
- On success: shows what tasks were unblocked and what waits became active
  - Example: "BY-07 done. Unblocked: BY-08, BY-09. Waits now active: BY-03W"
- If task has `auto_complete: true` tasks depending on it, those may auto-complete
- Waits that were blocked by this task become active (dormant → actionable/pending)

When dropping a task (`tk drop BY-07`):
- If task has dependent tasks OR blocked waits: error, suggest `--drop-deps` or `--remove-deps`
- With `--drop-deps`: drops all dependent tasks AND blocked waits recursively
- With `--remove-deps`: removes BY-07 from all downstream blocked_by lists
  - Dependent tasks become unblocked
  - Blocked waits become active (dormant → actionable/pending)
- Requires `--reason` for the drop reason

When dropping a wait (`tk wait drop BY-03W`):
- If wait has tasks waiting on it: error, suggest `--drop-deps` or `--remove-deps`
- With `--drop-deps`: drops all tasks waiting on this wait recursively
- With `--remove-deps`: removes BY-03W from all downstream waiting_on lists
  - Those tasks may become ready if they have no other blockers
- Requires `--reason` for the drop reason

## Error Handling

- Errors go to stderr with non-zero exit code
- Error messages are human-readable and actionable
- No "did you mean" suggestions (keep it simple)

## Output Formatting

- Colored output auto-detected (color if TTY, plain if piped)
- `tk list` shows flat list with status indicators [blocked], [waiting], etc.
- Future enhancement: additional output styles and colors for `tk list`

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
