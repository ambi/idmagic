---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-07-04
priority: p2
change_kind: feature
affected_spec:
  - { path: docs/contexts/system/scenarios.feature.md, requirement: REQ-SYSTEM-001 }
---

# OpenTelemetry の伝播をデータベース、外部 HTTP、非同期ジョブまで延ばす

## Motivation

API サーバーとワーカーには、OpenTelemetry の Provider、OTLP exporter、W3C Trace Context の伝播、HTTP 受信スパン、ログとの相関、Collector の構成がすでにある。

残っている問題は、受信したトレースが PostgreSQL、外部 HTTP、重要なユースケース、非同期ジョブの境界で途切れることである。

認証要求からデータベース操作や後続ジョブまでを一つの因果関係として追えなければ、サービス目標の逸脱を検知できても、遅延や失敗が生じた境界を特定できない。

本項目は既存の基盤を置き換えず、依存境界の子スパンと非同期の伝播を完成させる。

## Scope

- 現在の HTTP、PostgreSQL、外部 HTTP、ユースケース、ジョブ境界を棚卸しし、欠けている伝播とスパンを確定する。
- PostgreSQL と外部 HTTP の呼び出しに、値や資格情報を記録しない子スパンを追加する。
- 認証、トークン発行など、複数の依存を調整する重要なユースケースだけに手動スパンを追加する。
- ジョブの永続化境界で trace context を専用メタデータとして保存し、ワーカー側で producer span へ link した consumer span を開始する。
- 再試行回数、結果、低カーディナリティのエラー分類を記録し、tenant、user、token、IP、SQL bind 値を属性へ含めない。
- exporter の停止、キューの飽和、不正なリモート親が製品要求を失敗させず、観測可能な drop として扱われることを検証する。
- `docs/observability.md` にスパン分類、伝播境界、属性の許可リスト、サンプリングと障害時の扱いを記録する。

## Out of Scope

- 既存の Provider、OTLP exporter、Collector、HTTP 受信スパンを別実装へ置き換えること。
- 特定の商用 APM 向けライブラリの導入。
- プロファイラの常時監視。
- すべての関数へ手動スパンを追加すること。
- トレース情報を業務ペイロードとして公開すること。

## Design

Provider、propagator、resource、shutdown の所有者は引き続き composition root とし、各 Context は OpenTelemetry の大域設定を変更しない。

domain model は OpenTelemetry に依存しない。

ユースケースと依存ポートは `context.Context` を渡し、HTTP client と PostgreSQL の adapter が子スパンを作る。

ジョブでは業務パラメーターと trace context を分離する。

再試行は同じ処理の再開ではなく新しい試行として consumer span を作り、元の producer span との link と `retry_attempt` を持たせる。

属性は許可リスト方式とし、route template、method、status、operation、error class などに限定する。

## Tasks

- [ ] T001 [Inventory] 現在の計装と HTTP、PostgreSQL、外部 HTTP、ユースケース、ジョブ境界を棚卸しする。
- [ ] T002 [Docs] スパン分類、伝播、属性、サンプリング、障害時の扱いを `docs/observability.md` に記録する。
- [ ] T003 [Acceptance RED] リモート親から PostgreSQL とジョブまで同じ因果関係で追えない現状をテストで固定する。
- [ ] T004 [Storage/HTTP] PostgreSQL と外部 HTTP の adapter に子スパンと安全な属性を追加する。
- [ ] T005 [Usecase] 選定した重要な調整処理へ手動スパンを追加する。
- [ ] T006 [Async] ジョブの専用メタデータへ trace context を保存し、ワーカーの consumer span、link、再試行属性を実装する。
- [ ] T007 [Verify] リモート親、正常処理、エラー、再試行、exporter 障害、属性漏えいを検証する。

## Verification

- `mise run test-go-race`
- `mise run verify-go`
- `mise run check-monitoring`
- `mise run verify`
- in-memory exporter を用いたテストで、HTTP 受信、依存呼び出し、ジョブ処理の親子関係または link を確認する。
- トレース属性とログを走査し、禁止した識別子、資格情報、SQL bind 値が含まれないことを確認する。

## Risk Notes

リスクは low であり、観測を追加するだけで製品の成功条件は変えない。

最大の危険は、便利な識別子を属性へ載せて秘密情報や個人情報を外部の観測基盤へ流すことである。

属性を許可リストに限定し、exporter の失敗は要求処理から切り離す。
