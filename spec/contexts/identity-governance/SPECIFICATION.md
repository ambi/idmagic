---
context: identity-governance
updated_at: 2026-08-11
---

# IdGovernance Specification

## Overview

identity ガバナンスの policy と orchestration を所有する。LifecycleWorkflow (JML 自動化) の
definition・trigger 評価・WorkflowRun 実行・executor action を集約し、将来の IGA 機能
(認証キャンペーン、アクセスリクエスト/承認、エンタイトルメント/SoD、JIT 特権昇格) の受け皿となる。
identity principal の record-of-truth は IdManagement が、Application 割当は Application が
所有し、IdGovernance は User ライフサイクルイベントを購読し、冪等コマンド surface 経由で
これら record context の状態を書き換える。

The `IdGovernance` context owns `LifecycleWorkflow` policy and orchestration — trigger definitions,
action definitions, and run/step execution — while record contexts (`IdManagement`, `Application`)
keep owning the state those actions change. This context was split out of `IdManagement` once the
lifecycle-workflow slice had grown large enough to smear across
layers of a context whose primary job is being the identity-principal record of truth; the rejected
alternative of leaving it module-local inside `IdManagement` would not provide a home for the
broader IGA roadmap (access campaigns, entitlements, JIT elevation).

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| LifecycleWorkflow | テナント管理者が事前定義する、User の作成・属性変更・status 遷移を trigger として<br>構造化 action を実行する定義。draft / enabled / disabled / archived のライフサイクルを持ち、<br>enable された revision だけが新規 WorkflowRun を生成する。DAG や条件分岐は持たず、action は<br>定義順の線形 list。 | lifecycle workflow, ライフサイクルワークフロー |
| WorkflowTrigger | LifecycleWorkflow の revision が固定する発火条件。user_created / user_attributes_changed /<br>user_status_changed のいずれかの kind と、0〜20 件の型付き filter (field/operator/value の<br>AND) からなる。評価は User mutation の post-state に対して行い、属性変更の before/after 判定<br>だけは mutation 差分を使う。 | workflow trigger, トリガー |
| WorkflowAction | WorkflowRun が展開する desired-state 操作 1 件。add_group_member / remove_group_member /<br>assign_application / unassign_application / set_required_action / clear_required_action /<br>enable_user / disable_user / send_email のいずれかで、既に目的状態なら no_op として扱う<br>冪等な操作。 | workflow action, アクション |
| WorkflowRun | WorkflowTrigger の発火 1 回から生成される実行単位。作成時の workflow_id、revision、<br>trigger occurrence、対象 user、展開済み action list を固定し、後から definition が変わっても<br>意味が変わらない。Jobs の lifecycle_workflow_run Job に紐付いて実行される。 | workflow run, ワークフロー実行 |
| WorkflowStep | WorkflowRun 内の action 1 件分の実行記録。outcome (changed / no_op / failed / canceled) と<br>sanitised error code、実行時刻を checkpoint する。retry は failed step だけを再実行し、<br>changed / no_op 済み step は飛ばす。 | workflow step, ステップ |

## State Transitions

### WorkflowDefinitionLifecycle

LifecycleWorkflow.status の状態機械。draft で作成し、完全 validation に成功した revision を
enable すると新規 trigger 評価対象になる。disable は新規 trigger を止め、再度 enable できる。
管理者の削除は draft/enabled/disabled のすべてから可能で、enabled の削除は新規 trigger を止めて
queued run を cancel する。archived は削除済み定義を参照整合性のため内部保持する終端状態であり、
管理画面と通常 API には公開しない。run 履歴と LifecycleWorkflowDeleted 監査イベントは保持する。

Initial: `draft`
Terminal: `archived`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| draft | LifecycleWorkflowEnabled | — | enabled |  |
| enabled | LifecycleWorkflowDisabled | — | disabled |  |
| disabled | LifecycleWorkflowEnabled | — | enabled |  |
| draft | LifecycleWorkflowDeleted | — | archived |  |
| enabled | LifecycleWorkflowDeleted | — | archived |  |
| disabled | LifecycleWorkflowDeleted | — | archived |  |

### WorkflowRunLifecycle

WorkflowRun.status の状態機械。queued で作成され、dispatcher が job_id を関連付けて
running に遷移する (最初の step 実行開始)。1 attempt では未完了 step を定義順にすべて試し、
Job の attempt 上限到達時に全 step 成功なら succeeded、成功と失敗が混在すれば
partially_failed、成功が一つもなければ failed に終端する (no-op は成功として扱う)。
Jobs レベルの retry 中は run.status を running のまま保持する。disable は未開始の queued run を
canceled にし、running run は現在 step の checkpoint 後、次 step の前で canceled にする。

Initial: `queued`
Terminal: `succeeded`, `partially_failed`, `failed`, `canceled`

| From | Event | Guard | To | Effects |
|---|---|---|---|---|
| queued | LifecycleWorkflowRunStarted | — | running |  |
| running | LifecycleWorkflowRunSucceeded | — | succeeded |  |
| running | LifecycleWorkflowRunPartiallyFailed | — | partially_failed |  |
| running | LifecycleWorkflowRunFailed | — | failed |  |
| queued | LifecycleWorkflowRunCanceled | — | canceled |  |
| running | LifecycleWorkflowRunCanceled | — | canceled |  |

## Authorization Boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.

## Design

### Trigger capture via transactional outbox

`IdManagement` writes User lifecycle events (`UserCreated`, `UserAttributesChanged`,
`UserStatusChanged`) to the existing transactional outbox in the same transaction as the user
mutation itself. `IdGovernance` consumes these to create `WorkflowRun`/`WorkflowStep` rows. This
replaces the earlier same-context, same-transaction capture — used back when `LifecycleWorkflow`
still lived inside `IdManagement` — with a cross-context contract that closes the same failure
window — a User update whose triggering run silently never gets created — without requiring a
single shared transaction across contexts. Because delivery is at-least-once, the
`(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)` uniqueness constraint
still collapses duplicate deliveries of the same trigger occurrence into one run.

### Action execution via published command surface

The nine executor actions (group membership add/remove, application assign/unassign, user
enable/disable, required-action set/clear, send email) never call record-context domain types
directly. `IdManagement` and `Application` instead expose idempotent command surfaces as published
interfaces that the executor calls across the context boundary; composition-root wiring injects the
concrete adapters (the same ports-and-adapters pattern established before the context split).
Promoting these to published interfaces — rather than leaving them as internal interfaces — was
possible once `IdGovernance` became the dependency source: `IdGovernance` depending on
`IdManagement`/`Application` cannot cycle back, unlike the earlier `IdManagement -> Application`
direction that had to be routed around.

### Partial failure and loop suppression

A single attempt runs every not-yet-completed step in definition order and does not stop at a
failed step, so an unrelated notification failure cannot block an access-revocation step; the run
terminates `succeeded`, `partially_failed`, or `failed` depending on the mix, with no cross-context
compensation/rollback. `WorkflowRunTriggerSnapshot` records the origin run/step of any User mutation
an action performs, and trigger evaluation excludes mutations carrying that origin metadata — this
closes the action-to-trigger loop structurally, at the evaluator's input, rather than with a runtime
guard. `WorkflowRun`/`WorkflowStep`/`LifecycleNotificationDelivery` follow the same 30-day retention
as `Job` records.

### Design Decisions

- `IdGovernance` is split out of `IdManagement` to own `LifecycleWorkflow` policy and orchestration,
  leaving `IdManagement` as the identity-principal record of truth, so the lifecycle-workflow slice
  has a home for the broader IGA roadmap (access campaigns, entitlements, JIT elevation) instead of
  continuing to smear across a record context's layers.
- The transactional trigger-capture pattern (writing the workflow run in the same transaction as the
  triggering mutation), the trigger-occurrence uniqueness constraint that collapses at-least-once
  redelivery, ports-and-adapters action execution, and the revision-pinning/partial-failure/loop-
  suppression/retention rules were all established for `LifecycleWorkflow` while it still lived
  inside `IdManagement`, and carried forward unchanged when `IdGovernance` was split out.

## Scenarios

### REQ-IDGOVERNANCE-001: 管理者はライフサイクルワークフローを作成できる
- ACTOR TenantAdministrator
- GIVEN 管理者が認証済みである
- WHEN 管理者が一覧画面の新規作成操作から専用の作成画面へ移動する
- THEN trigger 種別と action 種別が日本語の説明付きで表示される
- WHEN 管理者が trigger 種別と action 種別を選択する
  - ALT group または application を使う action に選択可能な参照先がない → 作成は拒否され、先に参照先を作成する必要があることが日本語で表示される
- WHEN 管理者が action ごとの必須設定を入力し、複数 action の実行順を並べ替える
  - ALT trigger または action の必須設定が不足している → 作成操作は無効になり、不足している設定が日本語で表示される
- WHEN 管理者がワークフローを作成する
- THEN 選択した trigger と順序付き action が revision 1 の draft として返される
- THEN UI は trigger、action、draft、revision、current_revision などの内部識別子を露出せず、日本語の表示名と意味を表示する

### REQ-IDGOVERNANCE-002: 管理者は既存ライフサイクルワークフローの定義を編集できる
- ACTOR TenantAdministrator
- GIVEN 管理者が既存ワークフローを選択している
- WHEN 管理者が一覧画面の編集操作から専用の編集画面へ移動する
- THEN 現在の trigger と順序付き action が日本語の表示名と説明付きでフォームに復元される
- WHEN 管理者が trigger または action を変更して保存する
- THEN current_revision が増え、変更した定義が一覧と編集フォームに反映される

### REQ-IDGOVERNANCE-003: 管理者が定義したworkflowが部署変更でグループとアプリ割当を更新する
- ACTOR TenantAdministrator
- GIVEN roles=["admin"] の管理者 "operator" が LifecycleWorkflow "engineering-onboarding" (trigger=user_attributes_changed department、action=add_group_member+assign_application) を enable 済みである
- GIVEN ユーザー "alice" は department="Sales" である
- WHEN 管理者 "operator" がユーザー "alice" の department を "Engineering" に更新する
- THEN WorkflowRun が作成され trigger snapshot に changed_fields=["department"] を保持する
- THEN worker が run を実行し add_group_member と assign_application の step が changed になる
- THEN run の status は succeeded である
- THEN "LifecycleWorkflowRunSucceeded" が発行される

### REQ-IDGOVERNANCE-004: User変更とWorkflowRunのcaptureは同じ整合性境界で確定しenqueue障害から回復する
- ACTOR TenantAdministrator
- GIVEN user_created / user_attributes_changed / user_status_changed のいずれかに一致する enabled workflow がある
- WHEN workflow に一致する User mutation が実行される
- THEN User mutation と重複排除キー (tenant_id, workflow_id, revision, source_occurrence_id, target_user_id) を持つ queued WorkflowRun および pending WorkflowStep が同一 transaction で確定する
- THEN 同じ source occurrence の再配送は既存 WorkflowRun に収束し、step を重複作成しない
- WHEN lifecycle_workflow_run Job を enqueue する
  - ALT enqueue が一時的に失敗する → WorkflowRun は job_id 未設定の queued 状態で残り、worker の周期的 dispatcher が再走査して重複しない Job を関連付ける
- THEN Job が WorkflowRun に関連付けられる

### REQ-IDGOVERNANCE-005: 既に目的状態のactionはno_opとして扱われる
- ACTOR TenantAdministrator
- GIVEN ユーザー "alice" は既に対象 Group のメンバーである
- GIVEN enable 済みの workflow は "alice" の department 変更を trigger とし add_group_member action を定義している
- WHEN workflow が "alice" の add_group_member step を実行する
- THEN step outcome は no_op である
- THEN membership の重複行は作成されない
- THEN run の status は succeeded である

### REQ-IDGOVERNANCE-006: 値が変わらない属性更新とdynamic groupへのaction指定は境界として扱われる
- ACTOR TenantAdministrator
- GIVEN "alice" の department は既に "Engineering" である
- WHEN 管理者が "alice" の department に同じ値 "Engineering" を指定して更新する
- THEN user_attributes_changed trigger は発火せず WorkflowRun は作成されない
- WHEN 管理者が workflow の add_group_member action に dynamic group を指定して保存する
- THEN 保存は InvalidRequestError で拒否される

### REQ-IDGOVERNANCE-007: 未知fieldや別テナントresourceを参照するworkflowはenableできない
- ACTOR TenantAdministrator
- GIVEN tenant_id "acme" の管理者が workflow を編集している
- WHEN TenantUserAttributeSchema に存在しない field を filter に指定して保存する
- THEN エラー "InvalidRequestError"
- WHEN tenant_id "default" の group_id を action に指定して enable する
- THEN エラー "InvalidRequestError"

### REQ-IDGOVERNANCE-008: 通知失敗があってもアクセス剥奪actionは前進する
- ACTOR TenantAdministrator
- GIVEN enable 済みの leaver workflow が disable_user、remove_group_member、send_email を定義順に持つ
- GIVEN 対象 User に verified primary email がない
- WHEN 対象 User に退職相当の status 変更が発生する
- THEN WorkflowRun が作成される
- THEN disable_user と remove_group_member の step は changed になる
- THEN send_email の step は blocked failure になる
- THEN run の status は partially_failed である
- THEN "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される

### REQ-IDGOVERNANCE-009: 一時的な失敗はretryで成功済みstepを飛ばして収束する
- ACTOR TenantAdministrator
- GIVEN enable 済みの workflow による run が 1 attempt 目で repository timeout により一部 step 未完了である
- WHEN handler が Jobs に retryable error を返す
- THEN Jobs が backoff 後に同じ run の Job を再試行する
- THEN retry では changed / no_op 済み step は再実行されず failed step だけが再実行される
- WHEN 管理者が run detail を確認する
- THEN status は succeeded である

### REQ-IDGOVERNANCE-010: 同一ユーザーのrunは発火順に直列化され副作用前に再検証される
- ACTOR TenantAdministrator
- GIVEN 同じ target user に対する queued WorkflowRun が triggered_at 順に2件ある
- WHEN worker が同じ target user の queued WorkflowRun を取得する
- THEN 先行 run が terminal になるまで後続 run の action を開始しない
- THEN 各 action の直前に対象 user と group/application resource を同一 tenant 内で再取得する
- THEN 削除済みまたは別 tenant の resource を参照する step は sanitised error code で failed checkpoint される

### REQ-IDGOVERNANCE-011: disableは未開始のqueued runをcancelし実行中のrunはstep境界で止まる
- ACTOR TenantAdministrator
- GIVEN workflow が enabled で running の WorkflowRun "run-1" と queued の WorkflowRun "run-2" が存在する
- WHEN 管理者が workflow を disable する
  - ALT disable と retry が競合する → disable 済み workflow の run に対する retry は新しい step を開始しない → エラー "InvalidRequestError"
- THEN "run-2" は直ちに canceled になる
- THEN "run-1" は現在の step の checkpoint 後、次の step の前で canceled になる
- THEN disable 後に新しい trigger は run を作らない

### REQ-IDGOVERNANCE-012: 別テナントのworkflowとresourceはテナント境界を越えない
- ACTOR TenantAdministrator
- GIVEN tenant_id "default" に workflow が存在する
- WHEN tenant_id "acme" の管理者がその workflow_id を指定して取得する
- THEN workflow は未存在として扱われる
- WHEN tenant_id "acme" の workflow action が tenant_id "default" の group_id を参照する
- THEN 保存または run 実行時に InvalidRequestError で拒否され、別テナントの同名 resource へ fallback しない

### REQ-IDGOVERNANCE-013: dry-runはaction結果を試算するがWorkflowRunやJobを作成しない
- ACTOR TenantAdministrator
- GIVEN workflow が enabled で対象 User が action の一部について既に目的状態 (例: 既に group member) である
- WHEN 管理者が dry_run を呼び出す
  - ALT enable 済みの draft 編集が current_revision にあり enabled_revision は未編集の内容を指す → dry-run は enabled_revision の action と trigger を評価し draft の変更を反映しない
- THEN 対象 User の現在の group membership / application assignment / required action / status / email 検証状態が実際に読まれる
  - ALT workflow の trigger の filter が対象 User の現在属性に一致しない → 応答は全 action を blocked、reason "trigger_not_matched" として返す
- THEN 既に目的状態の action は no_op、状態が変わる action は would_change、resource が存在しないなど実行不能な action は blocked と理由を返す
- THEN WorkflowRun、Job、membership、assignment、required action、status、email は一切作成・変更されない
