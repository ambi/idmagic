namespace IdMagic.Contract {

@doc("User の運用状態。IdManagement.UserStatus の published language stub。trigger の from_status/to_status 評価にのみ用いる。値の唯一の真実は IdManagement が所有する。")
enum UserStatus {
  Active: "Active",
  Disabled: "Disabled",
  PendingDeletion: "PendingDeletion",
  Deleted: "Deleted",
  Locked: "Locked",
  Staged: "Staged",
  Suspended: "Suspended",
}


@doc("次回ログイン時にユーザへ強制するアクション。IdManagement.RequiredAction の published language stub。set_required_action/clear_required_action の action 定義参照にのみ用いる。")
enum RequiredAction {
  UpdatePassword: "UpdatePassword",
  VerifyEmail: "VerifyEmail",
  ConfigureTotp: "ConfigureTotp",
  UpdateProfile: "UpdateProfile",
  TermsAndConditions: "TermsAndConditions",
}


@doc("割当の可視性。Application context が所有する定義の published language stub。assign_application action の visibility 指定にのみ用いる。")
enum AssignmentVisibility {
  visible: "visible",
  hidden: "hidden",
}


@doc("LifecycleWorkflow のライフサイクル。archived は削除済み定義を実行履歴の参照整合性のため内部だけで保持する実装状態で、管理画面と通常 API には公開しない。")
enum LifecycleWorkflowStatus {
  draft: "draft",
  enabled: "enabled",
  disabled: "disabled",
  archived: "archived",
}


@doc("WorkflowTrigger の発火条件種別。初期版は User mutation 起点の3種のみ (日付・cron・待機 step は対象外)。")
enum WorkflowTriggerKind {
  user_created: "user_created",
  user_attributes_changed: "user_attributes_changed",
  user_status_changed: "user_status_changed",
}


@doc("WorkflowFilter で使える比較演算子。eq/not_eq は完全一致・不一致、in は値集合のいずれか一致、exists は値の有無判定。")
enum WorkflowFilterOperator {
  eq: "eq",
  not_eq: "not_eq",
  in: "in",
  exists: "exists",
}


@doc("WorkflowAction の種別。いずれも desired-state で、既に目的状態なら no_op になる冪等操作。")
enum WorkflowActionKind {
  add_group_member: "add_group_member",
  remove_group_member: "remove_group_member",
  assign_application: "assign_application",
  unassign_application: "unassign_application",
  set_required_action: "set_required_action",
  clear_required_action: "clear_required_action",
  enable_user: "enable_user",
  disable_user: "disable_user",
  send_email: "send_email",
}


@doc("WorkflowRunLifecycle の状態。no-op は成功として扱う。")
enum WorkflowRunStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  partially_failed: "partially_failed",
  failed: "failed",
  canceled: "canceled",
}


@doc("WorkflowStep 1 件の実行結果。pending はまだ試行されていない初期値。")
enum WorkflowStepOutcome {
  pending: "pending",
  changed: "changed",
  no_op: "no_op",
  failed: "failed",
  canceled: "canceled",
}


@doc("UpdateLifecycleWorkflow が input.request.expected_revision と現在の\nLifecycleWorkflow.current_revision の不一致を検出した (lost update の拒否)。呼び出し側は\n最新 revision を再取得してから再試行する。")
model WorkflowRevisionConflictError {

}


@doc("WorkflowTrigger.filters の1項。TenantUserAttributeSchema に存在し型が一致する field\nだけを保存でき、PII の自由文字 regex や未知 field は保存時に拒否する。複数 filter は AND。")
model WorkflowFilter {
  @doc("TenantUserAttributeSchema.key または User の built-in attribute key。")
  @minLength(1)
  field: string;
  operator: WorkflowFilterOperator;
  @doc("operator=exists では未指定。field の AttributeType と一致する型付き値 (in は配列)。")
  value?: unknown;
}


@doc("LifecycleWorkflowRevision が固定する trigger 定義。kind に応じて watched_attributes\nまたは from_status/to_status を用いる。filters は 0〜20 件で AND。")
model WorkflowTriggerDef {
  kind: WorkflowTriggerKind;
  @doc("kind=user_attributes_changed のとき必須。列挙した field のいずれかが実際に変化した\n場合だけ発火する (request に含まれただけで値が同じ場合は発火しない)。")
  watched_attributes?: string[];
  @doc("kind=user_status_changed のとき必須。許可された UserLifecycle 遷移の from。")
  from_status?: UserStatus;
  @doc("kind=user_status_changed のとき必須。許可された UserLifecycle 遷移の to。")
  to_status?: UserStatus;
  filters: WorkflowFilter[];
}


@doc("LifecycleWorkflowRevision が固定する action 定義 1 件。kind ごとに使うフィールドが異なり、\ncreate/update と enable の双方で参照先の存在を検証する。")
model WorkflowActionDef {
  kind: WorkflowActionKind;
  @doc("kind=add_group_member/remove_group_member のとき必須。dynamic group を指定した場合は\n保存時に validation error。")
  group_id?: string;
  @doc("kind=assign_application/unassign_application のとき必須。")
  application_id?: string;
  @doc("kind=assign_application のとき任意 (省略時 visible)。Application context が所有する定義の published language stub。")
  visibility?: AssignmentVisibility;
  @doc("kind=set_required_action/clear_required_action のとき必須。")
  required_action?: RequiredAction;
  @doc("kind=enable_user/disable_user のとき任意の記録用理由。")
  reason?: string;
  @doc("kind=send_email のとき必須。workflow 作成者が付ける通知の識別ラベル。送信文面は Tenancy の通知テンプレートカタログの LifecycleWorkflowNotification から解決し、この値は本文の差し込み変数として渡す。workflow ごとに別テンプレートを割り当てる機構は本 context の対象外。")
  template_key?: string;
}


@doc("LifecycleWorkflow の不変な definition 断面。意味の変わる update ごとに revision を 1 増やし、\n過去 revision は immutable に残す。enable された revision だけが新規 WorkflowRun を生成する。")
model LifecycleWorkflowRevision {
  workflow_id: string;
  tenant_id: string;
  revision: int64;
  trigger: WorkflowTriggerDef;
  actions: WorkflowActionDef[];
  created_at: utcDateTime;
}


@doc("テナント内で name が一意な workflow 定義集約。create は revision 1 の draft を作る。\nenable した revision だけが trigger 評価対象になり、意味の変わる update は current_revision を\n増やすだけで status/enabled_revision は変えない。管理者は状態に関係なく削除でき、削除済み定義は\n一覧・取得・名前重複判定から除外する。既存の run と監査イベントは定義とは独立して保持する。")
model LifecycleWorkflow {
  @doc("生成識別子。UUID v4。不変。")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("所属テナント ID。cross-tenant 参照は禁止。")
  tenant_id: string = "default";
  @doc("テナント内で一意な表示名。")
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  status: LifecycleWorkflowStatus = LifecycleWorkflowStatus.draft;
  @doc("直近に作成した revision 番号。")
  current_revision: int64;
  @doc("現在 trigger 評価対象になっている revision。draft/disabled/archived では未設定のことがある。")
  enabled_revision?: int64;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("WorkflowRun 作成時に固定する trigger 発火の要約。履歴には changed field 名と predicate の\n成否だけを残し、属性値そのものは保持しない (PII 保護)。")
model WorkflowRunTriggerSnapshot {
  kind: WorkflowTriggerKind;
  @doc("発火元の User mutation domain event ID。同一 occurrence の再配送は同じ run に収束する (dedup)。")
  source_occurrence_id: string;
  @doc("kind=user_attributes_changed のとき、実際に変化した field 名一覧 (値は含まない)。")
  changed_fields: string[];
  from_status?: UserStatus;
  to_status?: UserStatus;
}


@doc("WorkflowTrigger の発火 1 回から生成される実行単位。作成時の workflow_id、revision、\ntrigger snapshot、target_user_id、展開済み action list を固定し、後から definition を変更しても\n既存 run の意味は変わらない。job_id は dispatcher が enqueue 後に関連付ける。action 実行直前に\ntenant 内の対象 user と resource を再検証し、同一 target_user_id の run は triggered_at 順に\n直列化する。\n(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id) が一意制約。")
model WorkflowRun {
  @doc("生成識別子。UUID v4。")
  id: string;
  tenant_id: string;
  workflow_id: string;
  revision: int64;
  trigger: WorkflowRunTriggerSnapshot;
  target_user_id: string;
  @doc("revision から展開・固定した action list。definition が後から変わっても本 run の意味は変わらない。")
  actions: WorkflowActionDef[];
  status: WorkflowRunStatus;
  @doc("dispatcher が enqueue 後に関連付ける Jobs.JobRef.id。未 enqueue の間は未設定。")
  job_id?: string;
  triggered_at: utcDateTime;
  started_at?: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("WorkflowRun 内の action 1 件分の実行記録。定義順の index を持ち、outcome と sanitised\nerror code を checkpoint する。attribute value / email 本文 / secret は保持しない。")
model WorkflowStep {
  run_id: string;
  tenant_id: string;
  @doc("WorkflowRun.actions 内の 0 始まり順序。")
  step_index: int64;
  action_kind: WorkflowActionKind;
  outcome: WorkflowStepOutcome = WorkflowStepOutcome.pending;
  @doc("sanitised error code。attribute 値や email 本文は含めない。")
  error_code?: string;
  attempted_at?: utcDateTime;
}


@doc("send_email action の delivery 冪等キー。(run_id, step_index) 単位で一回分だけ送信したことを記録する。")
model LifecycleNotificationDelivery {
  run_id: string;
  step_index: int64;
  tenant_id: string;
  template_key: string;
  delivered_at: utcDateTime;
}


@doc("dry-run 対象 user_id の入力。対象 revision は enabled_revision (未指定なら current_revision)。")
model LifecycleWorkflowDryRunRequest {
  target_user_id: string;
}


@doc("dry-run の action 1 件分の判定結果。対象 User の group membership、application\nassignment、required action、status、email 検証状態という現在の状態を実際に読み、本実行の\naction executor と同じ判定 (would_change / no_op / blocked) を副作用なしで返す。属性値や\nemail address は含まない。")
model LifecycleWorkflowDryRunStepResult {
  action_kind: WorkflowActionKind;
  @doc("would_change / no_op / blocked のいずれか。")
  would_change: string;
  @doc("blocked の理由コード (resource_not_found / dependency_unavailable /\nnotification_unavailable / trigger_not_matched 等)。would_change / no_op では省略する。")
  reason?: string;
}


@doc("dry-run 結果。WorkflowRun / Job / membership / assignment / required action / status / email\nのいずれも作成・変更しない。evaluated_at と revision を含め、本実行の保証とは表示しない。")
model LifecycleWorkflowDryRunResponse {
  workflow_id: string;
  revision: int64;
  target_user_id: string;
  evaluated_at: utcDateTime;
  steps: LifecycleWorkflowDryRunStepResult[];
}


@doc("workflow 作成リクエスト。作成すると revision 1 の draft になる。")
model LifecycleWorkflowCreateRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  trigger: WorkflowTriggerDef;
  actions: WorkflowActionDef[];
}


@doc("workflow 更新リクエスト。expected_revision と現在の current_revision が一致しない場合は\nWorkflowRevisionConflictError で拒否する (lost update 防止)。")
model LifecycleWorkflowUpdateRequest {
  expected_revision: int64;
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  trigger?: WorkflowTriggerDef;
  actions?: WorkflowActionDef[];
}


@doc("workflow 定義の表現 (current_revision の内容を含む)。")
model LifecycleWorkflowResponse {
  id: string;
  name: string;
  description?: string;
  status: LifecycleWorkflowStatus;
  current_revision: int64;
  enabled_revision?: int64;
  trigger: WorkflowTriggerDef;
  actions: WorkflowActionDef[];
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("workflow 一覧レスポンス。")
model LifecycleWorkflowListResponse {
  workflows: LifecycleWorkflowResponse[];
}


@doc("run detail 内の step 表現。secret / email 本文 / attribute value は含めない。")
model WorkflowStepResponse {
  step_index: int64;
  action_kind: WorkflowActionKind;
  outcome: WorkflowStepOutcome;
  error_code?: string;
  attempted_at?: utcDateTime;
}


@doc("run detail レスポンス。trigger、固定 revision、target user、Job attempt、各 step の\noutcome/error_code/timestamp を表示し、secret / email 本文 / attribute value は含めない。")
model WorkflowRunResponse {
  id: string;
  workflow_id: string;
  revision: int64;
  trigger: WorkflowRunTriggerSnapshot;
  target_user_id: string;
  status: WorkflowRunStatus;
  job_id?: string;
  steps: WorkflowStepResponse[];
  triggered_at: utcDateTime;
  started_at?: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("workflow の run 一覧レスポンス。")
model WorkflowRunListResponse {
  runs: WorkflowRunResponse[];
  @doc("ページング継続用カーソル。")
  next_after?: utcDateTime;
}


@doc("管理者が LifecycleWorkflow を作成した (revision 1 の draft)。")
model LifecycleWorkflowCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("管理者が LifecycleWorkflow を更新した。trigger/actions が変わり revision が増えた場合は newRevision を含む。")
model LifecycleWorkflowUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
  newRevision?: int64;
}


@doc("管理者が current_revision を validation に成功させ enabled_revision として enable した。")
model LifecycleWorkflowEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
  revision: int64;
}


@doc("管理者が LifecycleWorkflow を disable した。新規 trigger を止め、未開始の queued run を canceled にする。")
model LifecycleWorkflowDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("管理者が LifecycleWorkflow を削除した。削除前の状態にかかわらず発行し、監査証跡として保持する。")
model LifecycleWorkflowDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("dispatcher が WorkflowRun を worker に渡し、最初の step 実行を開始した。")
model LifecycleWorkflowRunStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("WorkflowRun の全 step が changed または no_op で終端した。")
model LifecycleWorkflowRunSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("WorkflowRun が attempt 上限に達し、成功と失敗が混在したまま終端した。")
model LifecycleWorkflowRunPartiallyFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("WorkflowRun が attempt 上限に達し、成功した step が一つもないまま終端した。")
model LifecycleWorkflowRunFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("disable により未開始の queued run が、または実行中の run が現在 step の checkpoint 後に停止した。")
model LifecycleWorkflowRunCanceled {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("WorkflowStep 1 件が failed outcome で確定した。監査検索用に workflow_id/run_id/step_id/error_code を残す。")
model LifecycleWorkflowStepFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  stepIndex: int64;
  actionKind: WorkflowActionKind;
  errorCode: string;
}

}
