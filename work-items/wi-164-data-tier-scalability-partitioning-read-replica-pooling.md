---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# データ層を 10万テナント・1000万ユーザで飽和しないよう分割・リードレプリカ・接続プール・Valkey クラスタ化する

## Motivation
app 層を stateless に水平スケールしても、1000万ユーザ・10万テナント規模では
最初に飽和するのは共有データ層である。具体的には (1) 全テナントのユーザ・監査イベント・トークンが
同一テーブルに積み上がり index / autovacuum / plan が肥大化する、(2) レプリカ数を増やすほど
PostgreSQL への直結コネクションが線形に増え `max_connections` を食い潰す、
(3) 読み取り負荷（discovery / JWKS / introspection / 一覧）が単一 primary に集中する、
(4) Valkey が単一ノードだと session / code / throttle 用のメモリと帯域が上限になる、という 4 点。

[[wi-108-database-connection-resilience-circuit-breaker]] は障害時の**耐性**、
[[wi-161-large-tenant-performance-foundation]] は大規模**単一**テナントの read path を扱うが、
「テナント総数・総行数がフリート規模で増えたときにデータ層をどうスケールさせるか」は未着手である。
この WI は [[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]] が定めた容量目標を、
データ層のトポロジ（分割 / リードレプリカ / 接続プール / Valkey クラスタ）で実際に満たす。

## Scope
- **decision**:
  - 新規 ADR: PostgreSQL の大規模化方針。テナント / 時系列でのパーティショニング境界（特に audit event, token, session, auth-event bucket など append-heavy テーブル）、
    read/write 分離の可否と一貫性境界、接続プール（アプリ側プール vs 外部 pooler）の選定、`tenant_id` key policy ([[tenant-id-key-policy-adr083]]) との整合を記録する。
  - 新規 or 既存 ADR 追補: Valkey のクラスタ / シャーディング / レプリケーション方針。key 空間（session / authz code / PAR / device code / throttle / denylist）のシャード適合性と、fail-closed 縮退（[[wi-106-distributed-login-throttle-and-shared-state-ha]] と整合）を維持することを明記する。
- **scl**:
  - `System` の該当 objective / constraint に、read/write 分離時の**読み取り一貫性境界**（認可・quota・throttle は強整合、discovery / 一覧 / dashboard は短時間 stale 許容）を明文化する。
  - パーティション / レプリカ運用でも tenant isolation が崩れない guarantee を追記する。
- **persistence**:
  - `deploy/schema/postgres.sql`（宣言的スキーマ）に append-heavy テーブルのパーティション定義を導入し、既存 index / 制約 / タイムスタンプ列ポリシーと整合させる。
  - 読み取りをリードレプリカへ振り分けられるよう、repository / 接続取得層に read/write のルーティング境界を用意する（強整合が必要な path は primary 固定）。
  - 接続プール前提（外部 pooler 経由でも壊れないよう prepared statement / session 依存を点検）を満たす。
- **go/usecase**:
  - 接続取得を read-intent / write-intent で区別できる薄い抽象を追加し、既存 usecase を安全側（write=primary）既定で移行する。
  - stale 読み取りを許容する path だけを明示的に replica-eligible にする。
- **tests / performance**:
  - 10万テナント規模の seed（多数小テナント + 少数巨大テナント）で、パーティション pruning が効くこと、`tenant_id` 条件で全テナント scan にならないことを query plan で検証する。
  - pooler 経由・レプリカ遅延ありの条件で契約テストが通ることを確認する。

## Out of Scope
- app 層 stateless 化そのものと容量目標の定義。→ [[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]]
- マルチAZ / 自動フェイルオーバー / DR。→ [[wi-165-high-availability-and-failover-resilience-topology]] と [[wi-101-backup-restore-and-disaster-recovery]]
- アプリケーション水準のシャーディング（複数 DB クラスタへのテナント分散配置）。まず単一クラスタ内の分割・レプリカで容量目標を満たせるか検証し、届かない場合に別 WI と ADR を切る。
- 外部検索エンジン導入。
- memory persistence adapter の大規模化（単一レプリカ / テスト専用のまま）。

## Plan
- パーティショニングは append-heavy かつ tenant / 時系列で自然に切れるテーブル（audit event, auth-event bucket, token, session）を第一候補にし、
  参照制約とタイムスタンプ列ポリシー、`tenant_id` key policy を壊さない範囲で declarative schema に落とす。
- read/write 分離は「まず抽象だけ入れて全て primary、その後 stale 許容 path を replica へ」の段階移行にし、
  一貫性事故（throttle / quota / 認可を stale で読む）を設計で塞ぐ。
- 接続プールは外部 pooler（例: transaction pooling）でも壊れない実装制約を先に点検し、アプリ挙動を非依存に保つ。
- Valkey クラスタ化は key 空間のシャード適合性を確認し、fail-closed 縮退方針を維持する。
- 大規模性能検証は [[wi-161-large-tenant-performance-foundation]] の seed / benchmark 基盤を再利用し、通常 verify と perf smoke を分離する。

## Tasks
- [ ] T001 [ADR] PostgreSQL パーティション / read-write 分離 / 接続プール方針を記録する。
- [ ] T002 [ADR] Valkey クラスタ / シャーディング / 縮退方針を記録する。
- [ ] T003 [SCL] 読み取り一貫性境界と tenant isolation guarantee を追記し、`just scl-render` を通す。
- [ ] T004 [Persistence] declarative schema に append-heavy テーブルのパーティションを導入する。
- [ ] T005 [Persistence/Go] read/write ルーティング抽象を追加し、既存 usecase を write=primary 既定で移行する。
- [ ] T006 [Persistence] pooler 経由での動作制約（prepared statement / session 依存）を点検・修正する。
- [ ] T007 [Perf] 10万テナント seed でパーティション pruning と replica 経路を検証する `just` recipe を追加する。
- [ ] T008 [Verify] `just yaml-check`、`just verify-go`、`just check-ids`、perf smoke を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just check-ids`
- perf smoke 用 `just` recipe
- 手動: 10万テナント seed で代表 query の plan にパーティション pruning と `tenant_id` 条件・期待 index が使われ、全テナント scan が出ないことを確認する。
- 手動: リードレプリカ遅延を注入し、強整合 path（throttle / quota / 認可）が primary を読み、stale 許容 path のみ replica を読むことを確認する。
- 手動: 外部 pooler（transaction pooling）経由で契約テストと基本フローが通ることを確認する。

## Risk Notes
パーティション境界と `tenant_id` key policy を誤ると、tenant 混在や pruning 不発という不可逆・高コストな事故になる。
read/write 分離は「stale を読んではいけない path を replica に流す」のが最大の危険で、既定を write=primary にして明示的にだけ replica-eligible にすることで fail-safe にする。
接続プール（transaction pooling）は prepared statement / セッション状態の前提を壊すことがあるため、アプリ側の依存を先に洗い出す。
Valkey クラスタ化でも throttle の fail-closed 縮退を崩さないことを最優先で検証する。
