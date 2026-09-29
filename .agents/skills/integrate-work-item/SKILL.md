---
name: integrate-work-item
description: Integrate a completed work-item branch into main or another target branch by fast-forward, rebase, or merge, and verify only when integration produces a tree that was not already verified, then remove the source worktree and branch.
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

3. Choose the history operation requested by the user. When no method is named, use `git merge --ff-only` if
   the target is an ancestor of the source; otherwise merge. If the user requests linear history and the
   branches diverged, rebase the source onto the target and then fast-forward the target. Use
   `resolving-merge-conflicts` for an in-progress conflict. Integrate one source branch at a time.
4. Confirm the expected source and target refs, then compare the integrated target tree with the verified tree:

   ```sh
   integrated_tree=$(git rev-parse 'HEAD^{tree}')
   test "$integrated_tree" = "$source_tree"
   ```

   - When the trees are identical, finish immediately. The target contains exactly the already verified result;
     rerunning `check`, `spec-render`, or `verify` adds no evidence.
   - When the trees differ, integration created a new composition. Run the narrow checks required by conflict
     resolution, then `mise run check` and `mise run verify`. Run `spec-render` only when the composed change
     affects TypeSpec or its design inputs.

   Tree identity, not commit identity, controls this decision. A merge or rebase may change commit metadata
   without changing files.
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
