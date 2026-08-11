# IdGovernance Requirements

> This Markdown file is the normative, language-independent home for product requirements. Models and API contracts live in the adjacent TypeSpec source.

## Requirements

### REQ-IDGOVERNANCE-001: 管理者はライフサイクルワークフローを作成できる
- Actor: TenantAdministrator
- Given: 管理者が認証済みである
- Then: 管理者が一覧画面の新規作成操作から専用の作成画面へ移動する
- Then: 管理者が日本語の説明を確認しながら trigger 種別と action 種別を選択する
- Then: 管理者が action ごとの必須設定を入力し、複数 action の実行順を並べ替える
- Then: 管理者がワークフローを作成すると、選択した trigger と順序付き action が revision 1 の draft として返される
- Then: UI は trigger、action、draft、revision、current_revision などの内部識別子を露出せず、日本語の表示名と意味を表示する
- Alternative (trigger または action の必須設定が不足している): 作成操作は無効になり、不足している設定が日本語で表示される
- Alternative (group または application を使う action に選択可能な参照先がない): 作成は拒否され、先に参照先を作成する必要があることが日本語で表示される

### REQ-IDGOVERNANCE-002: 管理者は既存ライフサイクルワークフローの定義を編集できる
- Actor: TenantAdministrator
- Given: 管理者が既存ワークフローを選択している
- Then: 管理者が一覧画面の編集操作から専用の編集画面へ移動する
- Then: 現在の trigger と順序付き action が日本語の表示名と説明付きでフォームに復元される
- Then: 管理者が trigger または action を変更して保存する
- Then: current_revision が増え、変更した定義が一覧と編集フォームに反映される

### REQ-IDGOVERNANCE-003: 管理者が定義したworkflowが部署変更でグループとアプリ割当を更新する
- Actor: TenantAdministrator
- Given: roles=["admin"] の管理者 "operator" が LifecycleWorkflow "engineering-onboarding" (trigger=user_attributes_changed department、action=add_group_member+assign_application) を enable 済みである
- Given: ユーザー "alice" は department="Sales" である
- Then: 管理者 "operator" がユーザー "alice" の department を "Engineering" に更新する
- Then: WorkflowRun が作成され trigger snapshot に changed_fields=["department"] を保持する
- Then: worker が run を実行し add_group_member と assign_application の step が changed になる
- Then: run の status は succeeded である
- Then: "LifecycleWorkflowRunSucceeded" が発行される

### REQ-IDGOVERNANCE-004: User変更とWorkflowRunのcaptureは同じ整合性境界で確定しenqueue障害から回復する
- Actor: TenantAdministrator
- Given: user_created / user_attributes_changed / user_status_changed のいずれかに一致する enabled workflow がある
- Then: User mutation と重複排除キー (tenant_id, workflow_id, revision, source_occurrence_id, target_user_id) を持つ queued WorkflowRun および pending WorkflowStep が同一 transaction で確定する
- Then: 同じ source occurrence の再配送は既存 WorkflowRun に収束し、step を重複作成しない
- Alternative (Job enqueue が一時的に失敗する): WorkflowRun は job_id 未設定の queued 状態で残る → worker の周期的 dispatcher が未 enqueue run を再走査し、重複しない lifecycle_workflow_run Job を関連付ける

### REQ-IDGOVERNANCE-005: 既に目的状態のactionはno_opとして扱われる
- Actor: TenantAdministrator
- Given: ユーザー "alice" は既に対象 Group のメンバーである
- Given: enable 済みの workflow が "alice" の department 変更で add_group_member を実行する
- Then: add_group_member の step outcome は no_op である
- Then: membership の重複行は作成されない
- Then: run の status は succeeded である

### REQ-IDGOVERNANCE-006: 値が変わらない属性更新とdynamic groupへのaction指定は境界として扱われる
- Actor: TenantAdministrator
- Given: "alice" の department は既に "Engineering" である
- Then: 管理者が "alice" の department に同じ値 "Engineering" を指定して更新する
- Then: user_attributes_changed trigger は発火せず WorkflowRun は作成されない
- Alternative (workflow の add_group_member action に dynamic group を指定して保存する): 保存は拒否される → エラー "InvalidRequestError"

### REQ-IDGOVERNANCE-007: 未知fieldや別テナントresourceを参照するworkflowはenableできない
- Actor: TenantAdministrator
- Given: tenant_id "acme" の管理者が workflow を編集している
- Then: TenantUserAttributeSchema に存在しない field を filter に指定して保存する
- Then: エラー "InvalidRequestError"
- Alternative (tenant_id "default" の group_id を action に指定して enable する): enable は拒否される → エラー "InvalidRequestError"

### REQ-IDGOVERNANCE-008: 通知失敗があってもアクセス剥奪actionは前進する
- Actor: TenantAdministrator
- Given: enable 済みの leaver workflow が disable_user、remove_group_member、send_email を定義順に持つ
- Given: 対象 User に verified primary email がない
- Then: 退職相当の status 変更で run が作成される
- Then: disable_user と remove_group_member の step は changed になる
- Then: send_email の step は blocked failure になる
- Then: run の status は partially_failed である
- Then: "LifecycleWorkflowRunPartiallyFailed" と "LifecycleWorkflowStepFailed" が発行される

### REQ-IDGOVERNANCE-009: 一時的な失敗はretryで成功済みstepを飛ばして収束する
- Actor: TenantAdministrator
- Given: enable 済みの workflow による run が 1 attempt 目で repository timeout により一部 step 未完了である
- Then: handler が Jobs に retryable error を返す
- Then: Jobs が backoff 後に同じ run の Job を再試行する
- Then: retry では changed / no_op 済み step は再実行されず failed step だけが再実行される
- Then: 管理者が run detail を確認すると status は succeeded である

### REQ-IDGOVERNANCE-010: 同一ユーザーのrunは発火順に直列化され副作用前に再検証される
- Actor: TenantAdministrator
- Given: 同じ target user に対する queued WorkflowRun が triggered_at 順に2件ある
- Then: worker は先行 run が terminal になるまで後続 run の action を開始しない
- Then: 各 action の直前に対象 user と group/application resource を同一 tenant 内で再取得する
- Then: 削除済みまたは別 tenant の resource を参照する step は sanitised error code で failed checkpoint される

### REQ-IDGOVERNANCE-011: disableは未開始のqueued runをcancelし実行中のrunはstep境界で止まる
- Actor: TenantAdministrator
- Given: workflow が enabled で running の WorkflowRun "run-1" と queued の WorkflowRun "run-2" が存在する
- Then: 管理者が workflow を disable する
- Then: "run-2" は直ちに canceled になる
- Then: "run-1" は現在の step の checkpoint 後、次の step の前で canceled になる
- Then: disable 後に新しい trigger は run を作らない
- Alternative (disable と retry が競合する): disable 済み workflow の run に対する retry は新しい step を開始しない → エラー "InvalidRequestError"

### REQ-IDGOVERNANCE-012: 別テナントのworkflowとresourceはテナント境界を越えない
- Actor: TenantAdministrator
- Given: tenant_id "acme" の管理者が tenant_id "default" の workflow_id を指定して取得する
- Then: workflow は未存在として扱われる
- Alternative (tenant_id "acme" の workflow の action が tenant_id "default" の group_id を参照する): 保存または run 実行時に拒否され、別テナントの同名 resource へ fallback しない → エラー "InvalidRequestError"

### REQ-IDGOVERNANCE-013: dry-runはaction結果を試算するがWorkflowRunやJobを作成しない
- Actor: TenantAdministrator
- Given: workflow が enabled で対象 User が action の一部について既に目的状態 (例: 既に group member) である
- Then: 管理者が dry_run を呼び出す
- Then: 対象 User の現在の group membership / application assignment / required action / status / email 検証状態が実際に読まれる
- Then: 既に目的状態の action は no_op、状態が変わる action は would_change、resource が存在しないなど実行不能な action は blocked と理由を返す
- Then: WorkflowRun、Job、membership、assignment、required action、status、email は一切作成・変更されない
- Alternative (enable 済みの draft 編集が current_revision にあり enabled_revision は未編集の内容を指す): dry-run は enabled_revision の action と trigger を評価し draft の変更を反映しない
- Alternative (workflow の trigger の filter が対象 User の現在属性に一致しない): 応答は全 action を blocked、reason "trigger_not_matched" として返す

### REQ-IDGOVERNANCE-014: ListLifecycleWorkflows
管理者が所属テナントの未削除 LifecycleWorkflow を一覧する。削除済み定義は返さない。

### REQ-IDGOVERNANCE-015: CreateLifecycleWorkflow
管理者が LifecycleWorkflow を作成する。workflow 本体と revision 1 の draft をともに永続化し、name は未削除 workflow の間でテナント内一意とする。削除済み workflow の name は再利用できる。
- Postcondition: emitted.exists(e, e is LifecycleWorkflowCreated)

### REQ-IDGOVERNANCE-016: GetLifecycleWorkflow
管理者が LifecycleWorkflow 1 件を取得する。別テナントの workflow は未存在として扱う。

### REQ-IDGOVERNANCE-017: UpdateLifecycleWorkflow
管理者が LifecycleWorkflow の trigger/actions/表示情報を更新する。expected_revision が
current_revision と一致しない場合は WorkflowRevisionConflictError で拒否する (lost update 防止)。
trigger または actions が変わる場合は current_revision を 1 増やすが status/enabled_revision は
変えない (再度 EnableLifecycleWorkflow を呼ぶまで旧 revision の trigger 評価が続く)。archived
workflow は拒否する。

### REQ-IDGOVERNANCE-018: EnableLifecycleWorkflow
管理者が current_revision を validation し、成功すれば enabled_revision に設定して
status=enabled にする。以後この revision が trigger 評価対象になる。validation 失敗または
archived workflow は拒否する。

### REQ-IDGOVERNANCE-019: DisableLifecycleWorkflow
管理者が LifecycleWorkflow を disable する。新規 trigger を止め、未開始の queued run を
canceled にする。running run は現在 step の checkpoint 後、次 step の前で停止する。

### REQ-IDGOVERNANCE-020: DeleteLifecycleWorkflow
管理者が LifecycleWorkflow を状態に関係なく削除する。削除後は一覧・取得・更新・名前重複判定から除外する。enabled の場合は新規 trigger を止め、queued run を canceled にする。既存 run と監査イベントは保持する。
- Postcondition: emitted.exists(e, e is LifecycleWorkflowDeleted)

### REQ-IDGOVERNANCE-021: DryRunLifecycleWorkflow
管理者が enabled_revision (未指定なら current_revision) を対象 user へ試算する。同じ
validator / trigger evaluator / action planner を使い、対象 User の現在の group membership /
application assignment / required action / status / email 検証状態を実際に読んで action ごとの
would_change / no_op / blocked を判定する。trigger の filter が対象 User に一致しない場合は
全 action を blocked (reason: trigger_not_matched) として返す。対象 user が存在しないか別
tenant の場合は InvalidRequestError を返す。WorkflowRun / Job / membership / assignment /
required action / status / email のいずれも作成・変更しない。

### REQ-IDGOVERNANCE-022: ListLifecycleWorkflowRuns
管理者が特定 LifecycleWorkflow の run 履歴を新しい順に一覧する。

### REQ-IDGOVERNANCE-023: GetLifecycleWorkflowRun
管理者が WorkflowRun 1 件の詳細 (trigger、revision、target user、Job attempt、各 step) を取得する。別テナントの run は未存在として扱う。

### REQ-IDGOVERNANCE-024: RetryLifecycleWorkflowRun
管理者が partially_failed または failed の WorkflowRun を再試行する。未完了 (failed) step
だけを対象に新しい Job を関連付ける。changed / no_op 済み step は変更しない。disable と競合する
場合は新しい step を開始しない。

## Glossary

| Term | Definition | Aliases |
|---|---|---|
| LifecycleWorkflow | テナント管理者が事前定義する、User の作成・属性変更・status 遷移を trigger として<br>構造化 action を実行する定義。draft / enabled / disabled / archived のライフサイクルを持ち、<br>enable された revision だけが新規 WorkflowRun を生成する。DAG や条件分岐は持たず、action は<br>定義順の線形 list。 | lifecycle workflow, ライフサイクルワークフロー |
| WorkflowTrigger | LifecycleWorkflow の revision が固定する発火条件。user_created / user_attributes_changed /<br>user_status_changed のいずれかの kind と、0〜20 件の型付き filter (field/operator/value の<br>AND) からなる。評価は User mutation の post-state に対して行い、属性変更の before/after 判定<br>だけは mutation 差分を使う。 | workflow trigger, トリガー |
| WorkflowAction | WorkflowRun が展開する desired-state 操作 1 件。add_group_member / remove_group_member /<br>assign_application / unassign_application / set_required_action / clear_required_action /<br>enable_user / disable_user / send_email のいずれかで、既に目的状態なら no_op として扱う<br>冪等な操作。 | workflow action, アクション |
| WorkflowRun | WorkflowTrigger の発火 1 回から生成される実行単位。作成時の workflow_id、revision、<br>trigger occurrence、対象 user、展開済み action list を固定し、後から definition が変わっても<br>意味が変わらない。Jobs の lifecycle_workflow_run Job に紐付いて実行される。 | workflow run, ワークフロー実行 |
| WorkflowStep | WorkflowRun 内の action 1 件分の実行記録。outcome (changed / no_op / failed / canceled) と<br>sanitised error code、実行時刻を checkpoint する。retry は failed step だけを再実行し、<br>changed / no_op 済み step は飛ばす。 | workflow step, ステップ |

## State machines

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
| draft | LifecycleWorkflowEnabled | "" | enabled |  |
| enabled | LifecycleWorkflowDisabled | "" | disabled |  |
| disabled | LifecycleWorkflowEnabled | "" | enabled |  |
| draft | LifecycleWorkflowDeleted | "" | archived |  |
| enabled | LifecycleWorkflowDeleted | "" | archived |  |
| disabled | LifecycleWorkflowDeleted | "" | archived |  |

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
| queued | LifecycleWorkflowRunStarted | "" | running |  |
| running | LifecycleWorkflowRunSucceeded | "" | succeeded |  |
| running | LifecycleWorkflowRunPartiallyFailed | "" | partially_failed |  |
| running | LifecycleWorkflowRunFailed | "" | failed |  |
| queued | LifecycleWorkflowRunCanceled | "" | canceled |  |
| running | LifecycleWorkflowRunCanceled | "" | canceled |  |

## Authorization boundary

Authorization semantics are enforced by the application and its tests. This specification records API authentication, but intentionally defines no policy DSL. A separate work item will evaluate Cedar before any policy language is adopted.
