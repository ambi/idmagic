---
status: pending
authors: [tn]
risk: high
created_at: 2026-07-18
depends_on: []
---

# LifecycleWorkflow を新 Bounded Context `IdGovernance` へ切り出す

## Motivation

Identity Lifecycle / Workflow（JML 自動化 = LifecycleWorkflow）は現在 `IdManagement`
context に User / Group / Agent / Attribute / DynamicGroupRule と同居している。これは repo で
2 番目に大きい context（backend ~15,973 行 / 88 files、SCL 4,078 行）で、LifecycleWorkflow
スライス（~20 files、SCL ~166 行 / 38 models / 11 endpoints / 11 events）が per-layer module に
smear されており、1 機能の理解に 4 module + Jobs + Audit + frontend を横断する必要がある。
AI 駆動開発にも人間の認知にも不利な context locality の低さである。

ADR-113 決定1は「当時は新 context を作らず IdManagement に置く。将来 trigger/action の
種類が増え明確に分離すべき規模になれば再検討する」とした。広義 IGA ガバナンス
（wi-213 認証キャンペーン、wi-214 アクセスリクエスト/承認、wi-154 エンタイトルメント/SoD、
wi-152 JIT 特権昇格）が近〜中期ロードマップとして既に WI 化されており、その再検討条件が満たされた。
今のうちに新 Bounded Context `IdGovernance` を strangler seam として切り出し、これら IGA を
育てる受け皿を作る。あわせて IdManagement を「identity principal の record-of-truth」へ
痩せさせ、両 context の locality を最大化する。

## Scope

- **Context 名短縮**: `IdentityManagement`→`IdManagement`、`IdentityGovernance`→`IdGovernance`
  にリポジトリ全体で改名する（spec context 名 + Go パッケージ/ディレクトリ `identitymanagement`→
  `idmanagement` + import + Deps フィールド + sqlc.yaml + ARCHITECTURE module 名/パス + live コア
  ドキュメント）。歴史的記録（done/ の完了 WI、過去 ADR）は作成時点の名前を保持し改名しない。
- `spec/scl.yaml` `context_map`: `IdGovernance` を追加（`depends_on` = IdManagement /
  Application / Jobs、すべて `via: published_language`）。
- `spec/contexts/identity-governance.yaml`（新規）: LifecycleWorkflow の models（~38）/ interfaces
  （11 endpoint、`/api/admin/lifecycle_workflows`）/ events（11）/ `LifecycleWorkflowStatus` state
  machine / objective / scenarios / flows を `identity-management.yaml` から移設。
- `spec/contexts/identity-management.yaml`: LifecycleWorkflow セクションを除去。`publishes` に User
  ライフサイクルイベント（`UserCreated` は既存、`UserAttributesChanged` / `UserStatusChanged` を追加）
  と、governance executor 向けの冪等コマンド surface（group membership / enable-disable /
  required-action）を published interface として追加。`UserLifecycle` state machine は User と共に残す。
- `spec/contexts/application.yaml`: `AssignApplicationDesiredState` / `UnassignApplicationDesiredState`
  （ADR-113 で追加済みの internal interface）の published 化を検討・整理。
- `spec/contexts/jobs.yaml`: `JobKind.lifecycle_workflow_run` を汎用 enum から外し caller 登録へ。
- `spec/contexts/audit.yaml`: 11 個の LifecycleWorkflow* イベントの emitter を IdGovernance へ帰属。
- `ARCHITECTURE.md`: 新 context エントリ + `idgovernance-{domain,ports,usecases,adapters}` module
  + depends_on エッジ + runtime_unit 影響を同期。

## Out of Scope

- 将来 IGA 機能（wi-213 / wi-214 / wi-154 / wi-152）そのものの実装。本 WI は受け皿となる context 境界の
  確立まで。
- SCIM inbound（Scim context）と Application catalog/assignment の統合。両者は正当に独立した境界であり
  移設しない。境界は「Governance = policy/orchestration、record context = state」と ADR で明文化するのみ。
- `DynamicGroupRule` の物理移設。co-location は ADR で判断し、Group 書き込み結合を伴うため本 WI では
  提案の記録に留め、実移設は後続 WI に切り出しうる（Phase 4 で判断）。

## Plan

RA/SCL-first に従い、設計成果物を先に確定し、コード移設は strangler で段階実行する。各 Phase 後に
verify green を維持する。

### 境界を貫く 2 契約（核心）

現状 LifecycleWorkflow は User 集約変更に**トランザクション的に密結合**している（`UserWorkflowCapture`
= User mutation と WorkflowRun enqueue を同一 Tx で束ねる port、ADR-113 決定2）。context 分離では
これを 2 つの published 契約へ分解する。

1. **Read 契約（トリガ）— 既存 outbox 基盤を再利用**: IdManagement が User ライフサイクル
   イベントを発火し、既存の transactional outbox → eventsink → relay
   （`oauth2/adapters/persistence/postgres/outbox.go`、`oauth2/ports/event_sink.go`、`idmagic-relay`、
   `shared/adapters/eventsink/kafka_relay.go`）へ載せる。User mutation と同一 Tx で outbox へ書けば、
   従来 `UserWorkflowCapture` が保証していた原子性（「User 更新は成功したが run が二度と作られない」
   障害窓を閉じる）を outbox が代替する。`UserCreated` は `admin_users.go:135` で発火済み。
   IdGovernance がイベントを購読し WorkflowRun を生成。ADR-113 決定2 はこの cross-context 版へ
   精緻化される。
2. **Write 契約（アクション）— published command surface**: executor の 9 アクション
   （add/remove_group_member, assign/unassign_application, enable/disable_user, set/clear_required_action,
   send_email）は record context を書き換える。IdManagement / Application に冪等コマンド surface を
   published interface として追加し、executor は published interface 経由で呼ぶ（domain 直呼びしない）。
   ADR-113 決定8 の composition-root 配線先例に整合させる。

### 横断リファクタ

- **Jobs kind リーク是正**: `jobs/domain/job.go:44` の `KindLifecycleWorkflowRun` ハードコードと
  `jobs/.../sqlcgen/models.go:222-260` の LifecycleWorkflow* struct 混入を解消。kind は caller 登録へ、
  sqlc モデルは新 context の schema/sqlc へ（ADR-090 context-local persistence）。

### 検討したが採らない案

- 現状維持（Context 内 module 化のみ）: locality は改善するが確定 IGA ロードマップの受け皿にならず二度手間。
- プロビジョニング全統合（SCIM/Application も governance へ）: 過剰結合。policy/state の責務分離で対応。

## Tasks

- [x] T001 [ADR] ADR-117 を起票: IdGovernance 切り出しの決定、read(event)+write(command) 契約、
      strangler 戦略、Jobs kind リーク是正、DynamicGroupRule co-location 判断、ADR-113 との supersede 関係。
      → `decisions/ADR-117-extract-identity-governance-context.md` 作成、ADR-113 に supersede 注記追加。`just check-ids` green。
- [~] T002 [SCL] context 分割は完了。残: 前方 published 契約は Phase 1 (T004) と同時に確定する。
      - [x] `spec/scl.yaml` context_map に IdGovernance（depends_on: Tenancy/IdManagement/Application/Jobs、すべて published_language）追加。
      - [x] `spec/contexts/identity-governance.yaml` 新規作成: LifecycleWorkflow の glossary(5) / models(38) /
            states(2) / interfaces(11) / authorization / objective(1) / scenarios(13) / flow(1) を移設。
            foreign 型は published-language stub（User / UserStatus / RequiredAction / AssignmentVisibility /
            InvalidRequestError / AccessDeniedError）で解決。
      - [x] `identity-management.yaml` から LifecycleWorkflow セクション除去（4078→2903 行、残留参照 0）。
      - [x] `jobs.yaml` の lifecycle_workflow_run シナリオ actor を IdGovernance へ帰属（enum 値は allowlist として維持）。
      - [x] `audit.yaml` の workflow.* 検索属性の emitter prose を IdGovernance へ帰属。
      - [x] `just yaml-check` green、`just scl-render` で OpenAPI/HTML/JSONSchema 再生成（IdGovernance 帰属で反映）。
      - [ ] IM published surface（`UserAttributesChanged` / `UserStatusChanged` events + 冪等コマンド surface）を T004 で追加。
      - [ ] `application.yaml` の `AssignApplicationDesiredState` / `UnassignApplicationDesiredState` published 昇格を T004 で実施。
- [~] T003 [Arch] `ARCHITECTURE.md` に IdGovernance context を登録し `just yaml-check`（architecture
      cross-check）green。context_map は acyclic（IG→{IM,Application,Jobs,Tenancy}、逆依存なし）を検証済み。
      残: `idgovernance-{domain,ports,usecases,adapters}` module / depends_on エッジ / runtime_unit は
      コード移設（T005）で実 path が生成された後に追加する（module path 実在チェックのため先行不可）。
- [ ] T004 [App] Phase 1: `UserAttributesChanged` / `UserStatusChanged` の発火を IM に追加し outbox へ載せる。
      write command surface を published interface として実装。同一 Tx の `UserWorkflowCapture` を
      outbox + event 購読へ置換（原子性は outbox が担保）。
- [ ] T005 [App] Phase 2: `backend/idgovernance/{domain,ports,usecases,adapters}` を新設し
      `lifecycle_workflow*` 20 files を移設。専用 migration + sqlc package + 新 `idgovernance.Module`。
      `idmanagement.Module` から LifecycleWorkflow* フィールドを除去。
- [ ] T006 [App] Phase 3: Jobs kind リーク是正、Audit イベント帰属付け替え、bootstrap
      （`memory.go` / `postgres_valkey.go`）・worker（`cmd/idmagic-worker/worker.go`）・中央 `routes.go` を
      新 context へ配線。
- [ ] T007 [App] Phase 4: `frontend/src/features/admin-lifecycle-workflows` と routes を新 context feature へ。
      DynamicGroupRule co-location を ADR 判断どおり実施 or 後続 WI へ分離。
- [ ] T008 [Verify] `/scl-render` で派生成果物再生成、全 Phase の検証を実施（下記）。

## Verification

- 各 Phase 後: `just yaml-check`（SCL 妥当性）、`ra verify` / executable architecture map で module graph が
  acyclic かつ context 依存が published_language 経由であることを機械検証。
- `just verify-go`（build + test）、`just test-go`。特に Phase 1 は
  `lifecycle_workflow_dispatcher_test.go` / `lifecycle_workflows_test.go` で outbox 経由トリガの原子性
  （User 作成 → run 生成）が保たれることを確認。
- `/scl-render` で OpenAPI に `/api/admin/lifecycle_workflows` の 11 endpoint が新 context 帰属で出ることを確認。
- `just verify-ui` / `just test-ui-e2e` で Phase 4 の管理 UI 導線。
- E2E: `just dev` で管理者が LifecycleWorkflow を enable → User を作成/属性変更 → WorkflowRun が生成・実行され
  group/application 割当が反映されることを実アプリで観測。

## Risk Notes

- **high**: 複数 context を跨ぐ構造変更、DB migration、cross-context の published 契約（event + command）、
  Audit/Jobs/worker/bootstrap の横断配線を伴う。最大のリスクは同一 Tx トリガ捕捉（ADR-113 決定2）の
  置換で、既存 outbox パターン（oauth2）を流用し原子性を維持することで軽減する。
- strangler の各 Phase を独立に verify green で刻み、必要なら T005 以降を個別 WI へ分割可能。
- ADR-113 の決定 2/8 を supersede/精緻化するため、両 ADR の整合を ADR-117 で明記する。
