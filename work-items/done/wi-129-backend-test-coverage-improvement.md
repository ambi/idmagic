---
depends_on: []
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-05
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "既存の振る舞いに対するテストの追加だけであり、契約も規範シナリオも変えない。" }
initial_context:
  specification: []
  typespec: []
  source:
    - backend/shared/security/tokens_jose
    - backend/signingkeys/keys_memory
    - backend/shared/spec
    - backend/authentication/handlers_http
    - backend/shared/http/support_http
    - backend/sourcing/scim/usecases
    - backend/shared/resilience
    - backend/shared/events/sinks_console
    - backend/shared/observability/telemetry_otlp
    - backend/application/domain
  tests:
    - backend/shared/security/tokens_jose
    - backend/signingkeys/keys_memory
    - backend/shared/spec
    - backend/authentication/handlers_http
    - backend/shared/http/support_http
    - backend/sourcing/scim/usecases
    - backend/shared/resilience
    - backend/shared/events/sinks_console
    - backend/shared/observability/telemetry_otlp
    - backend/application/domain
  stop_before_reading: []
---

# Go バックエンドの残り低カバレッジ領域をまとめて底上げする

## Motivation
Go バックエンドのカバレッジ改善は複数の work item に分割され、`oauth2`、`application` /
`identity-management`、memory persistence、postgres repository の主要部分は完了済みである。
一方で、残っている `crypto`、`spec`、`authentication/adapters/http`、`http/support`、
`scim/usecases`、横断的な小規模 0% パッケージは、個別 WI に分けるほど独立した意味変更ではなく、
同じ「バックエンド低カバレッジの残り刈り取り」としてまとめた方が実装順序と検証が明確になる。

この WI は旧 `wi-137` と `wi-138` を統合した残作業の正本とする。

**実測 (2026-08-22、`mise run test-go-cover`)**。248 パッケージ中 58 が 0%、パッケージ単純平均で約 52%。
本 work item が名指ししている対象の現状は次のとおりで、下の閾値表とはまだ距離がある。

| Package | 現状 |
|---|---|
| `backend/application/domain` | 82.9% |
| `backend/signingkeys/keys_memory` | 75.7% |
| `backend/shared/observability/metrics_prometheus` | 71.4% |
| `backend/shared/http/support_http` | 62.3% |
| `backend/shared/security/tokens_jose` | 62.0% |
| `backend/shared/spec` | 58.4% |
| `backend/authentication/handlers_http` | 29.7% |
| `backend/sourcing/scim/usecases` | 29.4% |
| `backend/shared/resilience` | 0.0% |
| `backend/shared/events/sinks_console` | 0.0% |
| `backend/shared/observability/telemetry_otlp` | 0.0% |

最も遠いのは `authentication/handlers_http` と `sourcing/scim/usecases` の 2 つで、いずれも
拒否・期限切れ・テナント不一致という安全側の分岐を持つ。ここから着手する。

## Scope
- `backend/shared/security/tokens_jose` と `backend/signingkeys/keys_memory` の鍵生成・署名検証・エラー系を中心に、意味のあるユニットテストを追加する。
- `backend/shared/spec` の OpenAPI / specification 由来 spec helper をテストする。
- `backend/authentication/handlers_http` の正常系・異常系・認可 / CSRF / tenant 境界をテストする。
- `backend/shared/http/support_http` の auth、tenant middleware、CSRF、consent、response helper をテストする。
- `backend/sourcing/scim/usecases` の SCIM inbound provisioning use case をテストする。
- `backend/shared/resilience`、`backend/shared/events`、`backend/shared/observability` の横断的関心事をテストする。
- `backend/application/domain` の `ValidateApplication` / `ValidateBinding` などの不変条件をテストする。

## Out of Scope
- フロントエンド (ui) のテスト環境構築およびテスト追加。[[wi-130-frontend-testing-environment-and-coverage]] と [[wi-133-frontend-all-pages-presentation-logic-separation]] で完了済み。
- CI におけるテストカバレッジ強制ルールの適用。→ [[wi-131-testing-governance-and-ci-enforcement]]
- 完了済みの `oauth2`、`application` / `identity-management`、memory persistence、postgres repository の
  カバレッジ改善 WI をやり直すこと。
- カバレッジ数値だけを上げるための、挙動保証を持たない snapshot / smoke test の追加。

## Plan
- 旧 `wi-137` / `wi-138` の対象 package を本 WI に統合し、実装時は依存が少ない domain / helper から始める。
- **着手時に対象一覧を測り直す**。起票からパッケージの配置が二度変わっており (`internal/` → `backend/`、
  Context ごとの縦割りへの再編)、外部ブローカーの撤廃で `internal/relay` と `cmd/idmagic-relay` は
  消滅している。`mise run test-go-cover` の実測を先に取り、下の閾値表を現在のパッケージで作り直す。
- HTTP handler は既存 test helper を再利用し、tenant / CSRF / auth の境界を個別に検証する。
- observability / event sink は外部出力を fake sink に閉じ、ログや metric label に PII が出ないことも確認する。

## Tasks
- [x] T001 [Go] `mise run test-go-cover` で現在の低カバレッジ package を棚卸しし、下の閾値表を更新する。実装着手時に再実測し、Motivation の表の数値と完全に一致することを確認した (更新不要)。
- [x] T002 [Go] `backend/shared/security/tokens_jose` と `backend/signingkeys/keys_memory` と `backend/shared/spec` のユニットテストを追加する。結果: tokens_jose 88.9%、keys_memory 91.6%、spec 84.2% (いずれも閾値超過)。tokens_jose の ES256 DPoP で `jwkThumbprint` が RSA 専用のメンバー (e/kty/n) を前提としており EC 鍵の proof が署名検証後に必ず拒否される既存の制約を発見し、現状の挙動としてテストに固定した (振る舞いは変更せず、別途起票を推奨)。
- [x] T003 [Go] `backend/authentication/handlers_http` と `backend/shared/http/support_http` の HTTP 境界テストを追加する。結果: authentication/handlers_http 29.7% → 74.8%、support_http 62.3% → 85.7% (いずれも閾値超過)。
- [x] T004 [Go] `backend/sourcing/scim/usecases` の provisioning use case テストを追加する。結果: 29.4% → 79.9% (閾値超過)。
- [x] T005 [Go] `backend/shared/resilience` / `backend/shared/events` / `backend/shared/observability` の横断パッケージをテストする。結果: resilience 0% → 95.4%、events/sinks_console 0% → 100%、observability/telemetry_otlp 0% → 94.0% (いずれも閾値超過)。
- [x] T006 [Go] `backend/application/domain` の残り低カバレッジ領域をテストする。結果: 82.9% → 96.1% (`ValidateApplication`/`ValidateProtocol` の不変条件分岐を追加)。
- [x] T007 [Verify] `mise run verify-go` と `mise run test-go-cover` を通し、対象 package の改善を確認する。両方パス。`mise run spec-diff` は "no normative specification change against main" (想定通り無変更)。

## Verification
- `mise run verify-go`
- `mise run test-go-cover` で以下を確認 (T001 の実測で対象と数値を更新してから使う):
  - `backend/shared/security/tokens_jose` >= 75%
  - `backend/signingkeys/keys_memory` >= 75%
  - `backend/shared/spec` >= 75%
  - `backend/authentication/handlers_http` >= 70%
  - `backend/shared/http/support_http` >= 70%
  - `backend/sourcing/scim/usecases` >= 70%
  - `backend/shared/resilience` >= 70%
  - `backend/shared/events` >= 70%
  - `backend/shared/observability` >= 70%
  - `backend/application/domain` >= 80%

## Risk Notes
crypto と authentication HTTP はセキュリティ境界に近いため、正常系だけでなく拒否・期限切れ・tenant 不一致を
明示する。event sink / observability は副作用中心なので fake 実装で観測点を固定し、ログや metric label に
PII が混ざらないことを確認する。カバレッジの数値のみを追い求め、価値の低いテストを量産しないように注意する。

## Completion
- **Completed At**: 2026-08-22
- **Summary**:
  対象 10 package すべてに、拒否・期限切れ・tenant 不一致・CSRF 境界などの安全側分岐を中心とした
  ユニットテストを追加した。既存の振る舞いへの変更は無く (`spec_impact: none` の通り)、契約・規範シナリオへの
  影響も無い (`mise run spec-diff` で無変更を確認済み)。

  | Package | Before | After | 閾値 |
  |---|---|---|---|
  | `backend/shared/security/tokens_jose` | 62.0% | 88.9% | >=75% |
  | `backend/signingkeys/keys_memory` | 75.7% | 91.6% | >=75% |
  | `backend/shared/spec` | 58.4% | 84.2% | >=75% |
  | `backend/authentication/handlers_http` | 29.7% | 74.8% | >=70% |
  | `backend/shared/http/support_http` | 62.3% | 85.7% | >=70% |
  | `backend/sourcing/scim/usecases` | 29.4% | 79.9% | >=70% |
  | `backend/shared/resilience` | 0.0% | 95.4% | >=70% |
  | `backend/shared/events/sinks_console` | 0.0% | 100.0% | >=70% |
  | `backend/shared/observability/telemetry_otlp` | 0.0% | 94.0% | >=70% |
  | `backend/application/domain` | 82.9% | 96.1% | >=80% |

  全 package が閾値を超過した。全体平均 (`mise run test-go-cover` の `total`) は 60.5% → 63.3%。

  実装中に副次的な発見が 1 件あった: `tokens_jose.jwkThumbprint` (RFC 7638 canonical form) が RSA 専用の
  メンバー集合 (`e`/`kty`/`n`) を前提にしており、DPoP が受理を宣言している ES256 (EC 鍵) の proof は
  署名検証成功後もサムプリント計算で必ず拒否される。本 WI はテストの追加のみを scope とするため、この
  制約は現状の挙動としてテストに固定しただけで、修正はしていない。ES256 DPoP を実際に使うクライアントが
  出た時点で別途 work item を起票して直すことを推奨する。
- **Verification Results**:
  - `mise run verify-go` - passed (lint 0 issues, race テスト全 package pass)
  - `mise run test-go-cover` - passed, 上表の通り全対象 package が閾値超過
  - `mise run spec-diff` - "no normative specification change against main"
