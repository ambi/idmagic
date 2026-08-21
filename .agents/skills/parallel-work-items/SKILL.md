---
name: parallel-work-items
description: Set up and coordinate parallel git worktrees and branches for multiple specification-first work items. Use when the user wants to implement work items concurrently, create per-work-item workspaces, prepare branch/worktree commands, assign agents to work items, or integrate completed work-item branches back through an integration branch.
---

# Implementing work items in parallel

## Overview

Carry several work items forward in separate git worktrees and branches. Each worktree follows the
`implement-work-item` Skill; derived artifacts are regenerated once, at the integration point, with
the `spec-render` Skill.

## 0. Check the preconditions

1. Read `git status --short` at the repository root. If uncommitted changes exist, separate what
   belongs to this setup from the user's existing work.
2. Open the target work items and read their ids and status. Create anything missing with the
   `new-work-item` Skill first.
3. Decide the base branch. Without direction, use the current branch.
4. Decide where worktrees live. Without direction, create them next to the repository as
   `<repo>-<short-id>`.

## 1. Create branches and worktrees

Create a branch per work item. A branch is the unit of implementation, review, and integration, so do
not settle for a worktree alone.

- branch: `work-item/<work-item-id>`
- worktree: `../<repo>-<work-item-id>`, or shortened to `../<repo>-wi-<nn>`

A new branch:

```sh
git fetch --all --prune
git worktree add -b work-item/wi-42-example ../idmagic-wi-42 <base-branch>
```

An existing branch checked out into another worktree:

```sh
git worktree add ../idmagic-wi-42 work-item/wi-42-example
```

When setting up several work items, confirm each worktree after creating it:

```sh
git worktree list
git -C ../idmagic-wi-42 status --short --branch
```

## 2. Implement in each worktree

Assign each worktree to its own agent or terminal. Keep the instruction short and name both the work
item and the worktree path.

```text
Use the implement-work-item Skill in /path/to/idmagic-wi-42.
Implement work-items/wi-42-example.md end to end, verify it, update completion,
move it to done, and commit the branch. Do not push.
```

Every branch follows the order in the `implement-work-item` Skill.

- Use the `spec-change` Skill first whenever the specification changes.
- Do not commit TypeSpec derived artifacts, such as OpenAPI, on a work-item branch.
- Running `mise run spec-render` to check something is fine; leave the generated files untracked.
- On completion, append `completion` to the work item, move it to `work-items/done/`, and commit.
- Do not push until the user explicitly asks.

## 3. Coordinate while the branches run

Keep each branch inside its work item's `scope` so conflicts surface early.

- When several branches touch the same normative scenario, standard id, or TypeSpec symbol, agree on
  the order first.
- When two branches create the same id or filename, follow `mise run check-ids` and renumber one of them.
- Do not resolve conflicts in generated files by hand. Regenerate from the integrated TypeSpec.

## 4. Integrate

Prepare an integration branch and worktree, then take in the completed branches. Without direction,
name the branch `integration/work-items`.

```sh
git worktree add -b integration/work-items ../idmagic-integration <base-branch>
cd ../idmagic-integration
git merge --no-ff work-item/wi-42-example
git merge --no-ff work-item/wi-43-example
```

Verify the derived TypeSpec artifacts after integrating:

```sh
mise run check
mise run spec-render
mise run verify
```

Do not commit `spec/generated/`, and do not update the release baseline during ordinary integration.

## 5. Clean up

Remove merged worktrees only after the user confirms. Removal deletes working files, so check that
nothing is uncommitted first.

```sh
git -C ../idmagic-wi-42 status --short
git worktree remove ../idmagic-wi-42
```

Delete branches, and push or delete remote refs, only when the user explicitly asks.
