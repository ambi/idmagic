---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# 1000万ユーザ・10万テナント規模のフリート容量目標と水平スケール参照アーキテクチャを定義する

## Motivation
既存の非機能目標はエンドポイント単位の p99 レイテンシ・5xx 比率・可用性 (99.9% / 99.95%) と
`/token` 等のピーク throughput までは定義しているが、**フリート全体がどの規模まで耐える設計なのか**
——同時アクティブテナント数、総ユーザ数、総オブジェクト数 (client / application / group / session / token /
audit event)、ストレージ成長、集中ログイン時の総 rps——という「スケールの上限と根拠」が SCL に存在しない。
数字が無いままだと、レプリカを増やしても実際に 1000万ユーザ・10万テナントで詰まるのは
共有状態ストア、DB 接続、コネクションプール、鍵解決キャッシュのいずれかであり、
どこがボトルネックかを事前に語れない。

この WI は「idmagic を 1000万ユーザ・10万テナントで運用する」という **capacity target を SCL objective として明文化**し、
それを満たす**水平スケール参照アーキテクチャ**（stateless app レプリカ、共有状態ストア、データ層のトポロジと責務分界）を
ADR で確定する。実際のデータ層改修は [[wi-164-data-tier-scalability-partitioning-read-replica-pooling]]、
可用性・フェイルオーバートポロジは [[wi-165-high-availability-and-failover-resilience-topology]] が受け持つ。
本 WI はその 2 つが参照する**土台（目標値と全体設計）**である。

## Scope
- **scl**:
  - `System` context の `objectives` に fleet-scale capacity 目標を追加する。例:
    - 総テナント数 >= 100k、総ユーザ数 >= 10M、テナントあたり中央値 / p99 ユーザ数分布。
    - 同時アクティブ login session、集中時間帯の集約 `/token` `/authorize` `/introspect` rps 上限（既存 throughput SLO をフリート合算で裏付ける）。
    - オブジェクト総数の想定上限（application / oauth2_client / group / audit event retention）とストレージ成長率。
  - `constraints` / `guarantees` に「app 層は stateless で水平スケールする」「ephemeral 状態は共有ストアに載せる（既存 `system.yaml` の shared-store 目標を fleet 規模へ引き上げ）」を明文化・更新する。
  - `scenarios` に「10万テナント・1000万ユーザ規模のクラスタでの集中ログイン」「大量テナントの並行 discovery / JWKS 取得」を追加する。
- **decision**:
  - 新規 ADR: 水平スケール参照アーキテクチャ。app レプリカの stateless 性、共有状態ストア (Valkey) のトポロジ選定基準、データ層 (PostgreSQL) の分界、鍵・JWKS・discovery のキャッシュ戦略、容量計画の算出根拠（1 レプリカあたり処理能力 × 必要レプリカ数）を記録する。データ層と可用性の詳細設計は後続 WI へ委譲する分界も明記する。
- **documentation**:
  - `ARCHITECTURE.md` またはデプロイ参照ドキュメントに、フリート規模のリファレンストポロジ図（LB → N app レプリカ → Valkey → PostgreSQL）と容量計画の前提を追記する。

## Out of Scope
- PostgreSQL のパーティショニング / リードレプリカ / 接続プール実装。→ [[wi-164-data-tier-scalability-partitioning-read-replica-pooling]]
- マルチAZ / マルチリージョン / 自動フェイルオーバー / ゼロダウンタイム移行の実装。→ [[wi-165-high-availability-and-failover-resilience-topology]]
- 特定クラウドのマネージド製品前提の Terraform / Helm chart 実装。参照アーキテクチャは製品中立に記述する。
- 大規模**単一**テナントの検索・集計 read path 最適化。→ [[wi-161-large-tenant-performance-foundation]]
- アプリケーションロジック / HTTP API の挙動変更。

## Plan
- まず capacity target の数字を確定する。テナント規模分布（多数の小テナント + 少数の巨大テナント）を前提に、
  ユーザ・オブジェクト・session・token を「テナント数 × テナントあたり分布」で積み上げ、SLO の throughput と突き合わせる。
- 参照アーキテクチャは既存の到達点（[[wi-98-kubernetes-health-probes-and-graceful-drain]] のプローブ、
  [[wi-106-distributed-login-throttle-and-shared-state-ha]] の共有状態、[[wi-108-database-connection-resilience-circuit-breaker]] の接続耐性）を
  前提に、「1000万/10万で最初に飽和する層」を特定して後続 WI のスコープ根拠にする。
- 容量計画は「1 app レプリカあたりの実測 or 目標処理能力」を仮置きし、必要レプリカ数・DB 接続数・Valkey メモリを算出する。
  実測は perf recipe（[[wi-161-large-tenant-performance-foundation]] の seed / benchmark 基盤）を流用できるか検討する。
- この WI 自体はコード変更を伴わない設計・目標定義に留め、実装は後続へ渡す。

## Tasks
- [ ] T001 [ADR] 水平スケール参照アーキテクチャと容量計画の算出根拠を記録する。
- [ ] T002 [SCL] `System.objectives` に fleet-scale capacity 目標、stateless / 共有ストア constraints、大規模クラスタ scenarios を追加する。
- [ ] T003 [Render] `just scl-render` で派生物を更新する。
- [ ] T004 [Doc] `ARCHITECTURE.md` に参照トポロジと容量前提を追記する。
- [ ] T005 [Verify] `just yaml-check`、`just check-ids` を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just check-ids`
- 手動: capacity target の数字が既存の throughput / availability SLO と矛盾しないこと、
  後続 2 WI が参照する分界（データ層 / 可用性）が ADR で一意に読めることをレビューで確認する。

## Risk Notes
容量目標は「守れない過大な数字」を置くと SLO 全体の信頼性を損ない、「過小」だと設計が 1000万/10万に届かない。
根拠（積み上げ計算と実測の別）を ADR に明記し、実測で更新可能にしておく。
参照アーキテクチャは製品中立に書き、特定クラウドへのロックインを設計段階で作り込まない。
