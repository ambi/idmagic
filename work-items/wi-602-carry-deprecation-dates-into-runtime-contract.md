---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: tooling
spec_impact:
  kind: none
  reason: "非推奨の API 操作が現時点で存在しないため、TypeSpec の宣言もレスポンスヘッダーも変わらない。変わるのは、宣言したときにヘッダーが付与されるまでの伝達経路である。"
---

# TypeSpec の非推奨日をランタイムの契約へ伝達する

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「非推奨の宣言」は、`deprecated_since` を設定したインターフェースに `Deprecation` ヘッダーを、`sunset_at` を設定したインターフェースに `Sunset` ヘッダーを付与すると定める。
ヘッダーを付与する `support_http.DeprecationHeadersMiddleware` は存在するが、`CurrentRuntimeContract()` は `Deprecations` を常に空のマップで返す。
`tools/generate-contract` は `deprecated` の真偽値のみを生成し、日付を運ばない。
このため、TypeSpec で API 操作を非推奨にしても、ヘッダーは付与されない。
非推奨の API 操作がまだ存在しないので、この欠落は観測されていない。

## Scope

- TypeSpec に非推奨の開始日と廃止日を宣言する方法を定める。
- `tools/generate-contract` がその日付を `RuntimeContract.Deprecations` へ生成する。
- `sunset_at` が `deprecated_since` から 12 か月以上後であることを生成時に検査する。
- 非推奨の API 操作を宣言した場合にヘッダーが付与されることを、テスト用の契約で固定する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- 実際の API 操作の非推奨化。

## Design

TypeSpec の `#deprecated` は理由の文字列しか持たないため、日付は OpenAPI 拡張（`x-deprecated-since`、`x-sunset-at`）として宣言する。
生成器は OpenAPI の `deprecated` とこれらの拡張の組み合わせを検証し、`deprecated` のない API 操作に日付がある場合や、その逆の場合を拒否する。

## Tasks

- [ ] T001 [Acceptance] 日付を宣言したテスト用の契約で、ヘッダーが付与されないことを RED として記録する。
- [ ] T002 [Tooling] 生成器を改める。
- [ ] T003 [Tooling] 12 か月の検査を加える。
- [ ] T004 [Docs] API ガイドラインの適用状況を改める。
- [ ] T005 [Verify] 変更を検証する。

## Verification

- `mise run check-generated-contract`
- `mise run test-tools`
- `mise run verify`

## Risk Notes

生成器の出力形式を変えると、`operations_gen.go` の差分が大きくなる。生成物の再現性を `mise run check-generated-contract` で確かめる。
