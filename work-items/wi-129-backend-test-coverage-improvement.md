---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-05
priority: p2
change_kind: tooling
spec_impact: { kind: none, reason: "既存の振る舞いに対するテストの追加だけであり、契約も規範シナリオも変えない。" }
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
- [ ] T001 [Go] `mise run test-go-cover` で現在の低カバレッジ package を棚卸しし、下の閾値表を更新する。
- [ ] T002 [Go] `backend/shared/security/tokens_jose` と `backend/signingkeys/keys_memory` と `backend/shared/spec` のユニットテストを追加する。
- [ ] T003 [Go] `backend/authentication/handlers_http` と `backend/shared/http/support_http` の HTTP 境界テストを追加する。
- [ ] T004 [Go] `backend/sourcing/scim/usecases` の provisioning use case テストを追加する。
- [ ] T005 [Go] `backend/shared/resilience` / `backend/shared/events` / `backend/shared/observability` の横断パッケージをテストする。
- [ ] T006 [Go] `backend/application/domain` の残り低カバレッジ領域をテストする。
- [ ] T007 [Verify] `mise run verify-go` と `mise run test-go-cover` を通し、対象 package の改善を確認する。

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
