---
name: implement-work-item
description: "選択した work item を、仕様先行、故障に応じた検証境界、発見の反映、検証、完了記録、コミットまで実装する。"
---

# Implementing a work item

1. Implement one work item in its own `work-item/<work-item-id>` branch and
   `.worktrees/<work-item-id>` worktree. If the current worktree is already dedicated to that branch, use it.
   Otherwise inspect the repository root with `git status --short`, preserve the user's existing changes, and
   create the branch and worktree from the user-named base branch or the current branch:

   ```sh
   git worktree add -b work-item/wi-42 .worktrees/wi-42 <base-branch>
   ```

   If the branch already exists and is not checked out elsewhere, attach it instead:

   ```sh
   git worktree add .worktrees/wi-42 work-item/wi-42
   ```

   Confirm the selected branch and a clean item workspace with `git status --short --branch`, then continue
   inside that worktree. Keep worktrees under `.worktrees/`, never `/tmp` or `$TMPDIR`. One record owns one independently
   acceptable outcome, while that record may use multiple commits to keep structural and behavioral changes
   separate as `docs/development/coding-style.md` requires. Then run
   `mise run brief -- <work-item>` and make a **readiness pass** before editing frontmatter. Read what the brief
   names: the work item, its direct normative-scenario and standard references, its TypeSpec symbols, the
   canonical documents those references resolve to, and the smallest code and test slice involved. For a
   standards-coverage item, resolve every scoped standard id to its declaration, debt entry, named tests,
   smallest production entry point, and any active work item that owns missing behavior. If completion needs
   out-of-scope implementation, record the prerequisite or defect and stop before broad checks or implementation.
   Read nothing else until something you have read sends you there. While you read, pay the workspace setup
   once: `mise run setup` (embedded tools including Mermaid, and UI dependencies) and then `mise run lint-go`.
   A fresh worktree lacks both, and the gap otherwise surfaces as a first failure of the final gate. The cold
   lint run is the expensive one, and paying it here leaves every run in step 8 at a few seconds.
2. When the work changes a specification, change it first with `spec-change` and pass `mise run check-spec`.
   When `spec_impact: none` is still correct after the readiness pass, do not invent a specification edit or run
   specification-only baseline gates; use the first check that can actually go RED for the work.
3. Resolve every open question that would change product behavior, the public contract, the selected design
   boundary, or the task breakdown. Move genuinely deferred choices to Out of Scope.
   **Do not split the record because it is large.** Volume is not a reason. Every new record repeats the
   readiness pass, the frontmatter, the Design, and the Completion evidence, so splitting an item N ways
   multiplies the fixed cost while the per-unit cost that made it feel large stays exactly where it was.
   Split only when the record carries changes a reader could accept separately. Give a structural change that
   can land before the behavioral change its own record and pull request; otherwise keep one record and use
   separate commits. When
   high per-unit cost is what makes a record look too big, that cost is the finding: measure what drives it
   (a fixture nobody can compose, an index that does not exist, judgment the tooling discards), and file a
   record against that instead. A repeated unit of work that stays expensive is a tooling defect wearing a
   scheduling costume.
4. Rewrite `initial_context` to the smallest slice actually read during readiness — the brief's draft is a
   starting point, not the answer, and `stop_before_reading` is always yours to decide. It is an audit trail,
   not a reason to read more: leave a category empty instead of opening files only to populate it.
   新規着手では `evidence_policy: risk-based-v4` とし、証拠の契約は
   `docs/development/specification-first-workflow.md`、記入形式は `WORK_ITEM_FORMAT.md` を参照する。
   該当する feature、bugfix、標準対応では、主要ユースケースの観測結果、`fault_model`、
   その故障を検出できる最小の境界とテストを決める。
   それ以外では Acceptance RED と Unit RED、または理由を伴う代替検査を決める。
   着手済みの v3 は既存の契約を維持し、移行するときだけ境界を選び直す。
5. Set the status to `in_progress` and pass `mise run check-work-items`. A later
   normative change returns to step 2; never weaken a scenario to pass code.
6. For changed core logic, make the work item's Design name the principal domain data types and operation
   signatures. Place time, randomness, identifier generation, configuration, persistence, notification, and
   other effects at explicit input, output, or port boundaries.
7. Before the first source or test edit, read `docs/development/coding-style.md`.
   既存の Go コードを変え、触れる範囲の振る舞いを `//spec:covers` を付けたテストが固定していなければ、
   本番コードより先に `TestCharacterize<対象>` の特性化テストで現在の振る舞いを固定し、
   `test-go-mutation` で変更する箇所への変異を検出することを確かめ、本番コードを変えない別のコミットにする。
   手順、検査、変更後の分類はワークフローの「特性化テスト」が定める。
   既存の `TestCharacterize*` が落ちたら、宣言のない振る舞いの変化である。期待値を書き換えて通さず、
   変更を戻すか、分類を利用者へ確認する。
   選択した境界の検査で RED を確認する。
   Domain → Use Cases → Adapters → Infrastructure / UI の必要な範囲を、一つの振る舞いずつ
   最も単純で完全な実装で GREEN にし、GREEN のまま refactor する。
   具体例をデータにすると短くなる場合は、`SPECIFICATION_FORMAT.md` の実行可能な具体例を使う。
   操作列が複雑ならコードのままにし、独立した期待結果と境界固有の表明を共有する。
   仕様にない分岐、エラー、イベント、副作用を書くことになったら、そこで止め、ワークフローの
   「仕様にない振る舞いの分類」に従って (a) 要件にする、(b) 書かない、(c) 実装を直す、のどれかに分け、
   結果を work item の設計に記録する。利用者が観測する結果を変える分類は、実装を進める前に利用者へ確認する。
   (c) は実装方針を切り替える合図であり、仕様を書き足して既存の実装を説明しない。
   分類の後は「実装とテストで見つけた振る舞いの反映」に従って、要件と理由、具体例、生成できる参照を
   それぞれの一次情報へ移す。
   A test claims a declared id with a `//spec:covers <id>[, <id>]: <what it fixes>` directive above the test
   function, and only that shape counts — see Citing a normative id from a test in
   `docs/development/specification-first-workflow.md`. Naming an id in prose claims nothing, so say freely in
   a comment that another record owns one. When the work moves between structural and behavioral changes,
   leave each category GREEN and commit it separately; when they share a pull request, put the structural
   commit first.
   Each time a task in the record reaches GREEN, stage everything and make a checkpoint commit titled
   `checkpoint(<work-item-id>): <task-id> <what is now GREEN>` without asking first. A lost or cleaned-up
   working tree then costs one task, not the whole record. Checkpoints are never pushed and never survive
   into a reviewed commit: before making a real commit — the structural one before behavioral work
   starts, or the final one in step 11 — fold them. Run `git log --oneline <last real commit>..HEAD`,
   confirm every listed commit is a checkpoint, then `git reset --soft <last real commit>` and commit the
   staged result. Interactive rebase is unavailable here, so do not fold with it.
   選択した検査の失敗、テスト名、該当する規範 ID をタスクに残す。For tooling,
   documentation, or pure refactoring without one of those boundaries, record `N/A: <reason>` and the alternate
   check that actually failed instead of inventing a product requirement or test boundary.
   Where the change parses, decodes, splits, normalizes, or compares untrusted input by hand, add a fuzz target
   beside the examples and give it an oracle stronger than "does not panic"; see Properties and fuzzing in
   `docs/development/specification-first-workflow.md`.
8. When modules, public packages, composition points, table owners, structure, technology, runtime
   composition, or core design rules change, use `update-design`; a boundary change also needs the boundary
   selection record that `docs/design/application/design-guidelines.md` defines, made before the code. Keep the feedback loop tight: use `mise run test-go-test -- <package> <test>` or
   `mise run test-ui-unit-file -- <file>` for each RED, GREEN, and fault injection; run the containing package
   once after a coherent behavior is GREEN; use `mise run test-go-changed` after the change crosses package
   boundaries. When several narrow checks are due at once, run them in parallel with
   `mise run -c test-go-package <package> ::: test-go-package <package>`. Drop the `--`: with
   `mise run <task> -- <args> ::: ...` every task after `:::` silently does not run. `-c` lets each task report
   even when another fails. Run the check that owns the layer you just touched while that layer is still what you are looking
   at, rather than saving it for final verification:

   | Just touched | Run next |
   | --- | --- |
   | An admin API or a DTO, after the specification is regenerated | `mise run check-contract-drift`, `mise run check-api-compat` |
   | Go, once one behavior is GREEN | `mise run lint-go` |
   | This record's frontmatter or Completion | `mise run check-work-items` |

   Each of those costs seconds once step 1 has paid the cold run. Run `lint-go` when a behavior reaches GREEN,
   not after every edit, so a lint fix never lands in the middle of a behavior that is still RED. The aggregate
   gates stay where they are, run once at step 10.
   Write the selected recipes into the task so the choice is made once rather than on every red-green turn.
9. Collect the risk-selected change-resistance evidence. Run `mise run test-go-mutation -- <package directory>`
   for changed Go rather than hand-writing the syntactic mutations; hand-write only the faults its operators
   cannot express — wiring removed, a `switch` default replaced, an effect redirected. Read the survivors and
   do not score them; Mutation testing in `docs/development/specification-first-workflow.md` says why. A
   survivor that exposes behavior no requirement states goes through the same (a)/(b)/(c) classification.
10. After every scoped behavior and its evidence can be completed, review the changed code with the seven
    perspectives in `docs/development/coding-style.md`. Then pass `mise run verify` once, and
    `mise run test-ui-e2e` as well when the change can reach the browser: the standard suite no longer starts
    the stack, so a browser regression is otherwise left to CI. Do not run an aggregate gate merely as a
    status check while a prerequisite still prevents completion. Complete
    every evidence field required by `WORK_ITEM_FORMAT.md`, reading the completion summary out of
    `mise run spec-diff`. Set the status to `completed`, pass
    `mise run check-work-items`, and move the file to `work-items/done/`.
11. Fold the remaining checkpoints as step 7 describes, then create any remaining Conventional Commit
    with `commit`. `git log --oneline <base>..HEAD` must show no `checkpoint(` subject afterwards. A record may have multiple commits only where
    separately reviewable structural and behavioral changes require that boundary. Keep all free-form prose in
    the Completion section in the language used by the work item's prose; do not change that section to English
    for the commit. For the final commit only, translate or summarize the Completion Summary into English and
    use it as the commit body rather than writing the diff back out. Report the final branch and commit so the
    verified tree can be handed to `integrate-work-item` without rediscovery. Do not push until explicitly told
    to.

State the Out of Scope items and anything left undone in the final report.

When the user asks for timing analysis, measure decision intervals separately from command `real` time, and
record cold versus cached runs, environmental retries, and approval waits separately; otherwise the numbers
cannot distinguish workflow cost from tool or sandbox cost.
