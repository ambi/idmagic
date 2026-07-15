---
depends_on: [wi-6-real-email-sender-adapter, wi-126-async-job-runner]
status: completed
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# identity lifecycle workflow の定義・revision・純粋 domain 基盤を確立する

## Motivation
IdMagic は User lifecycle、Group、Application assignment、required action を個別に管理できるが、
入社、異動、休職、退職などの identity lifecycle event を一貫したアクセス変更へ変換する仕組みを
持たない。そのため、属性や status が更新されても、グループ・アプリ割当、次回ログイン時の必須操作、
通知、アカウント無効化は運用者が別々に実施する必要があり、実施漏れ、順序違反、再試行時の重複が
発生し得る。

本 WI は後続 WI が安全に実行機構を実装できるよう、構造化 definition、immutable revision、
純粋な trigger 評価と run plan、定義永続化の基盤を確立する。

## Scope
- **SCL — `spec/contexts/identity-management.yaml`**:
  - `glossary` に LifecycleWorkflow、WorkflowTrigger、WorkflowAction、WorkflowRun、WorkflowStep を追加する。
  - `models` に workflow 定義・revision、構造化 trigger/filter/action、run/step status、dry-run plan を追加する。
  - `states` に WorkflowDefinitionLifecycle と WorkflowRunLifecycle を追加し、許可遷移と終端状態を固定する。
  - `interfaces` に workflow CRUD、enable/disable/archive、dry-run、run 一覧・詳細を追加する。
  - `authorization` / interface `access` にテナント境界内の workflow 管理・履歴閲覧権限を追加する。
  - `scenarios` に正常、no-op、境界、拒否、部分失敗、再試行、disable、テナント分離を追加する。
  - `objectives` に trigger handoff の遅延目標と run/step 履歴の保持期間を追加する。
  - `flows` に admin の一覧、編集、dry-run、実行履歴、失敗詳細の画面遷移を追加する。
- **SCL — cross-context contract**:
  - `spec/contexts/application.yaml` に workflow から user assignment を desired-state で付与・解除する internal interface と、manual/dynamic group 境界の scenario を追加する。
  - `spec/contexts/jobs.yaml` に `lifecycle_workflow_run` JobKind、run_id の dedup、未 enqueue run の回収 scenario を追加する。
  - `spec/contexts/audit.yaml` に workflow_id / run_id / step_id を検索可能にする監査属性と scenario を追加する。
- **Backend — `backend/identitymanagement/`**:
  - workflow aggregate、validator、trigger evaluator、run planner、definition/run state machine と definition repository port を追加する。
  - memory / PostgreSQL に workflow definition と immutable revision を永続化する。
- **Decision / architecture / documentation**:
  - workflow 所有 context、transactional trigger capture、revision、再試行、loop suppression、非補償方針を ADR に記録する。
  - 新しい bounded context は作らず IdentityManagement module 内に置く。実装後に module map の要素が増える場合は `ARCHITECTURE.md` を同期する。
  - README に初期 trigger/action 一覧、delivery semantics、失敗時の再実行手順、運用例を追記する。

## Out of Scope
- 日付属性からの相対時刻 trigger、cron、一定時間待機する step。初期版の joiner/mover/leaver は、HR/SCIM/管理者が User を更新した時点を起点にする。
- HR system からの inbound provisioning 自体、外部 SaaS connector、任意 webhook、任意コード実行。
- DAG、並列 branch、loop、条件付き action、承認 step、補償 transaction。action は定義順を持つ線形 list とする。
- workflow enable 時の全 User への遡及適用。既存 User は dry-run で確認できるが、新しい lifecycle event がない限り自動実行しない。
- workflow action が発生させた User event を別 workflow へ連鎖させること。初期版は origin run を持つ event を trigger 対象外にして循環を防ぐ。
- custom email template editor。通知 action は製品内の固定 template key と locale fallback だけを使う。
- 複数 context をまたぐ rollback / exactly-once transaction。完了 step は補償せず、desired-state action と checkpoint で収束させる。
- access certification / recertification。これは [[wi-213-access-certification-campaigns]] が扱う。
- ユーザー起点の access request / approval。これは [[wi-214-self-service-access-request-and-approval]] が扱う。
- 汎用 job 管理画面。これは [[wi-157-job-admin-operations-surface]] が扱い、本 WI の UI は WorkflowRun のみを表示する。

## Detailed Design

### Initial trigger catalog

| Trigger kind | 発火条件 | 保持する差分 | 備考 |
| --- | --- | --- | --- |
| `user_created` | User 作成が commit される | 作成後 snapshot と source occurrence ID | import apply を含む。同一 source occurrence は一度だけ run 化する。 |
| `user_attributes_changed` | 指定 field のいずれかが実際に変化する | changed field、before/after の型付き値 | request に field が含まれただけで値が同じ場合は発火しない。 |
| `user_status_changed` | UserLifecycle.status が from から to へ遷移する | before/after status | 許可された lifecycle 遷移だけを対象にする。 |

filter は `field`、`operator`、型付き `value` の積集合 (AND) とする。初期 operator は
`eq` / `not_eq` / `in` / `exists` に限定し、TenantUserAttributeSchema と型が一致しない定義、PII の
自由文字 regex、未知 field は保存時に拒否する。trigger 評価は mutation の post-state に対して行い、
属性変更 trigger の before/after 判定だけは mutation 差分を使う。

### Initial action catalog

| Action kind | 入力 | desired-state / no-op の意味 |
| --- | --- | --- |
| `add_group_member` / `remove_group_member` | group_id | manual group の membership を存在 / 不在にする。dynamic group は validation error。 |
| `assign_application` / `unassign_application` | application_id、visibility | direct user assignment を存在 / 不在にする。group assignment は変更しない。 |
| `set_required_action` / `clear_required_action` | RequiredAction | User.required_actions の集合に値を存在 / 不在にする。 |
| `enable_user` / `disable_user` | reason | 許可された UserLifecycle 遷移だけを行う。既に目的 status なら no-op。Deleted / PendingDeletion は拒否する。 |
| `send_email` | 固定 template key、locale policy | `(run_id, step_id)` の delivery key で一回分だけ送信する。verified primary email がない場合は blocked failure。 |

definition はテナント内で name を一意、action は 1〜20 件、filter は 0〜20 件とする。resource ID、
RequiredAction、template key、attribute type は create/update と enable の双方で検証する。enable 後に参照先が
削除・archive された場合は run 時に step を失敗として記録し、別 tenant の同名 resource へ fallback しない。

### Definition and revision semantics

- LifecycleWorkflow は `draft` / `enabled` / `disabled` / `archived` を持つ。archived は終端で、履歴保全のため hard delete しない。
- create は revision 1 の draft を作る。意味の変わる update は revision を 1 増やし、過去 revision を immutable に残す。
- enable は完全 validation に成功した revision だけを対象とする。disabled / archived workflow は新しい run を作らない。
- WorkflowRun は作成時の workflow_id、revision、trigger occurrence、対象 user、展開済み action list を固定する。後から definition を変更しても既存 run の意味は変わらない。
- disable は新規 trigger を止め、まだ step を開始していない queued run を canceled にする。running run は現在の step の checkpoint 後、次 step の前に停止する。definition update は既存 run を cancel しない。
- archive は disabled の workflow にだけ許可し、definition と run history は retention 期間中参照可能にする。

### Durable trigger handoff

1. User mutation use case は before/after と source occurrence ID を生成し、enabled workflow revision を評価する。
2. User 保存と、該当する WorkflowRun・WorkflowStep の queued record 作成を同じ IdentityManagement transaction 境界で commit する。該当 workflow が 0 件なら run は作らない。
3. commit 後に dispatcher が `job_id IS NULL AND status = queued` の run を取得し、Jobs へ `dedup_key=lifecycle-workflow-run:{run_id}` で enqueue して job_id を関連付ける。
4. API process の即時 dispatch が失敗しても、worker process の periodic dispatcher が未関連付け run を再走査する。したがって User mutation 成功後に run が黙って失われない。
5. dispatcher の競合は compare-and-set と Jobs の dedup で収束させる。run と異なる tenant_id の Job は handler が fail-closed で拒否する。

WorkflowRun の一意制約は `(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)` とする。
source occurrence と run/step payload には action 実行に必要な最小限だけを保持し、password、token、email 本文、
秘密値は含めない。trigger の before/after 属性値は run 作成 transaction 内の評価にだけ使い、履歴には
changed field 名と predicate の成否を残す。

### Execution, retry, and ordering

- handler は `(tenant_id, target_user_id)` 単位の排他を取得し、同じ User に対する run を `triggered_at, run_id` 順に直列化する。別 User は並列実行できる。
- 各 step の直前に workflow disable/cancel と tenant/user/resource の存在を再確認し、実行後に outcome (`changed` / `no_op` / `failed` / `canceled`) と sanitised error code を checkpoint する。
- 一回の attempt では未完了 step を定義順にすべて試し、ある step が失敗しても後続 action を続行する。アクセス剥奪 action が無関係な通知失敗で止まらないためである。
- retryable failure が一つでもあれば handler は Jobs に retryable error を返す。retry では `changed` / `no_op` 済み step を飛ばし、failed step だけを再実行する。
- Job の attempt 上限到達時、全 step 成功なら `succeeded`、成功と失敗が混在すれば `partially_failed`、成功がなければ `failed` に終端する。no-op は成功として扱う。
- validation、tenant mismatch、Deleted User、不正 lifecycle 遷移、未知 action は non-retryable とする。repository timeout、一時的 SMTP error、lease loss は retryable とする。
- 複数 context の action は補償しない。履歴は「どこまで収束したか」を step ごとに示し、管理者の retry は同じ run の未完了 step に新しい Job を関連付ける。

### Dry-run semantics

- dry-run は workflow revision と対象 user_id を受け、同じ validator / trigger evaluator / action planner を使って action ごとに `would_change` / `no_op` / `blocked` と理由を返す。
- dry-run は WorkflowRun、Job、membership、assignment、required action、status、email delivery を一切作成・変更しない。
- dry-run 中に対象 resource が変わり得るため、結果に `evaluated_at` と workflow revision を含め、本実行の保証とは表示しない。
- dry-run 呼び出し自体は actor、tenant_id、workflow_id、revision、target_user_id を監査するが、PII の属性値や email address は監査 payload に含めない。

### API and UI contract

- `GET/POST /api/admin/lifecycle_workflows`
- `GET/PUT /api/admin/lifecycle_workflows/{workflow_id}`
- `POST /api/admin/lifecycle_workflows/{workflow_id}/enable`
- `POST /api/admin/lifecycle_workflows/{workflow_id}/disable`
- `DELETE /api/admin/lifecycle_workflows/{workflow_id}` (archive)
- `POST /api/admin/lifecycle_workflows/{workflow_id}/dry_run`
- `GET /api/admin/lifecycle_workflows/{workflow_id}/runs`
- `GET /api/admin/lifecycle_workflow_runs/{run_id}`
- `POST /api/admin/lifecycle_workflow_runs/{run_id}/retry`

mutation は CSRF、admin authentication、tenant-scoped authorization、revision precondition を要求する。
他 tenant の ID は existence を漏らさず not-found として扱う。list/history の pagination は既存 admin API の
契約に合わせ、本 WI で独自の pagination 方式を増やさない。

editor は自由記述 script ではなく trigger/filter/action の型付き form とし、保存前 validation と action 順序を
表示する。run detail は trigger、固定 revision、target user、Job attempt、各 step の outcome/error code/timestamp を
表示し、secret、email 本文、attribute value は表示しない。

### Persistence and observability

- PostgreSQL は `lifecycle_workflows`、`lifecycle_workflow_revisions`、`lifecycle_workflow_runs`、
  `lifecycle_workflow_steps`、`lifecycle_notification_deliveries` を IdentityManagement ownership で持つ。
- definition の polymorphic trigger/action は versioned JSONB として保持し、identity、tenant、revision、state、時刻、
  一意制約、検索 index は型付き column で持つ。unknown schema version は実行せず fail-closed にする。
- run/step/delivery は tenant_id を冗長保持し、FK と repository query の双方で tenant boundary を強制する。
- run/step history と notification delivery key は 30 日保持する。workflow definition と revision は archive 後も
  参照中の run がある間は削除しない。保持期間 cleanup は tenant-safe な batch とする。
- `LifecycleWorkflowCreated/Updated/Enabled/Disabled/Archived`、`LifecycleWorkflowRunStarted/Succeeded/PartiallyFailed/Failed/Canceled`、
  `LifecycleWorkflowStepFailed` を emit し、workflow_id / run_id / target_user_id / outcome を監査検索可能にする。
- metric は trigger から run 作成までの遅延、queued run 数、run outcome、step failure、retry、dispatcher lag を
  tenant ID を label にせず集計する。log に attribute value、email、job params 全文を出さない。

## Plan
- 先に SCL と ADR で ownership、transactional capture、revision、run/step 状態、action port 契約を確定し、派生物を再生成する。
- IdentityManagement の domain model / validator / state machine を pure logic として実装し、正常・境界・拒否を table test する。
- repository と migration を memory → PostgreSQL の順で実装し、一意制約、tenant isolation、concurrent dispatch を adapter test する。
- trigger capture と dispatcher を実装してから action executor を一種類ずつ追加し、各 context の既存 use case と同じ invariant を通す。
- HTTP と UI は domain contract が固まった後に追加し、最後に API → run → worker → action → history の E2E を通す。
- 任意 expression engine、汎用 workflow context、別 bounded context 化は採らない。identity lifecycle に必要な型付き語彙を先に固定する。

## Tasks
- [x] T001 [SCL] IdentityManagement の glossary/models/states/interfaces/authorization/scenarios/objectives/flows を追加する。
- [x] T002 [SCL] Application の internal desired-state assignment、Jobs の JobKind/handoff、Audit の検索属性を追加する。
- [x] T003 [Decision] transactional trigger capture、revision、retry/checkpoint、loop suppression、retention、非補償方針を ADR に記録する。
- [x] T004 [Domain] workflow definition validator、trigger evaluator、run planner、definition/run state machine を実装する。
- [x] T005 [Persistence] memory / PostgreSQL の workflow definition/revision repository と schema を実装する。
- [x] T006 [Verify] SCL、domain、memory/PostgreSQL adapter を検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just scl-render`
- `just sqlc-generate`
- `just verify-go`
- `just verify-ui`
- `just test-ui-e2e`
- `just verify`
- 自動: 同一 source occurrence の再配送と Job lease reclaim 後も run が一つで、完了 step の副作用が増えない。
- 自動: User mutation commit 後に即時 enqueue を失敗させても periodic dispatcher が run を回収して終端させる。
- 自動: action 途中の一時失敗後に後続 action を実行し、retry では成功済み step を飛ばして失敗 step だけが収束する。
- 自動: 同一 User の複数 run は順序どおり、別 User の run は並列に進み、tenant を跨ぐ resource 参照は拒否される。
- 自動: disable と retry が競合しても新しい step を開始せず、archived workflow から run が生成されない。
- 手動: department 変更で group/application assignment と required action が反映され、run detail に各 step outcome が表示される。
- 手動: 退職相当の status change で access removal と disable が行われ、email 失敗時も access removal が完了する。
- 手動: dry-run は would-change/no-op/blocked を表示するが、run、Job、assignment、email を作成しない。

## Risk Notes
- **欠落**: User 更新と Job enqueue の単純な二重書きは障害窓を作る。User と WorkflowRun を同一 transaction で確定し、未 enqueue run を dispatcher が回収する。
- **重複**: Jobs は at-least-once である。source occurrence 一意制約、run dedup、step checkpoint、desired-state action、notification delivery key を重ねる。
- **部分成功**: context 間 rollback は実装しない。全 step を試して access removal を前進させ、結果を step 単位で残し、失敗分だけ再試行する。
- **循環**: workflow-origin event は新しい workflow trigger にしない。origin metadata が欠ける内部 mutation は validation/test で拒否する。
- **誤設定**: typed catalog、保存時/enable 時/run 時 validation、dry-run、revision 固定、上限、destructive action 警告で blast radius を抑える。
- **権限と PII**: workflow は system actor として動くが tenant boundary を越えない。attribute value、email、secret を run/error/log/audit に保存せず、履歴には ID と判定結果だけを残す。
- **停止操作**: disable は実行中 step を強制中断しない。step 境界で停止するため、UI と API response にこの意味を明記する。

## Completion

- **Completed At**: 2026-07-16
- **Summary**: LifecycleWorkflow definition と immutable revision、純粋な trigger evaluator/run planner、memory/PostgreSQL repository を追加した。durable run の capture・execution は wi-217 以降へ分割した。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just test-go` - passed
- **affected_guarantees_state**: definition/revision の tenant 分離、楽観ロック、archive 終端、workflow-origin mutation の loop suppression、run plan の action snapshot をテストで保証する。
- **evidence**:
  - procedure: SCL validation and Go test suite
    result: passed
    artifacts: []
