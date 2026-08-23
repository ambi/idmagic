---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
initial_context:
  specification: [docs/SPECIFICATION.md, docs/contexts/signing-keys/SPECIFICATION.md]
  source: [backend/signingkeys/handlers_http, backend/shared/http/support_http/problem.go]
  tests: [backend/oauth2/handlers_http]
  stop_before_reading: [frontend]
---

# signingkeys context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `signingkeys` context に
適用する。

## Scope

- `backend/signingkeys/handlers_http/admin_key_handler.go` (6 箇所)

## Out of Scope

- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。`docs/contexts/signing-keys/SPECIFICATION.md` の該当 model はいずれも
  既存 Go 実装と status が一致しているため status 変更は不要。

## Plan

1. 既存ハンドラテストのうち body 形式に依存するものを Problem Details 形式へ
   更新し RED を確認する。
2. `WriteBrowserError` → `WriteProblem` に置換する。
3. `just verify` を通す。

## Tasks

- [x] T001 [App] `admin_key_handler.go` を `WriteProblem` へ移行する。
      RED→GREEN: `TestAdminKeysGetUnknownKidReturns404`。
- [x] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

小規模 (1 ファイル・6 箇所)。特記事項なし。

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  `backend/signingkeys/handlers_http/admin_key_handler.go` の 6 箇所を
  `support.WriteProblem` へ置き換えた。status 変更は不要
  (`key_not_found` 404・`key_store_unavailable` 503・
  `active_key_cannot_be_disabled` 400)。
  このパッケージ自身にハンドラテストはないが、`/api/admin/v1/keys` は
  `backend/oauth2/handlers_http/admin_key_handler_test.go` の harness から
  実際に叩かれているため、そこの 404 ケースに Content-Type と `type` URN の
  照合を足して RED を作った。
  仕様変更はない (`just spec-diff`: no normative specification change)。
- **Verification Results**:
  - `just test-go-package ./backend/oauth2/handlers_http` - RED
    (`TestAdminKeysGetUnknownKidReturns404` が
    `Content-Type: application/json` で失敗) → GREEN
  - `just verify` - passed
  - `just spec-diff` - no normative specification change against main
