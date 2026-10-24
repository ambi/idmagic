---
name: integrate-work-item
description: Integrate a completed work-item branch into main or another target branch by rebasing and fast-forwarding without a merge commit, verify only a newly composed tree, then remove the source worktree and branch.
---

# Integrating a completed work item

1. Identify the completed `work-item/<work-item-id>` source branch and the user-named target branch. Without a
   named target, use the current branch. Inspect both worktrees and preserve unrelated user changes. The source
   worktree must be clean, its work item completed, and its final commit already verified by
   `implement-work-item`.
2. Record the verified source tree before changing commits or refs:

   ```sh
   source_tree=$(git rev-parse 'work-item/wi-42^{tree}')
   ```

3. Keep the history linear. When the source is not already based on the target, rebase the source onto the
   target; use `resolving-merge-conflicts` for an in-progress conflict. Compare the rebased source tree with
   the recorded verified source tree. If they differ, the rebase composed a new tree. Choose the checks for that
   new tree the way [最終検査の選択](../../../docs/development/specification-first-workflow.md#最終検査の選択)
   chooses them for a change, from two path sets: the source's own changes
   (`git diff --name-only main...work-item/wi-42`) and what the rebase brought in
   (`git diff --name-only "$source_tree" 'work-item/wi-42^{tree}'`).
   - When both sets touch only Markdown under `work-items/` and `docs/`, run `check-work-items` and
     `check-links`, plus `check-agent-guidance` when agent guidance changed. The incoming changes were verified
     on the target, and Markdown cannot change what the code checks observe.
   - When either set touches code, TypeSpec, `mise.toml`, or tooling, run `mise run check` and
     `mise run verify` in the source worktree, because the two sides may interact.
   - Always rerun the narrow checks that the conflict resolution itself requires.

   Do not create a merge commit. Integrate one source branch at a time, and start the next work item from the
   updated target so that it needs no rebase.
4. Fast-forward the target to the rebased source, then compare the integrated target tree with that verified tree:

   ```sh
   integrated_tree=$(git rev-parse 'HEAD^{tree}')
   test "$integrated_tree" = "$source_tree"
   ```

   - When the trees are identical, finish immediately. The target contains exactly the already verified result;
     rerunning `check`, `build-site`, or `verify` adds no evidence.
   - A tree difference here means the fast-forward did not produce the verified source result. Stop and inspect
     it before cleanup; a normal fast-forward must be identical.

   Tree identity, not commit identity, controls this decision. Rebase may change commit metadata without
   changing files.
5. Clean up the integrated source from the target worktree, once the target contains it and any required
   verification has passed:

   ```sh
   git worktree remove .worktrees/wi-42
   git branch -d work-item/wi-42
   ```

   `git branch -d` refuses a branch the target does not contain, which guards against deleting unintegrated work.
   If `git worktree remove` fails partway, confirm the worktree's `git status --porcelain` lists only ` D` lines
   before retrying with `--force`.
6. Report the integration method, target commit, tree comparison, verification performed, and the removed
   worktree and branch. Leave `spec/generated/` and `site/` uncommitted. Update remote refs or push only when the
   user explicitly asks.
