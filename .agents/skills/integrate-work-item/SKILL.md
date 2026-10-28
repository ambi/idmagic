---
name: integrate-work-item
description: Integrate a completed work-item branch into main or another target branch by rebasing and fast-forwarding without a merge commit, verify only a newly composed tree, then remove the source branch and its worktree if it had one.
---

# Integrating a completed work item

1. Identify the completed `work-item/<work-item-id>` source branch, the checkout that has it open (the default
   checkout at the repository root, or `.worktrees/<work-item-id>`), and the user-named target branch, `main`
   when none is named. The target may be open in some checkout, or open nowhere when the default checkout holds
   the source branch. Preserve unrelated user changes in every checkout. The source checkout must be clean, its
   work item completed, and its final commit already verified by `implement-work-item`.
2. Record the verified source commit and tree before changing commits or refs:

   ```sh
   source_commit=$(git rev-parse 'work-item/wi-42^{commit}')
   source_tree=$(git rev-parse 'work-item/wi-42^{tree}')
   ```

3. Keep the history linear. When the source is not already based on the target, rebase the source onto the
   target; use `resolving-merge-conflicts` for an in-progress conflict. Compare the rebased source tree with
   the recorded verified source tree. If they differ, the rebase composed a new tree. Verify it in the source
   checkout with:

   ```sh
   mise run verify-integration -- "$source_commit"
   ```

   Each side was verified on its own tree, so the task reruns only the layers and Go packages that both the
   source's own change and the incoming change can reach. It records what it ran and what it skipped in
   `.verification/verify-selection.json`.
   Also rerun the narrow checks that the conflict resolution itself requires.

   Do not create a merge commit. Integrate one source branch at a time, and start the next work item from the
   updated target so that it needs no rebase.
4. Fast-forward the target to the rebased source, then compare the integrated target tree with that verified tree.
   When the target is open in a clean checkout, fast-forward it there; when it is open nowhere, move only the
   ref. Both refuse anything that is not a fast-forward.

   ```sh
   git -C <target-checkout> merge --ff-only work-item/wi-42   # the target is open in that checkout
   git fetch . work-item/wi-42:main                           # the target is open nowhere
   integrated_tree=$(git rev-parse 'main^{tree}')
   test "$integrated_tree" = "$(git rev-parse 'work-item/wi-42^{tree}')"
   ```

   - When the trees are identical, finish immediately. The target contains exactly the already verified result;
     rerunning `check`, `build-site`, or `verify` adds no evidence.
   - A tree difference here means the fast-forward did not produce the verified source result. Stop and inspect
     it before cleanup; a normal fast-forward must be identical.

   Tree identity, not commit identity, controls this decision. Rebase may change commit metadata without
   changing files.
5. Clean up the integrated source once the target contains it and any required verification has passed. When the
   source was open in the default checkout, return that checkout to the target; when it had a worktree, remove it:

   ```sh
   git switch main                        # source in the default checkout
   git worktree remove .worktrees/wi-42   # source in a worktree
   git branch -d work-item/wi-42
   ```

   `git branch -d` refuses a branch the target does not contain, which guards against deleting unintegrated work.
   If `git worktree remove` fails partway, confirm the worktree's `git status --porcelain` lists only ` D` lines
   before retrying with `--force`.
6. Report the integration method, target commit, tree comparison, verification performed, and the removed
   branch and worktree. Leave `spec/generated/` and `site/` uncommitted. Update remote refs or push only when the
   user explicitly asks.
