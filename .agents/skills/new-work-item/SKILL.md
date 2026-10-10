---
name: new-work-item
description: Create a specification-first work item under work-items using the canonical format, requirement/TypeSpec references, tasks, design alternatives, and verification plan.
---

# Creating a work item

1. Read `docs/formats/work-item-format.md` as the authority for the format.
2. `mise run work-item-number` を実行し、`work-items/active/wi-<出力した番号>-kebab-title.md` を作る。
   最大の既存番号を数えない。push 済みの番号しか見えず、並列 worktree で衝突するためである。
3. Write Motivation, Scope, Out of Scope, Design, Plan, Tasks, Verification, and Risk Notes. Surface open
   questions in Design or Plan; resolve every question that would change what gets built before
   implementation, and move genuinely deferred choices to Out of Scope.
4. For `feature`, `bugfix`, and `operations` items, make `affected_spec` a direct reference to a
   normative scenario or standard id, or to a TypeSpec symbol. Do not write `initial_context` when
   filing the item; it is written when the work starts.
5. Keep rejected options and change-specific judgment in Design. Do not create an ADR. When the item adds,
   splits, or merges a module, widens a public package, adds a cross-module dependency, or changes a table
   owner or composition point, apply the boundary selection procedure in
   `docs/design/application/design-guidelines.md` and write its record in Design.
6. Pass `mise run check-work-items`.

完了時は完了節を追記し、`status: completed` にして `mise run move-work-item -- <id>` を実行する。
コマンドが `work-items/done/` への移動と Markdown リンクの更新を行う。
中止時も判断を記録し、`status: cancelled` にして同じコマンドを使う。
移動後は `mise run check-work-items` と `mise run check-links` を通す。
