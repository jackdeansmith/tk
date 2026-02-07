# Orchestrator Session Report

## What we did

Over a single session, we closed 10 of 14 bugs in the `tk` backlog by delegating work to subagents running in parallel git worktrees. I acted as coordinator: creating worktrees, writing prompts, spawning agents, reviewing diffs, resolving merge conflicts, and squash-merging results into main.

### Progression

1. **Take 1 (single agent, DF-01):** Exploratory. Gave minimal prompt, agent fixed the bug but hardcoded magic numbers and wrote no tests. Reverted.

2. **Take 2 (single agent, DF-01 retry):** Added "write tests" to the prompt. Agent independently introduced constants, wrote 9 test functions, and marked the task done. Accepted.

3. **Take 3 (2 agents in parallel, DF-02 + DF-04):** First use of git worktrees for isolation. Both succeeded. One agent forgot to run `tk done`; the prompt had lost that instruction during iteration. Fixed the template.

4. **Take 4 (3 agents in parallel, DF-03 + DF-05 + DF-06):** Smooth run. All three completed, clean diffs, all marked done. Merged serially with no conflicts.

5. **Take 5 (6 agents in parallel, DF-07 + DF-08 + DF-09 + DF-10 + DF-11 + DF-23):** Most ambitious batch. 4 of 6 completed successfully. DF-08 and DF-10 ran out of API credits mid-execution. One merge conflict between DF-09 and DF-23 (both appended tests to the same file), resolved manually.

### Final prompt template

```
You are working in a Go project at {worktree_path}. This is a git worktree
on branch {branch}. The project is a CLI task management tool called `tk`.

Your assignment:
```
{tk_list_line}
```

You can use `go run ./cmd/tk show {task_id}` for details (run from within your worktree).

Your job is to:
1. Understand the bug by exploring the codebase and using tk
2. Fix it with the necessary code changes
3. Write tests that cover your fix
4. Use `go run ./cmd/tk done {task_id}` to mark it complete
5. Commit all changes (including .tk data) to this branch

You are free to do manual testing with `go run ./cmd/tk ...` in your worktree
to verify your fix works. However, keep your .tk data changes clean -- the only
.tk change in your final commit should be {task_id} being marked done. If you
created any test tasks during manual testing, delete them before your final
commit so the diff is clean.

Do NOT push, merge, or modify other branches.
Do NOT create test tasks in the .tk tracker -- use unit tests only.

When you're done, write a brief retrospective about the assignment.
```

### Results

| Task | Description | Status | Agent quality |
|------|-------------|--------|---------------|
| DF-01 | Validate priority range | Merged | Good (take 2) |
| DF-02 | Reject empty titles | Merged | Good |
| DF-03 | Reject empty/whitespace tags | Merged | Good |
| DF-04 | Fix --force hint | Merged | Good |
| DF-05 | Conflicting list filters | Merged | Good |
| DF-06 | Truncate long titles | Merged | Good |
| DF-07 | Document find command | Merged | Good |
| DF-08 | Simplify project new | Not completed | Ran out of credits |
| DF-09 | Rename --as flag | Merged | Good |
| DF-10 | Priority shorthands | Not completed | Ran out of credits |
| DF-11 | Reject newlines in titles | Merged | Good |
| DF-12 | Filter empty tag strings | Not attempted | Held back (file overlap with DF-03) |
| DF-13 | Project delete warning | Not attempted | Held back (file overlap with DF-08) |
| DF-23 | tk note command | Merged | Good |

---

## What went well

**Agents are surprisingly competent with minimal context.** The prompt gives them a one-line task description and tells them where to find details. They consistently explored the codebase, found the right files, followed existing patterns, and wrote reasonable tests. No agent needed hand-holding on architecture or conventions.

**Git worktrees are the right isolation primitive.** They share the object store (so they're fast to create), give each agent a fully independent working directory, and the branches merge cleanly with standard git tooling. No containers, no VMs, no complex setup.

**Squash merging keeps the history clean.** Each agent's branch might have messy intermediate commits, but the squash merge produces one clean commit per task on main. Easy to review, easy to revert if needed.

**The prompt converged quickly.** It took two iterations to get from "agent produces mediocre output" to "agent produces merge-ready output." The key additions were "write tests" and "mark done with tk." After that, the template was stable across all subsequent batches.

**Parallel execution is a real multiplier.** 6 agents running simultaneously, each taking 2-3 minutes, versus doing them serially at ~5 minutes each. Wall-clock time dropped from ~30 minutes to ~4 minutes for 6 tasks.

## What was annoying

**Merge conflicts in the test file.** Multiple agents append tests to the bottom of `commands_test.go`, which guarantees conflicts when merging serially. This is manageable with 2-3 agents but will get worse with more. The conflicts are always trivial (keep both sides), but resolving them manually is tedious.

**The .tk YAML file is a merge conflict magnet.** Every agent marks their task done, which modifies the same YAML file. Git auto-merged these successfully every time (different tasks = different lines), but it's fragile. With enough agents touching adjacent lines, this will break.

**No durable state for the coordinator.** Agent IDs, worktree paths, and branch names live in my context window. If the conversation compacts, I lose track of what's running. We mitigated this with Claude Code's session task tracking, but the real fix is a manifest file on disk.

**Credit exhaustion kills agents silently.** DF-08 and DF-10 ran out of API credits partway through. They left no commits, no partial work, no indication of how far they got. The worktrees just had uncommitted changes that we threw away. There's no way to resume a half-finished agent.

**Code review UX is split-brained.** I can read diffs in the terminal, but the user can't easily. We settled on `code /tmp/tk-worktrees/DF-XX` to open VS Code, but this means the review happens outside the conversation. There's no way to annotate, comment, or discuss specific lines in-band.

**The .gitignore `tk` pattern.** The unanchored `tk` in `.gitignore` matched `cmd/tk/`, requiring `-f` on every `git add`. We fixed this mid-session, but it tripped up agents too. Small environmental issues like this waste agent time.

## What would make it smoother

**A manifest file.** Write `worktrees.json` on agent spawn, read it on resume. Tracks worktree path, branch, agent ID, task ID, and status. Survives compaction and lets the coordinator recover state.

**Structured agent output.** Instead of asking for a "retrospective" as free text, define a schema: files changed, tests added, concerns, suggestions. This would make review briefs more consistent and enable automation.

**Incremental commits from agents.** If agents committed after each logical step (explore, fix, test, mark done), credit exhaustion wouldn't lose all progress. The worktree would have partial work that could be resumed or reviewed.

**A merge-aware test file structure.** Split `commands_test.go` by feature area (e.g., `list_test.go`, `wait_test.go`, `tag_test.go`). This would eliminate the "everyone appends to the same file" conflict pattern. This is a codebase design issue, not an orchestration issue, but orchestration exposes it.

**Agent log persistence.** Save agent transcripts and retrospectives to files in the worktree (`AGENT_LOG.md`) so they survive beyond the coordinator's context window. Useful for post-session review and for building institutional knowledge about what works.

**Resume for interrupted agents.** When an agent dies mid-task (credits, timeout, error), the ability to spawn a new agent in the same worktree with context like "the previous agent got this far, continue from here" would save significant rework.
