---
status: accepted
authors: [tn]
created_at: 2026-07-16
---

# ADR-113: Identity lifecycle workflow の所有・実行・整合性モデル

> **注記 (ADR-117 による更新)**: 決定1（新 context を作らず IdentityManagement に置く）は
> ADR-117 決定1 で supersede され、LifecycleWorkflow は新 bounded context `IdentityGovernance`
> へ切り出された。決定2（同一 transaction による trigger capture）は ADR-117 決定2 で
> transactional outbox を用いた cross-context 契約へ精緻化。決定8（Application 割当への internal
> interface 経路）は ADR-117 決定3 で published interface へ再解釈。決定 3/4/5/6/7 は継承。

## コンテキスト

wi-153 は、User の作成・属性変更・status 遷移を trigger として、テナント管理者が定義した
構造化 action (Group 割当、Application 割当、required action、User 無効化、通知) を
durable job として実行する仕組みを要求する。関連する既存機構は 3 つある:

- Jobs (wi-126) は tenant 境界を保つ汎用 at-least-once durable queue を提供するが、業務ロジックは
  呼び出し元 context の usecase に残す設計 (`EnqueueJob` は internal interface)。
- DynamicGroupRule (ADR-111) は単一目的 (Group membership) の CEL rule + 全件再評価 job で、
  複数 trigger 種別・複数 action 種別を持つ汎用性はない。
- Application は割当 (`ApplicationAssignment`) を所有し、`IdentityManagement` の `UserRef` /
  `GroupRef` を published language として消費する (`spec/scl.yaml` の `Application.depends_on.IdentityManagement`)。

決めるべきことは 3 つ: (1) LifecycleWorkflow をどの bounded context が所有するか、
(2) trigger 発火から実行完了までの at-least-once 環境での整合性保証をどう設計するか、
(3) IdentityManagement が Application の割当を変更する経路を、既存の
`Application → IdentityManagement` 依存と衝突させずにどう確保するか。

## 決定

1. **所有 context**: 新しい bounded context は作らず `IdentityManagement` に置く。
   trigger の発火源 (User mutation) と大半の action (Group membership、required action、
   User status) が同一 context 内にあり、独立した context に分離すると trigger 評価のたびに
   User 属性・status を context 跨ぎで取得する必要が生じ、かえって整合性が弱くなる。

2. **transactional trigger capture**: User mutation use case が before/after 差分と
   source occurrence ID を計算し、User の保存と、該当する enabled workflow revision 分の
   `WorkflowRun`/`WorkflowStep` (status=queued, job_id=NULL) 作成を **同一 IdentityManagement
   transaction** で commit する。dispatch は commit 後の別ステップとし、API process の即時
   `EnqueueJob` 呼び出しに加えて worker process の periodic dispatcher が
   `job_id IS NULL AND status = queued` の run を再走査して回収する。これにより
   「User 更新は成功したが run が二度と作られない」障害窓を、実装の作り込みではなく
   transaction 境界そのもので閉じる。

3. **revision 固定と楽観ロック**: `LifecycleWorkflow` は `current_revision` (直近作成) と
   `enabled_revision` (trigger 評価対象) を分離して持つ。意味の変わる update は
   `current_revision` を増やすだけで `enabled_revision`/`status` は変えず、管理者が明示的に
   `EnableLifecycleWorkflow` を呼ぶまで旧 revision の評価が続く。`UpdateLifecycleWorkflow` は
   `expected_revision` が現在の `current_revision` と一致しない場合を
   `WorkflowRevisionConflictError` で拒否する (lost update 防止)。`WorkflowRun` は作成時の
   revision と展開済み action list を固定するため、実行中に definition を変更しても
   進行中 run の意味は変わらない。

4. **重複排除**: `WorkflowRun` の一意制約
   `(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)` で
   同一 trigger occurrence の再配送を一つの run に収束させる。Jobs へは
   `dedup_key=lifecycle-workflow-run:{run_id}` で enqueue し、Jobs 自体の at-least-once 再配送
   による重複 enqueue を防ぐ。`send_email` action は `(run_id, step_index)` を delivery key として
   `LifecycleNotificationDelivery` に記録し、retry や at-least-once 実行下でも重複送信しない。

5. **部分失敗と非補償**: 1 attempt では未完了 step を定義順にすべて試し、途中の failed step で
   後続 action を止めない (アクセス剥奪が無関係な通知失敗で止まらないため)。attempt 上限到達時、
   全 step 成功なら `succeeded`、混在なら `partially_failed`、成功が皆無なら `failed` に終端する
   (no-op は成功として扱う)。context 跨ぎの補償 transaction は実装しない。desired-state な action
   設計 (「存在させる/存在させない」を宣言的に指定する) と step 単位の checkpoint により、
   retry は `changed`/`no_op` 済み step を飛ばし `failed` step だけを再実行することで収束させる。

6. **loop suppression**: `WorkflowRunTriggerSnapshot` は action 実行が発生させた User mutation の
   origin run/step metadata を持ち、trigger 評価は origin metadata を持つ mutation を対象外にする。
   これにより action → 新しい trigger → action の無限連鎖を、実行時のガードではなく trigger
   evaluator の入力自体から構造的に排除する。

7. **保持期間**: `WorkflowRun`/`WorkflowStep`/`LifecycleNotificationDelivery` は Jobs の
   `Job` (ADR-100, delete_after 30日) に合わせて 30 日保持する。`LifecycleWorkflow` と
   `LifecycleWorkflowRevision` は archive 後も、参照中の run が retention 期間内にある間は
   削除しない。保持期間の cleanup は tenant-safe な batch とし、`objectives` の SLO とは独立に
   ADR で運用値として固定する (SCL `objectives` は比率ベースの SLO 表現に限られ、保持期間の
   ような duration 値はモデル記述と ADR に置く)。

8. **Application 割当への経路と context 依存**: `spec/scl.yaml` の `context_map` は既に
   `Application.depends_on.IdentityManagement` を持つ (割当の subject が User/Group のため)。
   ここで `IdentityManagement.depends_on.Application` を追加すると `context_map` の
   `depends_on` は DAG であることを要求する `just yaml-check` の cycle 検出に反する
   (`tools/yaml-check/src/context-map.ts`)。そのため `AssignApplicationDesiredState` /
   `UnassignApplicationDesiredState` は **Application 側が所有する internal interface**
   として `spec/contexts/application.yaml` に追加するが、`context_map` には新しい
   `depends_on` エッジを追加しない。実行時の呼び出しは、IdentityManagement の
   composition root (`backend/cmd/idmagic-api` 相当) が Application の usecase を実装する
   adapter を IdentityManagement 側の action executor port へ注入する、Go レベルの
   ports-and-adapters 配線として解決する。SCL の `context_map` は「型/語彙としての依存」を
   表す DAG であり、composition root が握る具体的な呼び出し配線の全てを 1:1 で反映する
   ものではないことを明示する (既存の `dynamic_group_reconcile` → Jobs 連携も同様に、
   `EnqueueJob` 呼び出し自体は `identity-management.yaml` の interface としては現れず
   `depends_on: Jobs` にとどまる先例に合わせる)。

## 却下した代替案

- **新しい bounded context (Workflows) を切る**: trigger 評価のたびに User 属性を context 跨ぎで
  取得する必要が生じ、transactional trigger capture (決定2) が成立しなくなる。将来 trigger/action
  の種類が増え IdentityManagement から明確に分離すべき規模になった場合に再検討する。
- **`IdentityManagement.depends_on.Application` を追加して cycle を許容する**: `context_map` の
  DAG 制約を壊し `just yaml-check` の恒久的な例外運用が要る。将来 Application 側の
  `IdentityManagement` 依存 (UserRef/GroupRef) を型レベルの published language 参照から
  外せば解消できるが、本 WI の範囲では見送る。
- **context 間 rollback / saga による補償 transaction**: 実装・障害モードの複雑さに対し、
  「desired-state action + step checkpoint + 非補償」の組み合わせで同じ収束性を単純に得られる。
- **User mutation とは別 transaction で WorkflowRun を作成する (fire-and-forget)**: enqueue 失敗時に
  run が黙って失われる障害窓が生まれる。periodic dispatcher による回収だけでは
  transaction 境界の欠落を埋め合わせられない。

## 影響

- `spec/contexts/identity-management.yaml` に `LifecycleWorkflow` / `WorkflowTrigger` /
  `WorkflowAction` / `WorkflowRun` / `WorkflowStep` の models、`WorkflowDefinitionLifecycle` /
  `WorkflowRunLifecycle` の states、CRUD/enable/disable/archive/dry-run/run 履歴の interfaces、
  `LifecycleWorkflowTriggerHandoffLatency` の objective、scenarios、flows を追加する。
- `spec/contexts/application.yaml` に `AssignApplicationDesiredState` /
  `UnassignApplicationDesiredState` (internal interface) を追加する。`spec/scl.yaml` の
  `context_map` は変更しない (決定8)。
- `spec/contexts/jobs.yaml` の `JobKind` に `lifecycle_workflow_run` を追加する。
- `spec/contexts/audit.yaml` に `workflow.id` / `workflow_run.id` / `workflow_step.id` の
  検索属性を追加する (registry 実体は Go 側の単一の正)。
- `backend/identitymanagement/` に workflow aggregate、trigger evaluator、run planner、
  step executor、action port、dispatcher、`lifecycle_workflow_run` handler を実装する。
