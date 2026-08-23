---
depends_on: [wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]
status: pending
authors: ["tn"]
risk: critical
created_at: 2026-07-10
priority: p2
change_kind: operations
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-002 }
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-001 }
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

結果として (1) AZ / ノード障害時の app・PostgreSQL の冗長化と自動昇格、
(2) 集中負荷・スパイク時の load shedding / backpressure による過負荷での連鎖崩壊防止、
(3) スキーマ移行・ローリングデプロイをダウンタイム無しで行うための前後方互換の規約、
(4) フェイルオーバーが実際に機能することの検証（chaos / drill）、が未定義のままである。
この WI は [[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]] の参照アーキテクチャ上で
可用性を最大化し、単一障害で認証が止まらない状態を設計・検証で保証する。

**起票時からの前提の変化。** ステートフルな依存は PostgreSQL 1 つだけになった。
[[wi-278-consolidate-ephemeral-state-into-postgresql-remove-valkey]] で Valkey が廃止され、
[[wi-305-remove-external-message-brokers]] で外部メッセージブローカーも撤廃されたので、
「2 つ目のステートフル基盤の HA」という項目そのものが消えている。冗長化すべき障害単位は
AZ / PostgreSQL ノード / app レプリカの 3 つに縮んだ。

**縮退の順序は既に決まっている。** wi-163 の完了で `docs/capacity.md` に Degradation order が入り、
`bulk` → `default` → 管理集計/エクスポート → 動的登録 → 認証コアという落とす順序と、
「テナント境界・認証と認可の検証・再送防止・流量制限・監査の完全性は弱めない」不可侵線が確定した。
同文書は「障害種別ごとの遷移、冗長化方式、過負荷保護の実装、運用手順は高可用性設計で定める」と
明示的に本 work item へ委譲している。したがってここで縮退マトリクスを作り直してはならない。
残っているのは、その順序を**実際に強制する実装**と、**障害種別ごとの遷移とトポロジ**である。

## Scope
- **specification**:
  - `docs/deployment.md` に HA / フェイルオーバートポロジを記録する: app のマルチAZ 配置と anti-affinity、PostgreSQL の primary/standby 自動昇格方式、LB ヘルスチェックと [[wi-98-kubernetes-health-probes-and-graceful-drain]] readiness の連携、マルチリージョン到達目標（まず単一リージョン・マルチAZ を必達、リージョン喪失は [[wi-101-backup-restore-and-disaster-recovery]] の DR で受ける分界）。
  - `docs/deployment.md` に過負荷保護の適用点を記録する: load shedding / concurrency limit / backpressure の適用点と閾値、retry / timeout / circuit breaker（[[wi-108-database-connection-resilience-circuit-breaker]] を横断適用）の統一方針。`docs/capacity.md` の Degradation order をどう強制するかを書く形にし、順序そのものは再定義しない。
  - `docs/deployment.md` に「スキーマ移行とデプロイは前後方互換（N/N+1 が同時稼働可能）」を規約として書く。
  - `docs/capacity.md` の Service level objectives に、許容する同時障害単位（1 AZ 喪失 / 1 PostgreSQL ノード喪失 / N app レプリカ喪失で無停止）と自動フェイルオーバー時の RTO/RPO を Specification target として追加する（[[wi-101-backup-restore-and-disaster-recovery]] の DR runbook の目標とは別物として区別する）。
  - `docs/scenarios.md` に AZ 障害・primary DB 障害・スパイク過負荷・ローリングデプロイ中のリクエスト、の各シナリオを追加する。
- **go/usecase / http**:
  - 過負荷保護ミドルウェア（同時実行上限・キュー上限超過時の即時 503 + Retry-After、優先度の低い経路の load shedding）を追加する。認証コアと重い集計・エクスポートを分けて後者を先に落とす。
  - 依存縮退を判定する degradation state を readiness / metrics（[[wi-112-prometheus-metrics-and-authentication-golden-signals]] 相当）に反映する。
  - スキーマ前後方互換を守るため、破壊的移行を expand/contract 2 段に分ける規約をコード / migration に適用する。
- **tooling / verification**:
  - ローカル docker compose で「app レプリカ 1 台 kill」「PostgreSQL primary 停止 → standby 昇格」「負荷スパイク」を再現する failover / chaos drill recipe を `deploy` 配下に置く。
  - ゼロダウンタイム デプロイ drill（N と N+1 の同時稼働、drain 経由の無停止切替）を再現する。

## Out of Scope
- 特定クラウドのマネージド フェイルオーバー製品（RDS Multi-AZ / ElastiCache 等）への実装依存。`docs/deployment.md` は方式を製品中立に記述し、drill はローカルで再現する。
- マルチリージョン アクティブ／アクティブ（グローバル書き込み分散）。まず単一リージョン・マルチAZ で無停止を必達し、リージョン喪失は [[wi-101-backup-restore-and-disaster-recovery]] の DR（RPO/RTO 付き）で受ける。真のマルチリージョン active/active が必要なら別 WI と `docs/deployment.md` を切る。
- データ層のパーティショニング / リードレプリカ / 接続プール本体。→ [[wi-164-data-tier-scalability-partitioning-read-replica-pooling]]
- 容量目標そのものの定義。→ [[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]]

## Plan
- 「単一障害で認証が止まらない」を必達ラインに置き、障害単位（AZ / PostgreSQL ノード / app レプリカ）ごとに冗長化と自動昇格を `docs/deployment.md` で確定する。
- 縮退は fail-safe を原則にしつつ、セキュリティ境界（throttle / 認可 / quota）は [[wi-106-distributed-login-throttle-and-shared-state-ha]] と整合して fail-closed を維持する。「可用性のために安全を捨てない」線は `docs/capacity.md` の Degradation order に既にあるので、実装がそれを守っていることを検証で示す。
- 過負荷保護は認証コアを最後まで守るため、`docs/capacity.md` の Degradation order をそのままミドルウェアの優先度設計に落とす。
- スキーマ / デプロイの無停止化は expand/contract 規約を先に定め、既存 migration 運用（declarative schema）に載せる。
- 検証はローカルで再現可能な failover / chaos / zero-downtime drill を最優先で用意し、「実際に切り替わる」ことを継続確認できる状態にしてから本番手順化する。

## Tasks
- [ ] T001 [Spec] `docs/deployment.md` に HA / フェイルオーバートポロジ（マルチAZ・自動昇格・LB 連携・リージョン分界）を記録する。
- [ ] T002 [Spec] `docs/deployment.md` に load shedding / backpressure / 統一 retry-timeout-breaker の適用点と、`docs/capacity.md` の Degradation order をどう強制するかを記録する。
- [ ] T003 [Spec] `docs/capacity.md` に可用性トポロジ目標、`docs/deployment.md` に前後方互換の規約、`docs/scenarios.md` に障害シナリオを追加し `mise run check-spec` を通す。
- [ ] T004 [Go/HTTP] 過負荷保護ミドルウェア（同時実行上限・503+Retry-After・経路優先度 load shedding）を追加する。
- [ ] T005 [Go] 依存縮退 state を readiness / metrics に反映する。
- [ ] T006 [Migration] スキーマ移行を expand/contract 2 段に分ける規約を適用する。
- [ ] T007 [Drill] docker compose で app kill / DB 昇格 / スパイク / zero-downtime deploy の drill recipe を追加する。
- [ ] T008 [Verify] `mise run check`、`mise run verify-go`、`mise run check-ids`、failover drill を通す。

## Verification
- `mise run check`
- `mise run spec-render`
- `mise run verify-go`
- `mise run check-ids`
- failover / chaos / zero-downtime drill 用 `mise` task
- 手動: docker compose 上で PostgreSQL primary を停止し、standby 昇格後に `/token` `/authorize` が継続することを確認する。
- 手動: PostgreSQL へ到達できない状態を作り、ログインスロットルが fail-closed に落ちて安全側になることを確認する。
- 手動: app レプリカを 1 台 kill し、drain と LB 経路除外でリクエストが落ちないことを確認する。
- 手動: 負荷スパイクで重い経路が先に 503+Retry-After になり、認証コアが応答し続けることを確認する。
- 手動: N → N+1 のローリングデプロイ中に、新旧同時稼働でリクエストが失敗しないことを確認する。

## Risk Notes
可用性施策はセキュリティ境界（throttle / 認可 / quota）を「可用性のため」に緩めると silent なセキュリティ劣化になる。
縮退マトリクスで fail-safe（認証コア継続）と fail-closed（保証できない機能は拒否）の線を明示し、[[wi-106-distributed-login-throttle-and-shared-state-ha]] の方針を崩さない。
自動フェイルオーバーは「実際に切り替わるか」が最大の不確実性であり、机上設計だけでは信用しない。ローカルで再現可能な drill を先に整備し、継続検証してから本番手順化する。
過負荷保護の閾値は低すぎると正常時に誤って 503 を返すため、容量目標 ([[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]]) と突き合わせて設定する。
