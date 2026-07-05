---
id: wi-138-test-coverage-scim-usecases-and-zero-percent-packages
title: scim/usecases および 0% パッケージ群（resilience / eventsink / observability / domain / tenancy / relay）のユニットテスト追加
created_at: 2026-07-05
authors: [tn]
status: pending
risk: low
---

# Motivation

wi-129 のサブタスク E。SCIM Inbound Provisioning のユースケース層（0%）と、resilience / eventsink / observability / application/domain / relay といった小規模だがカバレッジ 0% のパッケージ群にテストが存在しない。
これらはプロビジョニングの中核処理や横断的関心事を担うため、テスト未整備は潜在的リスクとなっている。カバレッジを立ち上げて最低限の品質保証を確立する。

# Scope

- `internal/scim/usecases` — SCIM Inbound Provisioning ユースケースのテストを新規作成し、カバレッジを 0% → 70% 以上に引き上げる
- `internal/shared/resilience` — リトライ / サーキットブレーカー等のテストを新規作成し、カバレッジを 0% → 70% 以上に引き上げる
- `internal/shared/adapters/eventsink` — イベント発行処理のテストを新規作成し、カバレッジを 0% → 70% 以上に引き上げる
- `internal/shared/adapters/observability` — テレメトリ関連テストを新規作成し、カバレッジを 0% → 70% 以上に引き上げる
- `internal/application/domain` — ValidateApplication / ValidateBinding 等の不変条件バリデーションのテストを新規作成し、カバレッジを 0% → 80% 以上に引き上げる
- `internal/relay` — カバレッジを 45.9% → 70% 以上に引き上げる
- `cmd/idmagic-relay` — エントリポイントのカバレッジを 0% → 可能な範囲で向上

# Out of Scope

- scim/domain（100% — 対応不要）
- scim/adapters/http（35.4% — 本 wi 対象外）
- tenancy/usecases（83.5% — 十分な水準）
- tenancy context.go（100% — 対応不要）

# Initial Context

- parent: wi-129-backend-test-coverage-improvement
- packages:
  - internal/scim/usecases
  - internal/shared/resilience
  - internal/shared/adapters/eventsink
  - internal/shared/adapters/observability
  - internal/application/domain
  - internal/relay
  - cmd/idmagic-relay

# Affected Guarantees

- SCIM Inbound Provisioning ロジックの正当性が検証される
- 横断的関心事（resilience, eventsink, observability）の動作が検証される
- application ドメインモデルの不変条件が検証される
- relay プロセスの動作が検証される

# Verification

- `just verify-go`
- `just test-go-cover` で以下を確認:
  - `internal/scim/usecases` ≥ 70%
  - `internal/shared/resilience` ≥ 70%
  - `internal/shared/adapters/eventsink` ≥ 70%
  - `internal/shared/adapters/observability` ≥ 70%
  - `internal/application/domain` ≥ 80%
  - `internal/relay` ≥ 70%

# Risk Notes

scim/usecases のテストではリポジトリ層のモックが必要で、ports インターフェースの理解が前提となる。
0% パッケージの一部（eventsink, observability）はサイドエフェクト中心のため、テスト設計に工夫が要る。
