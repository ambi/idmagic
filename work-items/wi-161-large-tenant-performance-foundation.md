---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# 大規模テナントでも軽快に動く検索・集計・性能保証を整備する

## Motivation
ページネーションとクォータを入れても、画面が必要とする検索、絞り込み、件数表示、dashboard 集計、関連リソース数、
監査イベント検索が都度 OLTP テーブル全体を走査すると、大規模テナントでは軽快に動かない。
特に admin dashboard、role detail の付与人数、application assignment 数、audit event 検索、ユーザー・グループ検索は、
件数が増えるほど初期表示やフィルタ変更の体感速度へ直結する。

この WI は最大規模を想定した read path の設計、index、read model、計測、性能回帰テストをまとめて整備し、
「大量データでも最初の画面が速い」状態を保証する。

## Scope
- **scl**:
  - `System` または各 context の `objectives` に、大規模テナント時の admin / account 主要画面 p95 / p99、DB query budget、最大 page size、dashboard 集計 freshness を追加する。
  - `scenarios` に大量 users / groups / agents / audit events / applications を持つ tenant での検索・集計・画面表示例を追加する。
  - `flows` と `scenarios` に一覧や dashboard が全件取得に依存しないこと、件数は近似または非同期集計を許容する境界を明記する。
- **decision**:
  - 新規 ADR: read model / counter cache / materialized view / search index の採用方針、freshness、再構築、tenant isolation、PII 取り扱いを決める。
- **persistence**:
  - 主要検索条件に合わせた PostgreSQL index、covering index、partial index、text search 方針を追加する。
  - dashboard / role count / assignment count / quota usage などは必要に応じて read model または counter cache に切り出す。
  - query plan の期待値を test または documented evidence として残す。
- **go/usecase**:
  - UI 集計のために list endpoint を全件走査する箇所を summary endpoint または read model query に置き換える。
  - 大規模検索で未制限 wildcard や prefix 無し contains が高コストになる場合、検索 grammar と validation を定義する。
  - slow query / high cardinality filter を観測できる structured log と metrics を追加する。
- **ui**:
  - Dashboard、role detail、admin 一覧の summary 表示を capped list ではなく summary API / read model へ切り替える。
  - 高コスト検索は debounce、明示 submit、最小文字数、loading skeleton、キャンセルを導入する。
- **tests / performance**:
  - seed / fixture / benchmark で大規模 tenant データを作れるようにし、代表 query と画面表示の性能を検証する。
  - CI で重すぎる場合は nightly / opt-in recipe とし、通常 verify では contract と軽量 regression を通す。

## Out of Scope
- 一覧 API の cursor pagination 本体。これは [[wi-159-admin-resource-cursor-pagination]] で扱う。
- テナント resource quota の定義と作成拒否。これは [[wi-160-tenant-resource-quotas]] で扱う。
- 外部検索エンジンの導入を前提にした全文検索基盤。PostgreSQL で足りないことが確認された場合に別 WI と ADR を切る。
- CSV / bulk operation のジョブ化。

## Plan
- まず SCL objective で「大規模テナント」を具体化する。例: users 100k、groups 20k、agents 10k、applications 10k、audit events 10M retained など、ADR で検証用 scale profile を決める。
- 既存 UI の全件取得・list endpoint 集計パターンを棚卸しし、summary が必要な画面とページングで十分な画面を分ける。
- PostgreSQL を主対象に query plan と index を整える。memory persistence は contract 検証に留め、大規模性能保証の主対象にはしない。
- Read model は freshness を明示する。認可や quota enforcement に必要な値は強整合、dashboard 表示は短時間 stale を許容する。
- 性能検証は `just` recipe に載せ、通常 verify と長時間 perf smoke を分ける。

## Tasks
- [ ] T001 [ADR] 大規模テナント scale profile、read model 方針、freshness、検索制約、性能検証方式を記録する。
- [ ] T002 [SCL] performance objectives、large-tenant scenarios、UX の全件取得禁止・summary freshness を追加する。
- [ ] T003 [Render] `just scl-render` で派生物を更新する。
- [ ] T004 [Audit] 既存 UI / API の全件取得、list endpoint 集計、未制限検索を棚卸しして置換対象を確定する。
- [ ] T005 [Persistence] 主要 query の index / read model / counter cache / migration を実装する。
- [ ] T006 [Go] summary endpoint、検索 validation、slow query metrics / structured log を追加する。
- [ ] T007 [UI] dashboard / detail summary / 高コスト検索 UI を summary API と cancellable loading に移行する。
- [ ] T008 [Perf] 大規模 seed、query benchmark、画面 smoke、`just` recipe を追加する。
- [ ] T009 [Verify] `just yaml-check`、`just verify-go`、`just verify-ui`、perf smoke recipe を通す。

## Verification
- `just yaml-check`
- `just scl-render`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- perf smoke 用 `just` recipe
  - reason: 大規模 seed と query / UI 応答時間は通常 verify から分離しても、再現可能な recipe として必要なため。
- 手動: scale profile の tenant で admin dashboard、users、groups、agents、applications、audit events を開き、初期表示と検索が objective 内に収まることを確認する。
- 手動: query plan に tenant_id 条件と期待 index が使われ、全 tenant scan や filesort 相当の高コスト plan が出ていないことを確認する。

## Risk Notes
性能対応は個別の index 追加だけでは再発しやすい。UI が件数表示のために list endpoint を全件走査したり、
検索が未制限 contains になったりすると、ページネーション後も負荷は残る。
Read model は鮮度と整合性の説明がないと誤った運用判断につながるため、強整合が必要な quota / authorization と、
多少 stale でもよい dashboard summary を分ける。性能検証データには PII を入れず、tenant isolation を query plan と test で確認する。
