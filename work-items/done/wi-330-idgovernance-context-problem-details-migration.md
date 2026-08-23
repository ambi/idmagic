---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
initial_context:
  specification: [docs/SPECIFICATION.md, docs/contexts/identity-governance/SPECIFICATION.md]
  source: [backend/idgovernance/handlers_http, backend/shared/http/support_http/problem.go]
  tests: [backend/idgovernance/handlers_http]
  stop_before_reading: [frontend]
---

# idgovernance context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `idgovernance` context に適用する。

## Scope

- `backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go` (10 箇所)

## Out of Scope

- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。`docs/contexts/identity-governance/SPECIFICATION.md` の該当 model は
  wi-325 でこのコンテキストの granular error model を新設していないため、
  status 変更は不要 (`WorkflowRevisionConflictError` 等は元々 409 が既に
  Go 実装と一致している想定 — 実装時に個別確認する)。

## Plan

1. 既存ハンドラテストのうち body 形式に依存するものを Problem Details 形式へ
   更新し RED を確認する。
2. `WriteBrowserError` → `WriteProblem` に置換する。
3. `just verify` を通す。

## Tasks

- [x] T001 [App] `admin_lifecycle_workflow_handler.go` を `WriteProblem` へ移行する。
      RED→GREEN: `TestAdminLifecycleWorkflowDryRunReflectsActualUserState`。
- [x] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

小規模 (1 ファイル・10 箇所)。特記事項なし。

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  `backend/idgovernance/handlers_http/admin_lifecycle_workflow_handler.go` の
  10 箇所をすべて `support.WriteProblem` へ置き換えた。Design の想定どおり
  status を変える call site はなかった (`workflow_revision_conflict`/
  `workflow_name_conflict` は既に 409、`workflow_not_found` は 404、
  残りは `invalid_request`/`invalid_workflow` の 400)。
  既存ハンドラテストは status しか固定していなかったため、dry_run の
  存在しないユーザー指定ケースに Content-Type と `type` URN の照合を足して
  RED を作った。
  仕様変更はない (`just spec-diff`: no normative specification change)。
- **Verification Results**:
  - `just test-go-package ./backend/idgovernance/handlers_http` - RED
    (`TestAdminLifecycleWorkflowDryRunReflectsActualUserState` が
    `Content-Type: application/json` で失敗) → GREEN
  - `just verify` - passed
  - `just spec-diff` - no normative specification change against main
