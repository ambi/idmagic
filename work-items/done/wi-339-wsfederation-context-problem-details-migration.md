---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
initial_context:
  specification: [docs/SPECIFICATION.md, docs/contexts/ws-federation/SPECIFICATION.md]
  source: [backend/wsfederation/handlers_http, backend/shared/http/support_http/problem.go]
  tests: [backend/wsfederation/handlers_http]
  stop_before_reading: [frontend]
---

# wsfederation context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `wsfederation` context に
適用する。

## Scope

- `backend/wsfederation/handlers_http/admin_relying_party_handler.go` (4 箇所)
- `backend/wsfederation/handlers_http/admin_entra_handler.go` (3 箇所)

## Out of Scope

- WS-Federation プロトコル応答 (`wsignin1.0`/`wsignout1.0` の XML/redirect)
  自体の形式変更 (このコンテキストの HTTP エラー JSON API のみが対象)。
- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。`docs/contexts/ws-federation/SPECIFICATION.md` の該当 model はいずれも
  既存 Go 実装と status が一致しているため status 変更は不要。

## Plan

1. 既存ハンドラテストのうち body 形式に依存するものを Problem Details 形式へ
   更新し RED を確認する。
2. `WriteBrowserError` → `WriteProblem` に置換する。
3. `just verify` を通す。

## Tasks

- [x] T001 [App] `admin_relying_party_handler.go`・`admin_entra_handler.go` を
      `WriteProblem` へ移行する。
      RED→GREEN: `TestAdminRelyingParty_RejectsInvalid`。
- [x] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

小規模 (2 ファイル・7 箇所)。特記事項なし。

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  `backend/wsfederation/handlers_http` の `admin_relying_party_handler.go`
  (4 箇所)・`admin_entra_handler.go` (3 箇所) をすべて `support.WriteProblem`
  へ置き換えた。status 変更は不要 (`invalid_request` 400・
  `application_owned_protocol` 409)。WS-Federation プロトコル応答
  (`wsignin1.0`/`wsignout1.0` の XML・redirect、WS-Trust SOAP) は対象外のまま
  変更していない。長さ違反が既に 422 Problem Details で返っていること
  (`admin_relying_party_length_test.go`、wi-128) も引き続き green。
  仕様変更はない (`just spec-diff`: no normative specification change)。
- **Verification Results**:
  - `just test-go-package ./backend/wsfederation/handlers_http` - RED
    (`TestAdminRelyingParty_RejectsInvalid` が
    `Content-Type: application/json` で失敗) → GREEN
  - `just verify` - passed
  - `just spec-diff` - no normative specification change against main
