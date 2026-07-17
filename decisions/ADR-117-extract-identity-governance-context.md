---
status: accepted
authors: [tn]
created_at: 2026-07-18
---

# ADR-117: LifecycleWorkflow を新 bounded context IdGovernance へ切り出す

## コンテキスト

ADR-113 決定1 は「当時は新 context を作らず `IdManagement` に置く。将来 trigger/action の
種類が増え、IdManagement から明確に分離すべき規模になった場合に再検討する」とした。その
再検討条件は現時点で満たされている:

- LifecycleWorkflow スライスは backend ~20 files / SCL ~166 行（38 models / 11 endpoints / 11
  events / 2 state machine / 13 scenarios / 1 flow）に成長し、`IdManagement`
  （backend ~15,973 行 / 88 files、SCL 4,078 行）に per-layer で smear している。1 機能の理解に
  domain / usecase / adapter / Jobs / Audit / frontend を横断する必要があり、context locality が低い。
- 広義 IGA ガバナンス（wi-213 認証キャンペーン、wi-214 アクセスリクエスト/承認、wi-154
  エンタイトルメント/SoD、wi-152 JIT 特権昇格）が近〜中期ロードマップとして既に WI 化されており、
  これらを育てる受け皿が要る。

決めるべきことは 4 つ: (1) LifecycleWorkflow を切り出す新 context を作るか、(2) ADR-113 決定2 の
「同一 transaction による trigger capture」（今は同一 context 内なので成立）を context 分離後も
成立させる整合性設計、(3) executor の write action が record context（IdManagement /
Application）を書き換える経路を context DAG を壊さずに確保する方法、(4) Jobs / Audit / DynamicGroupRule
に残る結合の扱い。

## 決定

### 1. 新 bounded context `IdGovernance` を切り出す（ADR-113 決定1 を supersede）

LifecycleWorkflow スライス（models / interfaces / states / objective / scenarios / flow）を
`spec/contexts/identity-governance.yaml` へ移設し、`IdManagement` を「identity principal の
record-of-truth」へ痩せさせる。`IdGovernance` は policy/orchestration を、record context
（IdManagement / Application）は state を所有する、という責務分割を境界規範とする。
`spec/scl.yaml` の `context_map` に `IdGovernance` を追加し、`depends_on` は
`IdManagement` / `Application` / `Jobs` の 3 つ、すべて `via: published_language`。

### 2. 境界を貫く 2 契約（ADR-113 決定2 を cross-context 版へ精緻化）

ADR-113 決定2 の「User mutation と WorkflowRun 作成を同一 IdManagement transaction で
commit する」原子性保証は、context 分離により 1 つの transaction では表せなくなる。これを 2 つの
published 契約へ分解する:

- **Read 契約（トリガ）— 既存 transactional outbox を再利用**: `IdManagement` は User
  ライフサイクルイベント（`UserCreated`〈既存〉に加え `UserAttributesChanged` /
  `UserStatusChanged` を新設）を、User mutation と**同一 Tx** で transactional outbox
  （`oauth2/adapters/persistence/postgres/outbox.go`、`oauth2/ports/event_sink.go`、
  `shared/adapters/eventsink/kafka_relay.go`、`idmagic-relay`）へ書く。`IdGovernance` が
  outbox → relay 経由でこれを購読し `WorkflowRun` を生成する。これにより「User 更新は成功したが
  run が二度と作られない」障害窓を、outbox の同一 Tx 書き込みが ADR-113 決定2 の共有 transaction に
  代わって閉じる。at-least-once 配送のため、ADR-113 決定4 の
  `(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)` 重複排除は
  cross-context でも維持され、`source_occurrence_id` はイベントの occurrence ID にマップする。

- **Write 契約（アクション）— published command surface**: executor の 9 アクション
  （add/remove_group_member、assign/unassign_application、enable/disable_user、
  set/clear_required_action、send_email）は record context を書き換える。`IdManagement` と
  `Application` に**冪等コマンド surface を published interface** として追加し、executor は
  published interface 経由で呼ぶ（record context の domain を直呼びしない）。ADR-113 決定8 の
  composition-root 配線先例（IdManagement が Application usecase adapter を action executor
  port へ注入する ports-and-adapters）を、`IdGovernance → {IdManagement, Application}`
  の cross-context published 依存へ引き上げる。冪等性（ADR-113 決定5 の desired-state action +
  step checkpoint）は record context 側の command surface でも維持する。

### 3. context DAG 制約（ADR-113 決定8 の再解釈）

ADR-113 決定8 は `IdManagement.depends_on.Application` を追加すると
`Application.depends_on.IdManagement`（UserRef/GroupRef）と cycle になるため、
`AssignApplicationDesiredState` 等を Application 所有の internal interface に留め `context_map` に
エッジを足さない、とした。context を分離した本 ADR では `IdGovernance` が新しい依存元となり、
`IdGovernance.depends_on.{IdManagement, Application, Jobs}` を追加しても
`Application`/`IdManagement` は `IdGovernance` に依存し返さないため **cycle は生じない**。
よって command surface を Application / IdManagement の **published interface** に昇格でき、
`just yaml-check` の context-map cycle 検出（`tools/yaml-check/src/context-map.ts`）と
executable architecture map（ADR-116）の acyclic 検証を通す。

### 4. strangler で段階実行する

設計成果物（本 ADR + SCL + ARCHITECTURE.md）を先に確定し、コード移設は Phase 1〜4 で刻む
（各 Phase 後に verify green）。Phase 1: IM に User events + outbox 発火と record command surface を
足し、同一 Tx capture を outbox+購読へ置換。Phase 2: `backend/idgovernance/{domain,ports,
usecases,adapters}` を新設し `lifecycle_workflow*` を移設（専用 migration + sqlc package）。
Phase 3: Jobs kind リーク是正、Audit emitter 帰属、bootstrap/worker/routes 配線。Phase 4:
frontend feature 移設と DynamicGroupRule co-location 判断。

### 5. Jobs kind リーク是正

`jobs/domain/job.go` の `KindLifecycleWorkflowRun` ハードコードと
`jobs/.../sqlcgen/models.go` の LifecycleWorkflow* struct 混入は ADR-090（context-local
persistence）と Jobs の「業務ロジックは caller に残す」設計に反する。`JobKind` の
`lifecycle_workflow_run` を汎用 enum から外し、caller（IdGovernance）登録の kind とする。
sqlc モデルは新 context の schema/sqlc package へ移す。

### 6. DynamicGroupRule は本 WI では物理移設しない

`DynamicGroupRule`（ADR-111）は Group membership への書き込み結合を持つため、`IdGovernance`
への co-location は魅力的だが Group aggregate との書き込み結合をどう表すかの設計を要する。本 WI では
提案の記録に留め、実移設は後続 WI に切り出しうる（Phase 4 で判断）。

## 却下した代替案

- **現状維持（context 内 module 化のみ）**: locality は改善するが、確定済み IGA ロードマップ
  （wi-213/214/154/152）の受け皿にならず、それらを足すたびに IdManagement を再度肥大させる二度手間。
- **プロビジョニング全統合（SCIM inbound / Application catalog も governance へ）**: 過剰結合。
  SCIM（Scim context）と Application は正当に独立した境界であり、policy/state の責務分離で対応する。
- **command surface を Application/IM 所有の internal interface に留める（ADR-113 決定8 のまま）**:
  context を分離した以上、cross-context の write は published contract として明示すべきで、
  internal のままだと呼び出し配線が SCL 上に現れず executable architecture map で検証できない。
- **`UserWorkflowCapture` の共有 transaction を cross-context RPC 同期で温存**: context 間に同期的
  write coupling を残し、分離の意味を失う。outbox（既存 oauth2 基盤）で原子性は保てる。

## 影響

- `spec/scl.yaml` `context_map`: `IdGovernance` エントリを新設（`depends_on`
  = IdManagement / Application / Jobs、すべて `via: published_language`）。`publishes` は
  LifecycleWorkflow の外部参照点（当面なし〜`LifecycleWorkflowRef` 検討）。
- `spec/contexts/identity-governance.yaml`（新規）: LifecycleWorkflow の models（38）/ interfaces
  （11、`/api/admin/lifecycle_workflows`）/ events（11）/ `WorkflowDefinitionLifecycle` /
  `WorkflowRunLifecycle` state machine / `LifecycleWorkflowTriggerHandoffLatency` objective /
  scenarios（13）/ `AdminLifecycleWorkflows` flow。
- `spec/contexts/identity-management.yaml`: LifecycleWorkflow セクション除去。User ライフサイクル
  イベント `UserAttributesChanged` / `UserStatusChanged`（新設、`UserCreated` は既存）と、governance
  executor 向け冪等コマンド surface（group membership / enable-disable / required-action）を
  published interface として追加。`UserLifecycle` state machine は User と共に残す。
- `spec/contexts/application.yaml`: `AssignApplicationDesiredState` /
  `UnassignApplicationDesiredState`（ADR-113 で internal 追加済み）を published interface へ昇格。
- `spec/contexts/jobs.yaml`: `JobKind` から `lifecycle_workflow_run` を外し caller 登録へ。
- `spec/contexts/audit.yaml`: 11 個の LifecycleWorkflow* イベントの emitter 帰属を IdGovernance へ。
- `ARCHITECTURE.md`: `IdGovernance` context + `idgovernance-{domain,ports,usecases,adapters}`
  module + depends_on エッジ + runtime_unit 影響を同期。
- backend: `backend/idgovernance/` 新設、`lifecycle_workflow*` ~20 files 移設、専用 migration +
  sqlc package、`idgovernance.Module`。`idmanagement.Module` から LifecycleWorkflow*
  フィールド除去。bootstrap（`memory.go` / `postgres_valkey.go`）・worker（`cmd/idmagic-worker`）・
  中央 `routes.go` を新 context へ配線。
- frontend: `frontend/src/features/admin-lifecycle-workflows` と routes を新 context feature へ。
- ADR-113: 決定1 は本 ADR 決定1 で supersede、決定2 は本 ADR 決定2 で cross-context 版へ精緻化、
  決定8 は本 ADR 決定3 で再解釈。決定 3/4/5/6/7（revision 固定・重複排除・部分失敗・loop 抑制・保持）は
  そのまま継承する。
