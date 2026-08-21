---
name: commit
description: Commit the current working-tree changes with a proper Conventional Commits message (English subject + multi-line English body). Use when the user asks to commit — e.g. "commit", "コミットして", "これをコミット". Splits into multiple commits when the diff mixes clearly unrelated concerns.
---

# Committing changes

Commit the current working tree under Conventional Commits. Do not push until asked.

## Steps

1. **See what changed.** Read `git status`, `git diff` (both staged and unstaged), and untracked files.
2. **Group by meaning.**
   - Split into **several commits** when the diff carries **genuinely unrelated concerns**
     (a feature plus an unrelated tooling setting, for example).
   - **Keep loosely related work in one commit.** Do not slice it thinner than the meaning warrants.
   - Stage a split by file or path (`git add <paths>`). Do not use interactive hunk staging
     (`git add -p`) in this environment.
3. **Write the message for each commit** in the format below. **Subject and body are both English.**
4. Run `git commit`. For a single commit, stage everything; for a split, stage and commit one group
   at a time.
5. Confirm the result with `git log --oneline -n <count>` and report it.

## Message format (Conventional Commits)

**Always English. Never stop at one line — the body carries the *why* in several lines of detail.**

```
type(scope): summary

Why this change is needed / what problem it solves (not a restatement of the diff).
- key change 1
- key change 2
```

- `type`: one of `feat` `fix` `docs` `refactor` `chore` `test` `perf` `build` `ci` `style`. Mark a
  breaking change with `!` (for example `feat(idmagic)!: ...`).
- `scope`: the context or module that changed (`idmagic`, `tools`, `spec`, …).
- **Subject ≤ 72 characters**, imperative mood.
- The body states *why*, not *what*. The diff already says what changed.
- **Do not repeat the mistake of an English subject with a Japanese body. Both are English.**

## Cautions

- **Attribution follows the repository settings (`.claude/settings.json`).** Never add a
  `Co-authored-by` or similar footer by hand.
- Commit on the current branch; this repository commits to its default branch directly.
- Leave TypeSpec derived artifacts untracked, and do not update the release baseline in an ordinary
  commit. A work-item commit in a parallel worktree normally carries no generated files. A commit that
  lands on integration or main includes the result of `mise run spec-render`, keeping TypeSpec and its
  derived artifacts in step (see the `spec-render` Skill).
