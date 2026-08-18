---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
initial_context:
  specification: [spec/SPECIFICATION.md, spec/contexts/workloadidentity/SPECIFICATION.md]
  source: [backend/workloadidentity/handlers_http, backend/shared/http/support_http/problem.go]
  tests: [backend/workloadidentity/handlers_http]
  stop_before_reading: [frontend]
---

# workloadidentity context の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を `workloadidentity` context に
適用する。`wi-325` で `workloadidentity.yaml` に新設した 7 個の granular
error model (いずれも 422) に対応する Go 呼び出し箇所も、この移行と同時に
status を揃える。

## Scope

- `backend/workloadidentity/handlers_http/routes.go` (15 箇所)

現状 400 で specification は 422 を宣言している code (実装時に status も変更する、
`routes.go` 341〜357行):
- `workload_trust_bundle_jwks_required`
- `workload_trust_bundle_name_required`
- `workload_trust_bundle_issuer_required`
- `workload_trust_bundle_audiences_required`
- `workload_trust_bundle_invalid_ttl`
- `agent_workload_binding_agent_not_found`
- `agent_workload_binding_pattern_required`

上記以外の残り 8 箇所は既存 status のまま envelope だけ変える。

## Out of Scope

- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- `support.WriteBrowserError(...)` を `support.WriteProblem(...)` に機械的に
  置き換える。上記 7 箇所は置換と同時に `http.StatusBadRequest` を
  `http.StatusUnprocessableEntity` へ変更する。

## Plan

1. 既存ハンドラテストのうち body 形式・status に依存するものを Problem
   Details/新 status 前提へ更新し RED を確認する。
2. `WriteBrowserError` → `WriteProblem` に置換し、上記 7 箇所は status も揃える。
3. `just verify` を通す。

## Tasks

- [x] T001 [App] `routes.go` の 15 箇所を `WriteProblem` へ移行し、
      granular 7 model の status を 422 に揃える。
      RED→GREEN: `TestRegisterTrustBundleRejectsMissingName` (新規)。
- [x] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`

## Risk Notes

単一ファイル 15 箇所。trust bundle 系と agent-workload binding 系で status
変更対象/非対象が入り混じるため、行単位で `wi-325` Design 節の対応表と
突き合わせて確認すること。

## Completion

- **Completed At**: 2026-08-19
- **Summary**:
  `backend/workloadidentity/handlers_http/routes.go` の 15 箇所を
  `support.WriteProblem` へ置き換え、granular 7 code
  (`workload_trust_bundle_jwks_required`・`workload_trust_bundle_name_required`・
  `workload_trust_bundle_issuer_required`・
  `workload_trust_bundle_audiences_required`・`workload_trust_bundle_invalid_ttl`・
  `agent_workload_binding_agent_not_found`・
  `agent_workload_binding_pattern_required`) の status を 400 から仕様の
  宣言値 422 に揃えた。残り 8 箇所 (`invalid_request` 400、
  `*_not_found` 404、`*_conflict` 409) は status 据え置き。
  このパッケージには HTTP レベルのテストが 1 つもなかったため、
  `routes_test.go` を新設して trust bundle 登録の name 欠落ケースを固定した。
  仕様変更はない (`just spec-diff`: no normative specification change)。
- **Verification Results**:
  - `just test-go-package ./backend/workloadidentity/handlers_http` - RED
    (`TestRegisterTrustBundleRejectsMissingName` が status 400・旧 envelope で
    失敗) → GREEN
  - `just verify` - passed
  - `just spec-diff` - no normative specification change against main
