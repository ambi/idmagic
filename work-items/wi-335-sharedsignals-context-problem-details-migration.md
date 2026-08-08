---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-08-08
depends_on: [wi-326-http-error-responses-rfc9457-migration]
---

# sharedsignals context の admin stream API の `WriteBrowserError` 呼び出しを Problem Details へ移行する

## Motivation

`wi-326` が確立した RFC 9457 Problem Details 移行パターン (ADR-154、
`backend/shared/http/support_http/problem.go`、参照実装
`backend/apitoken/handlers_http/routes.go`) を SharedSignals の admin stream
管理 API に適用する。inbound SET receiver (`POST /ssf/streams/:id/events`) は
`spec/contexts/sharedsignals.yaml` の `error_format: set_delivery`
(RFC 8935 §2.3 が `{err, description}` 固定形式を MUST で規定、`wi-325` で
確認済み) のため対象外 (ADR-154、`wi-326` Scope)。`wi-325` で新設した 7 個の
granular error model (いずれも 422) に対応する Go 呼び出し箇所も、admin API
側はこの移行と同時に status を揃える。

## Scope

`backend/sharedsignals/handlers_http/routes.go` の `writeAdminSharedSignalsError`
(313〜329行) のうち、**inbound SET receiver 以外** の 11 箇所:

- `ssf_stream_not_found` (404、status 変更なし)
- `ssf_stream_event_types_required`・`ssf_stream_event_type_invalid`・
  `ssf_transmitter_delivery_endpoint_invalid`・`ssf_transmitter_audience_required`・
  `ssf_receiver_trusted_issuer_invalid`・`ssf_receiver_jwks_required`・
  `ssf_receiver_accepted_audiences_required` (7 箇所、現状 400 → SCL 宣言値 422
  へ揃える)
- `handleRegisterTransmitterStream`/`handleRegisterReceiverStream`/
  `handleUpdateStream` の `invalid_request` (3 箇所、status 変更なし)

## Out of Scope

- `handleReceiveSecurityEvent` (75〜81行、328〜329行) の
  `security_event_token_too_large`・`security_event_rejected` の 2 箇所
  (inbound SET receiver、RFC 8935 `set_delivery` 固定形式) — **Problem Details
  化しない**。
- `WriteBrowserError` 自体の削除 (全コンテキスト移行完了後の別 work item)。

## Design

- 対象 11 箇所を `support.WriteProblem(...)` に置き換える。
  granular 7 model は `http.StatusBadRequest` → `http.StatusUnprocessableEntity`
  も同時に変更する。
- `handleReceiveSecurityEvent` 側の 2 箇所は一切変更しない。

## Plan

1. 既存ハンドラテストのうち body 形式・status に依存するものを Problem
   Details/新 status 前提へ更新し RED を確認する (inbound SET receiver の
   テストは変更しない)。
2. admin stream API 側 11 箇所を `WriteProblem` へ置換し、granular 7 model の
   status を 422 に揃える。
3. `just verify` を通す。

## Tasks

- [ ] T001 [App] `writeAdminSharedSignalsError` の 11 箇所を `WriteProblem` へ
      移行し、granular 7 model の status を 422 に揃える。
      `handleReceiveSecurityEvent` の 2 箇所には触れない。
- [ ] T002 [Verify] `just verify` を通す。

## Verification

- `just verify-go`
- 回帰確認: `POST /ssf/streams/:id/events` (inbound SET receiver) のエラー
  レスポンスが引き続き `{err, description}` 形式・`application/json`・400 で
  あることを確認する (Problem Details 化されていないこと)。

## Risk Notes

- 同一ファイル内に admin API (移行対象) と inbound SET receiver (対象外) が
  混在するため、置換時に `handleReceiveSecurityEvent` 配下の呼び出しへ
  誤って手を入れないこと。
- 実装調査で判明: `handleReceiveSecurityEvent` が現在使っている
  `WriteBrowserError` の body は `{error, message}` であり、RFC 8935 §2.3 が
  MUST で要求する `{err, description}` フィールド名とは既に一致していない
  (SCL は `error_format: set_delivery` を正しく宣言しているが Go 実装が
  追従していない、`wi-325` 完了時点からの既知の乖離)。この修正は RFC 8935
  形式そのものへの対応であり Problem Details 移行とは別種の変更のため
  **本 work item のスコープ外**。気づいた時点で記録に残す — 別途 work item
  化を検討すること。
