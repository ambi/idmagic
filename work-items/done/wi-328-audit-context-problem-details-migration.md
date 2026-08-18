---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
initial_context:
  specification: [spec/SPECIFICATION.md, spec/contexts/audit/SPECIFICATION.md]
  source: [backend/audit/handlers_http, backend/shared/http/support_http/problem.go]
  tests: [backend/audit/handlers_http]
  stop_before_reading: [frontend]
---

# audit context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `audit` context に適用する。

## Scope

- `backend/audit/handlers_http/admin_audit_event_handler.go` (起票時 5 箇所、
  着手時点で 7 箇所)

## Out of Scope

- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。`spec/contexts/audit/SPECIFICATION.md` の該当 `kind: error` model はいずれも
  既存 Go 実装と status が一致しているため (wi-325 確認済み)、status 変更は不要。

## Plan

1. 既存ハンドラテストのうち body 形式に依存するものを Problem Details 形式へ
   更新し RED を確認する。
2. `WriteBrowserError` → `WriteProblem` に置換する。
3. `just verify` を通す。

## Tasks

- [x] T001 [App] `admin_audit_event_handler.go` を `WriteProblem` へ移行する。
      RED→GREEN: `TestAdminAuditEventsRejectsUnknownFilterField`。
- [x] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

小規模 (1 ファイル・5 箇所)。特記事項なし。

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  `backend/audit/handlers_http/admin_audit_event_handler.go` の 7 箇所
  (起票時の調査では 5 箇所、その後の変更で 2 箇所増えていた) をすべて
  `support.WriteProblem` へ置き換え、audit context の管理 API エラーが
  `application/problem+json` を返すようになった。`invalid_request` 400 と
  `event_not_found` 404 のみで、status を変える call site はなかった。
  仕様変更はない (`just spec-diff`: no normative specification change)。
- **Verification Results**:
  - `just test-go-package ./backend/audit/handlers_http` - RED
    (`TestAdminAuditEventsRejectsUnknownFilterField` が旧 envelope のまま失敗)
    → GREEN
  - `just verify` - passed
  - `just spec-diff` - no normative specification change against main
