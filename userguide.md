# tk User Guide

A personal task management system with first-class support for external blockers.

## Table of Contents

- [Overview](#overview)
- [Installation](#installation)
- [Getting Started](#getting-started)
- [Core Concepts](#core-concepts)
- [Working with Tasks](#working-with-tasks)
- [Working with Waits](#working-with-waits)
- [Dependencies](#dependencies)
- [Shell Completions](#shell-completions)
- [Configuration](#configuration)
- [Command Reference](#command-reference)

## Overview

tk is a lightweight task tracker designed for personal productivity. Unlike traditional todo apps, tk treats **external blockers as first-class citizens**. Need to wait for a package to arrive before you can continue? That's a "wait" in tk, and it's just as important as the tasks themselves.

Key features:
- **Projects** organize related tasks
- **Tasks** represent work you need to do
- **Waits** represent external conditions you're waiting on
- **Dependencies** let tasks and waits block each other
- **Git-friendly** YAML storage for easy version control
- **CLI-first** design that works well for humans and scripts

## Design Philosophy

tk is built around a few core principles:

1. **External blockers are explicit** — "Waiting for a package" is different from "blocked by another task". Waits make this distinction first-class.
2. **Git-friendly storage** — YAML files with sorted IDs produce clean diffs and are easy to inspect by hand.
3. **CLI-first** — Simple commands that work equally well for humans and scripts/AI agents.
4. **Keep everything** — Nothing is deleted; status changes are a permanent record.
5. **Protect against accidents** — Destructive operations require explicit flags.
6. **Radically simple architecture** — No databases, no caching, no daemons. Just YAML files in a `.tk/` directory.

## Installation

Build from source:

```bash
go build -o tk ./cmd/tk
# Move to somewhere in your PATH
mv tk /usr/local/bin/
```

Or install directly with `go install`:

```bash
go install github.com/jacksmith/tk/cmd/tk@latest
```

## Getting Started

### Initialize a New Tracker

Create a `.tk/` directory in your project or home folder:

```bash
tk init
```

This creates a default project. You can customize the initial project:

```bash
tk init --name="Personal Tasks" --prefix=PT
```

### Add Your First Task

```bash
tk add "Buy groceries"
```

Output: `DF-01 Buy groceries`

The ID `DF-01` means this is task #1 in the "Default" project (prefix `DF`).

### List Tasks

```bash
tk list
```

Shows all open tasks in active projects.

### Complete a Task

```bash
tk done DF-01
```

### Create Additional Projects

```bash
tk project new home --prefix=HM --name="Home Improvement"
tk add "Fix leaky faucet" -p HM
```

## Core Concepts

### Projects

A **project** is a container for related tasks and waits. Each project has:

- **ID**: A lowercase identifier (e.g., `home`, `backyard`)
- **Prefix**: 2-3 uppercase letters used in task IDs (e.g., `HM`, `BY`)
- **Name**: A human-readable display name
- **Status**: `active`, `paused`, or `done`

```bash
# List all projects
tk projects

# Show project summary
tk project backyard

# Create a new project
tk project new vacation --prefix=VC --name="Vacation Planning"
```

### Tasks

A **task** represents a unit of work. Each task has:

- **ID**: `{PREFIX}-{NUMBER}` (e.g., `BY-07`)
- **Title**: What needs to be done
- **Status**: `open`, `done`, or `dropped`
- **Priority**: 1 (urgent) to 4 (backlog)
- **Tags**: For categorization (e.g., `weekend`, `errand`)
- **Blockers**: Other tasks or waits that must complete first

Task states are derived from status and blockers:
- **ready**: Open with no incomplete blockers
- **blocked**: Open but waiting on another task
- **waiting**: Open but waiting on a wait
- **done**: Completed
- **dropped**: Abandoned

### Waits

A **wait** represents an external condition you're waiting on. There are two types:

1. **Time waits**: Auto-resolve after a specific date/time
   ```bash
   tk wait add -p BY --after=2026-01-15
   ```

2. **Manual waits**: Require you to answer a question
   ```bash
   tk wait add -p BY --question="Did the package arrive?"
   ```

Wait states:
- **actionable**: Ready for you to check/answer
- **pending**: Waiting for time to pass or check_after date
- **dormant**: Blocked by incomplete items
- **done**: Resolved
- **dropped**: Abandoned

## Working with Tasks

### Adding Tasks

```bash
# Basic task
tk add "Write report"

# With options
tk add "Call plumber" -p HM --priority=1 --tag=urgent --tag=call

# Blocked by another task
tk add "Install faucet" -p HM --blocked-by=HM-01

# With notes and due date
tk add "Submit taxes" --notes="Use TurboTax" --due-date=2026-04-15
```

### Viewing Tasks

```bash
# List open tasks
tk list

# Filter by state
tk list --ready      # Unblocked tasks
tk list --blocked    # Waiting on other tasks
tk list --waiting    # Waiting on waits
tk list --done       # Completed
tk list --all        # Everything

# Filter by project
tk list -p backyard

# Filter by priority
tk list --p1         # Priority 1 (urgent)
tk list --priority=2 # Priority 2 (high)

# Filter by tag
tk list --tag=weekend
tk list --tag=errand --tag=car  # Must have BOTH tags

# Filter by due date
tk list --overdue

# Show task details
tk show BY-07
```

### Editing Tasks

```bash
# Change specific fields
tk edit BY-07 --title="New title"
tk edit BY-07 --priority=2
tk edit BY-07 --notes="Additional context"

# Manage tags
tk tag BY-07 urgent        # Add tag
tk untag BY-07 weekend     # Remove tag
tk edit BY-07 --tags=a,b,c # Replace all tags

# Manage blockers
tk block BY-07 --by=BY-05
tk unblock BY-07 --from=BY-05

# Interactive editing in $EDITOR
tk edit BY-07 -i
```

### Completing and Dropping Tasks

```bash
# Complete a task
tk done BY-07

# Complete multiple tasks
tk done BY-07 BY-08 BY-09

# Force complete (removes incomplete blockers)
tk done BY-07 --force

# Drop a task
tk drop BY-07 --reason="No longer needed"

# Reopen a completed/dropped task
tk reopen BY-07
```

### Deferring Tasks

Create a time wait and link it to a task:

```bash
# Defer for N days
tk defer BY-07 --days=5

# Defer until a specific date
tk defer BY-07 --until=2026-02-01
```

## Working with Waits

### Creating Waits

```bash
# Manual wait (question to answer)
tk wait add -p BY --question="Did the fabric arrive?"

# Manual wait with title
tk wait add "Fabric delivery" -p BY --question="Did the fabric arrive?"

# Manual wait with check-after date
tk wait add -p BY --question="Did the PCBs arrive?" --check-after=2026-01-10

# Time wait (auto-resolves)
tk wait add -p BY --after=2026-01-15
tk wait add "After Jan 15" -p BY --after=2026-01-15T14:00:00
```

### Viewing Waits

```bash
# List open waits
tk waits

# Filter by state
tk waits --actionable  # Ready for you to check
tk waits --dormant     # Blocked by other items
tk waits --done        # Resolved
tk waits --all         # Everything

# Filter by project
tk waits -p backyard

# Show wait details
tk show BY-03W
```

### Resolving Waits

```bash
# Resolve a wait
tk wait resolve BY-03W

# Resolve with description
tk wait resolve BY-03W --as="Package arrived, looks good"
```

### Dropping and Deferring Waits

```bash
# Drop a wait
tk wait drop BY-03W --reason="No longer needed"

# Defer a wait's dates
tk wait defer BY-03W --days=3
tk wait defer BY-03W --until=2026-01-20
```

## Dependencies

### Viewing Dependencies

```bash
# What is blocking this item?
tk blocked-by BY-07

# What is this item blocking?
tk blocking BY-07

# Generate a dependency graph (DOT format)
tk graph
tk graph -p backyard | dot -Tpng -o deps.png
```

### Managing Dependencies

```bash
# Add a blocker to a task
tk block BY-07 --by=BY-05

# Remove a blocker
tk unblock BY-07 --from=BY-05

# Add blockers when creating
tk add "Do thing" --blocked-by=BY-01,BY-02W
```

### Auto-Complete

Tasks with `auto_complete` enabled complete automatically when all blockers are done:

```bash
tk add "Final step" --blocked-by=BY-01,BY-02 --auto-complete
```

### Moving Tasks

Move a task to a different project (task must have no blockers or dependents):

```bash
tk move BY-07 --to=HM
```

## Shell Completions

tk provides dynamic shell completions for commands, task IDs, wait IDs, project names, and tags.

### Bash

Add to your `~/.bashrc`:

```bash
source <(tk completion bash)
```

Or install permanently:

```bash
# Linux
tk completion bash > /etc/bash_completion.d/tk

# macOS (with Homebrew bash-completion)
tk completion bash > $(brew --prefix)/etc/bash_completion.d/tk
```

### Zsh

Add to your `~/.zshrc`:

```bash
# Enable completion system if not already enabled
autoload -U compinit; compinit

# Source tk completions
source <(tk completion zsh)
```

Or install permanently:

```bash
tk completion zsh > "${fpath[1]}/_tk"
```

### Fish

```fish
tk completion fish | source
```

Or install permanently:

```fish
tk completion fish > ~/.config/fish/completions/tk.fish
```

### What Gets Completed

- **Commands**: All tk commands and subcommands
- **Task IDs**: When typing task arguments (e.g., `tk done <TAB>`)
- **Wait IDs**: When typing wait arguments (e.g., `tk wait resolve <TAB>`)
- **Project IDs**: For `--project`/`-p` flag and project commands
- **Tags**: For `--tag`, `--add-tag`, `--remove-tag` flags
- **Blocker IDs**: For `--blocked-by`, `--by`, `--from` flags

## Configuration

Create `.tkconfig.yaml` next to your `.tk/` directory:

```yaml
# Auto-run `tk check` before read commands
autocheck: true

# Default project for `tk add` when -p not specified
default_project: default

# Default priority for new tasks (1-4)
default_priority: 3
```

### Available Options

| Option | Type | Description |
|--------|------|-------------|
| `autocheck` | bool | Auto-resolve time waits on read commands |
| `default_project` | string | Project ID used when `-p` not specified |
| `default_priority` | int | Default priority (1-4) for new tasks |

## Command Reference

### System Commands

| Command | Description |
|---------|-------------|
| `tk init` | Initialize a new .tk/ directory |
| `tk check` | Auto-resolve time-based waits that have passed |
| `tk validate` | Check data integrity |
| `tk validate --fix` | Auto-repair orphan references |
| `tk completion bash\|zsh\|fish` | Generate shell completion script |

### Project Commands

| Command | Description |
|---------|-------------|
| `tk projects` | List all active projects |
| `tk projects --all` | List all projects including paused/done |
| `tk project <id>` | Show project summary |
| `tk project new <id> --prefix=XX --name="Name"` | Create project |
| `tk project edit <id> [options]` | Edit project |
| `tk project delete <id> --force` | Delete project |
| `tk dump <project>` | Export project as plain text |

### Task Commands

| Command | Description |
|---------|-------------|
| `tk add <title> [options]` | Create a new task |
| `tk list [filters]` | List tasks |
| `tk show <id>` | Show task/wait details |
| `tk edit <id> [options]` | Edit a task |
| `tk done <id>...` | Complete task(s) |
| `tk drop <id> [--reason=...]` | Drop a task |
| `tk reopen <id>` | Reopen a done/dropped task |
| `tk defer <id> --days=N\|--until=DATE` | Defer a task |
| `tk move <id> --to=PROJECT` | Move task to another project |
| `tk tag <id> <tag>` | Add a tag |
| `tk untag <id> <tag>` | Remove a tag |

### Wait Commands

| Command | Description |
|---------|-------------|
| `tk waits [filters]` | List waits |
| `tk wait add [title] -p PROJECT --question=...\|--after=...` | Create wait |
| `tk wait edit <id> [options]` | Edit a wait |
| `tk wait resolve <id> [--as=...]` | Resolve a wait |
| `tk wait drop <id> [--reason=...]` | Drop a wait |
| `tk wait defer <id> --days=N\|--until=DATE` | Defer wait dates |

### Dependency Commands

| Command | Description |
|---------|-------------|
| `tk block <id> --by=<blocker>` | Add a blocker |
| `tk unblock <id> --from=<blocker>` | Remove a blocker |
| `tk blocked-by <id>` | Show what blocks an item |
| `tk blocking <id>` | Show what an item blocks |
| `tk graph [-p PROJECT]` | Generate DOT dependency graph |

### Shortcuts

| Command | Equivalent |
|---------|------------|
| `tk ready` | `tk list --ready` |
| `tk waiting` | `tk waits --actionable` |

### Common Options

| Option | Description |
|--------|-------------|
| `-p, --project` | Filter by or specify project |
| `--priority=N` | Set priority (1-4) |
| `--p1`, `--p2`, `--p3`, `--p4` | Priority shortcuts |
| `--tag=TAG` | Filter by or add tag |
| `--blocked-by=IDs` | Set blockers (comma-separated) |
| `--force` | Force operation (skip confirmations) |
| `-i, --interactive` | Edit in $EDITOR |
| `-h, --help` | Show help |

## Tips and Tricks

### Quick Workflows

```bash
# Morning review: what's actionable?
tk ready
tk waiting

# End of day: what did I complete?
tk list --done -p default

# Weekly review: all open items
tk list --all
```

### Command Shortcuts

Commands support unique prefix matching:

```bash
tk li      # tk list
tk ad      # tk add
tk pro     # tk projects
```

### Case-Insensitive IDs

Task IDs are case-insensitive and ignore leading zeros:

```bash
tk show by-7    # matches BY-07
tk show BY-007  # matches BY-07
```

### Dependency Graphs

Visualize your task dependencies:

```bash
# Generate PNG
tk graph | dot -Tpng -o tasks.png

# Generate SVG
tk graph -p backyard | dot -Tsvg -o backyard.svg

# Open directly (macOS)
tk graph | dot -Tpng | open -f -a Preview
```

### Git Integration

Since tk uses YAML files, you can version control your tasks:

```bash
cd ~
tk init
git init .tk
git add .tk
git commit -m "Initial task tracker"
```

### Multiple Trackers

tk looks for `.tk/` in the current directory (no tree walk). Use this to have separate trackers:

```bash
# Personal tasks in home directory
cd ~
tk init

# Work tasks in work directory
cd ~/work
tk init --name="Work Tasks" --prefix=WK
```

## Storage Format

All data is stored as YAML — one file per project containing all tasks and waits for that project.

```
.tk/
  config.yaml           # storage version
  projects/
    BY.yaml             # project "backyard" (prefix BY)
    EL.yaml             # project "electronics" (prefix EL)

.tkconfig.yaml          # user configuration (sibling to .tk/, never auto-generated)
```

Project files are named by their prefix (e.g., `BY.yaml` for prefix "BY"). This means task ID `BY-07` maps directly to file `BY.yaml` for instant lookup.

Each project file contains the project metadata followed by tasks and waits as sorted lists. Tasks and waits are sorted by numeric ID. Null/empty fields are omitted from the YAML output, and multi-line notes use block scalar style for clean diffs.

You can hand-edit these files directly — they're designed to be human-readable. Use `tk validate` afterward to check for any issues.

## Future Directions

Not in v1, but worth considering for the future:

- **JSON output** — `--json` flag for machine-readable output
- **Tree walk** — Walk up the directory tree to find `.tk/` like git does
- **Recurring waits** — "Check on X every week"
- **Time tracking** — Log time spent on tasks
- **Sync protocol** — Conflict resolution for multi-device use
- **Archive command** — Move completed projects out of main storage
- **Summary dashboard** — `tk status` showing counts across all projects
- **REST API** — HTTP server mirroring CLI commands
