---
depends_on: []
status: pending
authors: [tn]
risk: medium
created_at: 2026-07-05
---

# Go バックエンドの残り低カバレッジ領域をまとめて底上げする

## Motivation
Go バックエンドのカバレッジ改善は複数の work item に分割され、`oauth2`、`application` /
`identity-management`、memory persistence、postgres repository の主要部分は完了済みである。
一方で、残っている `crypto`、`spec`、`authentication/adapters/http`、`http/support`、
`scim/usecases`、横断的な小規模 0% パッケージは、個別 WI に分けるほど独立した意味変更ではなく、
同じ「バックエンド低カバレッジの残り刈り取り」としてまとめた方が実装順序と検証が明確になる。

この WI は旧 `wi-137` と `wi-138` を統合した残作業の正本とする。

## Scope
- `internal/shared/adapters/crypto` の鍵生成・署名検証・エラー系を中心に、意味のあるユニットテストを追加する。
- `internal/shared/spec` の OpenAPI / SCL 由来 spec helper をテストする。
- `internal/authentication/adapters/http` の正常系・異常系・認可 / CSRF / tenant 境界をテストする。
- `internal/shared/adapters/http/support` の auth、tenant middleware、CSRF、consent、response helper をテストする。
- `internal/scim/usecases` の SCIM inbound provisioning use case をテストする。
- `internal/shared/resilience`、`internal/shared/adapters/eventsink`、`internal/shared/adapters/observability` の横断的関心事をテストする。
- `internal/application/domain` の `ValidateApplication` / `ValidateBinding` などの不変条件をテストする。
- `internal/relay` と `cmd/idmagic-relay` の実行分岐を、外部副作用を避けられる範囲でテストする。

## Out of Scope
- フロントエンド（ui）のテスト環境構築およびテスト追加（`wi-130` / `wi-133` で対応）。
- CI におけるテストカバレッジ強制ルールの適用（`wi-131` で対応）。
- 完了済みの `oauth2`、`application` / `identity-management`、memory persistence、postgres repository の
  カバレッジ改善 WI をやり直すこと。
- カバレッジ数値だけを上げるための、挙動保証を持たない snapshot / smoke test の追加。

## Plan
- 旧 `wi-137` / `wi-138` の対象 package を本 WI に統合し、実装時は依存が少ない domain / helper から始める。
- HTTP handler は既存 test helper を再利用し、tenant / CSRF / auth の境界を個別に検証する。
- observability / eventsink は外部出力を fake sink に閉じ、ログや metric label に PII が出ないことも確認する。
- `cmd/idmagic-relay` は process 起動ではなく、設定解析や分岐を抽出できる範囲でテストする。

## Tasks
- [ ] T001 [Go] `internal/shared/adapters/crypto` と `internal/shared/spec` のユニットテストを追加する。
- [ ] T002 [Go] `internal/authentication/adapters/http` と `internal/shared/adapters/http/support` の HTTP 境界テストを追加する。
- [ ] T003 [Go] `internal/scim/usecases` の provisioning use case テストを追加する。
- [ ] T004 [Go] `internal/shared/resilience` / `eventsink` / `observability` の横断パッケージをテストする。
- [ ] T005 [Go] `internal/application/domain`、`internal/relay`、`cmd/idmagic-relay` の残り低カバレッジ領域をテストする。
- [ ] T006 [Verify] `just verify-go` と `just test-go-cover` を通し、対象 package の改善を確認する。

## Verification
- `just verify-go`
- `just test-go-cover` で以下を確認:
  - `internal/shared/adapters/crypto` >= 75%
  - `internal/shared/spec` >= 75%
  - `internal/authentication/adapters/http` >= 70%
  - `internal/shared/adapters/http/support` >= 70%
  - `internal/scim/usecases` >= 70%
  - `internal/shared/resilience` >= 70%
  - `internal/shared/adapters/eventsink` >= 70%
  - `internal/shared/adapters/observability` >= 70%
  - `internal/application/domain` >= 80%
  - `internal/relay` >= 70%

## Risk Notes
crypto と authentication HTTP はセキュリティ境界に近いため、正常系だけでなく拒否・期限切れ・tenant 不一致を
明示する。eventsink / observability は副作用中心なので fake 実装で観測点を固定し、ログや metric label に
PII が混ざらないことを確認する。カバレッジの数値のみを追い求め、価値の低いテストを量産しないように注意する。
