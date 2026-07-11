---
depends_on: [wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]
status: pending
authors: ["tn"]
risk: critical
created_at: 2026-07-10
---

# どの単一障害でも認証を止めないマルチAZ・自動フェイルオーバー・過負荷保護・ゼロダウンタイム移行のトポロジを整備する

## Motivation
idmagic は IdP として、ダウンすると全依存システムのログインが止まる単一障害点である。
既存の到達点は障害耐性の**部品**を揃えているが、「どのケースでも可用性を失わない」全体設計にはなっていない:
[[wi-98-kubernetes-health-probes-and-graceful-drain]] はプローブと drain、
[[wi-106-distributed-login-throttle-and-shared-state-ha]] は共有状態、
[[wi-108-database-connection-resilience-circuit-breaker]] は DB 接続耐性、
[[wi-101-backup-restore-and-disaster-recovery]] はバックアップ / DR runbook を扱う。
しかし wi-101 は**マルチリージョン アクティブ／アクティブ**と**自動フェイルオーバーのオーケストレーション**を
明示的に out of scope とし、他のどの WI もこれを扱っていない。

結果として (1) AZ / ノード障害時の app・Valkey・PostgreSQL の冗長化と自動昇格、
(2) 依存（DB / Valkey）部分障害時に「どの機能を落として何を守るか」の縮退マトリクス、
(3) 集中負荷・スパイク時の load shedding / backpressure による過負荷での連鎖崩壊防止、
(4) スキーマ移行・ローリングデプロイをダウンタイム無しで行うための前後方互換の規約、
(5) フェイルオーバーが実際に機能することの検証（chaos / drill）、が未定義のままである。
この WI は [[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]] の参照アーキテクチャ上で
可用性を最大化し、単一障害で認証が止まらない状態を設計・検証で保証する。

## Scope
- **scl**:
  - `System.objectives` に可用性トポロジ目標を追加する: 許容する同時障害単位（1 AZ 喪失 / 1 DB ノード喪失 / N app レプリカ喪失で無停止）、フェイルオーバー時の RTO/RPO（[[wi-101-backup-restore-and-disaster-recovery]] の DR runbook と整合、こちらは自動フェイルオーバー時の目標）、既存の `OverallAvailability` 99.9% を満たすための冗長度。
  - `guarantees` に「依存部分障害時の縮退は fail-safe に定義され、認証コアは可能な限り継続、保証できない機能は fail-closed で拒否する」を明文化する。
  - `constraints` に「スキーマ migration とデプロイは前後方互換（N/N+1 が同時稼働可能）」を規約化する。
  - `scenarios` に AZ 障害・primary DB 障害・Valkey 障害・スパイク過負荷・ローリングデプロイ中のリクエスト、の各シナリオを追加する。
- **decision**:
  - 新規 ADR (HA / フェイルオーバートポロジ): app のマルチAZ 配置と anti-affinity、PostgreSQL の primary/standby 自動昇格方式、Valkey の HA / 自動フェイルオーバー、LB ヘルスチェックと [[wi-98-kubernetes-health-probes-and-graceful-drain]] readiness の連携、マルチリージョン到達目標（まず単一リージョン・マルチAZ を必達、リージョン喪失は [[wi-101-backup-restore-and-disaster-recovery]] の DR で受ける分界）を記録する。
  - 新規 ADR (縮退・過負荷保護): 依存部分障害時の機能縮退マトリクス、load shedding / concurrency limit / backpressure の適用点と閾値、retry / timeout / circuit breaker（[[wi-108-database-connection-resilience-circuit-breaker]] を横断適用）の統一方針を記録する。
- **go/usecase / http**:
  - 過負荷保護ミドルウェア（同時実行上限・キュー上限超過時の即時 503 + Retry-After、優先度の低い経路の load shedding）を追加する。認証コアと重い集計・エクスポートを分けて後者を先に落とす。
  - 依存縮退を判定する degradation state を readiness / metrics（[[wi-112-prometheus-metrics-and-authentication-golden-signals]] 相当）に反映する。
  - スキーマ前後方互換を守るため、破壊的移行を expand/contract 2 段に分ける規約をコード / migration に適用する。
- **tooling / verification**:
  - ローカル docker compose で「app レプリカ 1 台 kill」「PostgreSQL primary 停止 → standby 昇格」「Valkey ノード停止」「負荷スパイク」を再現する failover / chaos drill recipe を `deploy` 配下に置く。
  - ゼロダウンタイム デプロイ drill（N と N+1 の同時稼働、drain 経由の無停止切替）を再現する。

## Out of Scope
- 特定クラウドのマネージド フェイルオーバー製品（RDS Multi-AZ / ElastiCache 等）への実装依存。ADR は方式を製品中立に記述し、drill はローカルで再現する。
- マルチリージョン アクティブ／アクティブ（グローバル書き込み分散）。まず単一リージョン・マルチAZ で無停止を必達し、リージョン喪失は [[wi-101-backup-restore-and-disaster-recovery]] の DR（RPO/RTO 付き）で受ける。真のマルチリージョン active/active が必要なら別 WI と ADR を切る。
- データ層のパーティショニング / リードレプリカ / 接続プール本体。→ [[wi-164-data-tier-scalability-partitioning-read-replica-pooling]]
- 容量目標そのものの定義。→ [[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]]

## Plan
- 「単一障害で認証が止まらない」を必達ラインに置き、障害単位（AZ / DB ノード / Valkey ノード / app レプリカ）ごとに冗長化と自動昇格を ADR で確定する。
- 縮退は fail-safe を原則にしつつ、セキュリティ境界（throttle / 認可 / quota）は [[wi-106-distributed-login-throttle-and-shared-state-ha]] と整合して fail-closed を維持する。「可用性のために安全を捨てない」線を明記する。
- 過負荷保護は認証コアを最後まで守るため、重い経路（一覧全件・エクスポート・集計）から先に load shed する優先度設計にする。
- スキーマ / デプロイの無停止化は expand/contract 規約を先に定め、既存 migration 運用（declarative schema）に載せる。
- 検証はローカルで再現可能な failover / chaos / zero-downtime drill を最優先で用意し、「実際に切り替わる」ことを継続確認できる状態にしてから本番手順化する。

## Tasks
- [ ] T001 [ADR] HA / フェイルオーバートポロジ（マルチAZ・自動昇格・LB 連携・リージョン分界）を記録する。
- [ ] T002 [ADR] 縮退マトリクスと load shedding / backpressure / 統一 retry-timeout-breaker 方針を記録する。
- [ ] T003 [SCL] 可用性トポロジ objective、fail-safe/fail-closed 縮退 guarantee、前後方互換 constraint、障害 scenarios を追加し `just scl-render` を通す。
- [ ] T004 [Go/HTTP] 過負荷保護ミドルウェア（同時実行上限・503+Retry-After・経路優先度 load shedding）を追加する。
- [ ] T005 [Go] 依存縮退 state を readiness / metrics に反映する。
- [ ] T006 [Migration] スキーマ移行を expand/contract 2 段に分ける規約を適用する。
- [ ] T007 [Drill] docker compose で app kill / DB 昇格 / Valkey 停止 / スパイク / zero-downtime deploy の drill recipe を追加する。
- [ ] T008 [Verify] `just yaml-check`、`just verify-go`、`just check-ids`、failover drill を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just check-ids`
- failover / chaos / zero-downtime drill 用 `just` recipe
- 手動: docker compose 上で PostgreSQL primary を停止し、standby 昇格後に `/token` `/authorize` が継続することを確認する。
- 手動: Valkey ノードを停止し、共有状態が HA 構成で継続、または throttle が fail-closed に落ちて安全側になることを確認する。
- 手動: app レプリカを 1 台 kill し、drain と LB 経路除外でリクエストが落ちないことを確認する。
- 手動: 負荷スパイクで重い経路が先に 503+Retry-After になり、認証コアが応答し続けることを確認する。
- 手動: N → N+1 のローリングデプロイ中に、新旧同時稼働でリクエストが失敗しないことを確認する。

## Risk Notes
可用性施策はセキュリティ境界（throttle / 認可 / quota）を「可用性のため」に緩めると silent なセキュリティ劣化になる。
縮退マトリクスで fail-safe（認証コア継続）と fail-closed（保証できない機能は拒否）の線を明示し、[[wi-106-distributed-login-throttle-and-shared-state-ha]] の方針を崩さない。
自動フェイルオーバーは「実際に切り替わるか」が最大の不確実性であり、机上設計だけでは信用しない。ローカルで再現可能な drill を先に整備し、継続検証してから本番手順化する。
過負荷保護の閾値は低すぎると正常時に誤って 503 を返すため、容量目標 ([[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]]) と突き合わせて設定する。
