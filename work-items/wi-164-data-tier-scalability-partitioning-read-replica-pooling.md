---
depends_on: [wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
priority: p3
change_kind: operations
affected_spec:
  - { path: docs/contexts/system/scenarios.md, requirement: REQ-SYSTEM-012 }
  - { path: docs/contexts/audit/scenarios.md, requirement: REQ-AUDIT-004 }
---

# データ層を 10万テナント・1000万ユーザで飽和しないよう分割・リードレプリカ・接続プールで支える

## Motivation
app 層を stateless に水平スケールしても、1000万ユーザ・10万テナント規模では
最初に飽和するのは共有データ層である。具体的には (1) 全テナントのユーザ・監査イベント・トークンが
同一テーブルに積み上がり index / autovacuum / plan が肥大化する、(2) レプリカ数を増やすほど
PostgreSQL への直結コネクションが線形に増え `max_connections` を食い潰す、
(3) 読み取り負荷（discovery / JWKS / introspection / 一覧）が単一 primary に集中する、という 3 点。

**4 点目は形が変わった。** 起票時は「Valkey が単一ノードだと session / code / throttle のメモリと帯域が上限になる」と書いていたが、
[[wi-278-consolidate-ephemeral-state-into-postgresql-remove-valkey]] で揮発性の一時状態はすべて PostgreSQL へ移り、Valkey は廃止された。
別基盤の容量上限という問題は消え、代わりに **同じ PostgreSQL クラスタに高 churn テーブルが 9 つ増えた**という形で戻ってきている。
認可コード・PAR・device code・WebAuthn チャレンジ・リプレイ JTI は `UNLOGGED`、失効リストとログインスロットルは `LOGGED` で、
いずれも書き込みと削除の回転が速い。したがって本 work item が扱うべきなのはクラスタ化ではなく、
**これら高 churn テーブルの dead tuple と autovacuum、そして introspection ごとに走りうる失効リストのホット読み取り**である。

[[wi-108-database-connection-resilience-circuit-breaker]] は障害時の**耐性**、
[[wi-161-large-tenant-performance-foundation]] は大規模**単一**テナントの read path を扱うが、
「テナント総数・総行数がフリート規模で増えたときにデータ層をどうスケールさせるか」は未着手である。
この WI は [[wi-163-fleet-scale-capacity-and-horizontal-scaling-architecture]] が定めた容量目標を、
データ層のトポロジ（分割 / リードレプリカ / 接続プール / 高 churn テーブルの回転）で実際に満たす。

## Scope
- **specification**:
  - `docs/persistence.md` と `docs/capacity.md` へ記録する決定: PostgreSQL の大規模化方針。テナント / 時系列でのパーティショニング境界（特に audit event, token, session, auth-event bucket など append-heavy テーブル）、
    read/write 分離の可否と一貫性境界、接続プール（アプリ側プール vs 外部 pooler）の選定、`docs/persistence.md` の tenant_id retention classes との整合を記録する。
  - `docs/persistence.md` へ、揮発性テーブル群（認可中間状態 / 認可コード / PAR / device code / リプレイ JTI / 失効リスト / WebAuthn チャレンジ / ログインスロットル / SAML リプレイ）の
    高 churn 対策を記録する。時間パーティション + `DROP PARTITION` による GC、`fillfactor` と HOT update、fail-closed 縮退（[[wi-106-distributed-login-throttle-and-shared-state-ha]] と整合）の維持を明記する。
  - `docs/capacity.md` に、read/write 分離時の**読み取り一貫性境界**（認可・quota・throttle は強整合、discovery / 一覧 / dashboard は短時間 stale 許容）と、
    パーティション / レプリカ運用でも tenant isolation が崩れないことを書く。
- **persistence**:
  - `infra/schema/postgres.sql`（宣言的スキーマ）に append-heavy テーブルのパーティション定義を導入し、既存 index / 制約 / タイムスタンプ列ポリシーと整合させる。
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
- アプリケーション水準のシャーディング（複数 DB クラスタへのテナント分散配置）。まず単一クラスタ内の分割・レプリカで容量目標を満たせるか検証し、届かない場合に別 WI と `docs/persistence.md` を切る。
- 外部検索エンジン導入。
- memory persistence adapter の大規模化（単一レプリカ / テスト専用のまま）。

## Plan
- パーティショニングは append-heavy かつ tenant / 時系列で自然に切れるテーブル（audit event, auth-event bucket, token, session）を第一候補にし、
  参照制約とタイムスタンプ列ポリシー、`tenant_id` key policy を壊さない範囲で declarative schema に落とす。
- read/write 分離は「まず抽象だけ入れて全て primary、その後 stale 許容 path を replica へ」の段階移行にし、
  一貫性事故（throttle / quota / 認可を stale で読む）を設計で塞ぐ。
- 接続プールは外部 pooler（例: transaction pooling）でも壊れない実装制約を先に点検し、アプリ挙動を非依存に保つ。
- 高 churn な揮発性テーブルは、時間パーティション + `DROP PARTITION` を第一候補にする。行単位の `DELETE` による GC は dead tuple と autovacuum の負荷そのものを生むため、規模が上がるほど不利になる。
  失効リストのホット読み取りは、共有ストアの選択と独立に「per-request lookup を避ける設計（短命トークン / インスタンス内キャッシュ / push 失効）」で解く。
- **[[wi-282-staging-load-testing-and-capacity-validation]] の実測結果を入力にする。** UNLOGGED / LOGGED の選択と ephemeral sweep の間隔は wi-282 が staging で確定する暫定値なので、
  本 work item の設計はその結果が出てから確定させる。順序を逆にすると、測っていない値を前提にパーティション設計を決めることになる。
- 大規模性能検証は [[wi-161-large-tenant-performance-foundation]] の seed / benchmark 基盤を再利用し、通常 verify と perf smoke を分離する。

## Tasks
- [ ] T001 [Spec] PostgreSQL パーティション / read-write 分離 / 接続プール方針を記録する。
- [ ] T002 [Spec] 高 churn な揮発性テーブルの時間パーティションと GC 方針、fail-closed 縮退の維持を記録する。
- [ ] T003 [Spec] 読み取り一貫性境界と tenant isolation guarantee を追記し、`mise run spec-render` を通す。
- [ ] T004 [Persistence] declarative schema に append-heavy テーブルのパーティションを導入する。
- [ ] T005 [Persistence/Go] read/write ルーティング抽象を追加し、既存 usecase を write=primary 既定で移行する。
- [ ] T006 [Persistence] pooler 経由での動作制約（prepared statement / session 依存）を点検・修正する。
- [ ] T007 [Perf] 10万テナント seed でパーティション pruning と replica 経路を検証する `mise` task を追加する。
- [ ] T008 [Verify] `mise run check`、`mise run verify-go`、`mise run check-ids`、perf smoke を通す。

## Verification
- `mise run check`
- `mise run spec-render`
- `mise run verify-go`
- `mise run check-ids`
- perf smoke 用 `mise` task
- 手動: 10万テナント seed で代表 query の plan にパーティション pruning と `tenant_id` 条件・期待 index が使われ、全テナント scan が出ないことを確認する。
- 手動: リードレプリカ遅延を注入し、強整合 path（throttle / quota / 認可）が primary を読み、stale 許容 path のみ replica を読むことを確認する。
- 手動: 外部 pooler（transaction pooling）経由で契約テストと基本フローが通ることを確認する。

## Risk Notes
パーティション境界と `tenant_id` key policy を誤ると、tenant 混在や pruning 不発という不可逆・高コストな事故になる。
read/write 分離は「stale を読んではいけない path を replica に流す」のが最大の危険で、既定を write=primary にして明示的にだけ replica-eligible にすることで fail-safe にする。
接続プール（transaction pooling）は prepared statement / セッション状態の前提を壊すことがあるため、アプリ側の依存を先に洗い出す。
高 churn テーブルの GC 方式を変えても throttle の fail-closed 縮退を崩さないことを最優先で検証する。
