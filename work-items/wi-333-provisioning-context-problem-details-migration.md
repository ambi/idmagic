---
status: pending
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
---

# provisioning context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `provisioning` context に
適用する。

## Scope

- `backend/provisioning/handlers_http/handlers.go` (4 箇所)
- `backend/provisioning/handlers_http/routes.go` (3 箇所)

## Out of Scope

- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。`spec/contexts/provisioning/SPECIFICATION.md` の該当 model はいずれも
  既存 Go 実装と status が一致している (wi-325 確認済み。
  `ProvisioningConnectionAlreadyExistsError`/`ProvisioningDeliveryNotRetryableError`
  =409、`ProvisioningConnectionNotFoundError`/`ProvisioningDeliveryNotFoundError`
  =404 等) ため status 変更は不要。

## Plan

1. 既存ハンドラテストのうち body 形式に依存するものを Problem Details 形式へ
   更新し RED を確認する。
2. `WriteBrowserError` → `WriteProblem` に置換する。
3. `just verify` を通す。

## Tasks

- [ ] T001 [App] `handlers.go`・`routes.go` を `WriteProblem` へ移行する。
- [ ] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

小規模 (2 ファイル・7 箇所)。特記事項なし。
