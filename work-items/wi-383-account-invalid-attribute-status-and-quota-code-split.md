---
status: pending
authors: [tn]
risk: low
created_at: 2026-08-19
change_kind: bugfix
priority: p2
depends_on: []
affected_spec:
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateUserProfile }
  - { path: spec/contexts/identity-management/models.tsp, symbol: IdMagic.Contract.ActiveJobQuotaExceededError }
---

# 同じ error code が接点によって別の意味を持つ 2 件を実装側で解消する

## Motivation

`wi-382` は契約を実装に合わせる作業だったが、T009 で「同じ code を別 status で返している 3 件」の判断を求められ、そのうち 2 件は実装側の欠陥だと結論した。契約は正しい側を書いてあるので、いま実装と契約がずれているのはこの 2 点である。

**`invalid_attribute` の status。** `PATCH /api/account/v1/profile` は属性スキーマ違反を 400 で返す (`backend/idmanagement/user/handlers_http/account_handler.go`)。同じ違反を `UpdateAdminUser` は 422 で返す。`spec/api-rules.md` の HTTP error responses は 400 を「リクエストを解析できないこと」、422 を「解析できた内容が業務規則に違反すること」と定める。テナントの属性スキーマへの適合は後者なので、422 が正しい。`wi-382` は契約側に 422 を書いた。

**`quota_exceeded` の code。** export の開始 (`handleStartExport`) は「実行中ジョブ数の上限」に達したとき 429 で `quota_exceeded` を返し、共通のエラーハンドラー (`backend/shared/http/support_http/error_handler.go`) は「テナント資源クォータ」に達したとき 422 で同じ `quota_exceeded` を返す。status は両方とも正しい — 前者は待てば通る一時的な状態 (RFC 6585)、後者は解析できた内容の業務規則違反である。欠陥は 1 つの code が 2 つの別概念を指していることで、クライアントは code だけでは再試行すべきかを判断できない。`wi-382` は契約側に `ActiveJobQuotaExceededError` を分けて置き、`type` がいまは同じ URN であることを `@doc` に明記した。

## Scope

- `writeAccountError` の `ErrInvalidAttribute` 分岐を 400 から 422 へ変える。frontend が 400 に依存していないことを確認する。
- 実行中ジョブ数の上限に固有の error code を与え (`active_job_quota_exceeded` 等)、`ActiveJobQuotaExceededError` の `@doc` から重複の注記を外す。
- 変更後の code と status を、対応する `REQ-` シナリオと Go のテストで固定する。

## Out of Scope

- `mfa_enrollment_not_allowed` の 403 と 422。`wi-382` はこれを「別の条件が同じ code を共有している」と判断し、status は双方正しいと結論した。ブラウザー経路の 403 が `ErrMfaAlreadyEnrolled` まで畳んでいる点だけが別の欠陥で、それは `wi-384` が扱う。
- 契約 (`spec/**/*.tsp`) の変更。`wi-382` で既に正しい側を書いてある。

## Verification

- `mise run verify`
- 手動確認: `PATCH /api/account/v1/profile` に属性スキーマ違反を送ると 422 と `urn:idmagic:error:invalid_attribute` が返る。
- 手動確認: 実行中ジョブ数の上限に達した export 開始が、テナント資源クォータとは別の code を 429 で返す。

## Risk Notes

400 から 422 への変更は、`response.status === 400` を見ている frontend の分岐を壊しうる。i18n 辞書は code で引いているので文面は影響を受けないが、分岐の有無を先に確認する。code の分割は、いまの `quota_exceeded` を見ているクライアントがあれば影響する。実在するのは自リポジトリの frontend だけなので、同一コミットで揃える。
