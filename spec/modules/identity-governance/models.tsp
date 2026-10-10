namespace IdMagic.Contract {

@doc("User の運用上の状態。IdManagement.UserStatus を公開契約として写したもので、トリガーの `from_status` と `to_status` の評価だけに使う。値の唯一の情報源は IdManagement が管理する。")
enum UserStatus {
  Active: "active",
  Disabled: "disabled",
  PendingDeletion: "pending_deletion",
  Deleted: "deleted",
  Locked: "locked",
  Staged: "staged",
  Suspended: "suspended",
}


@doc("次のサインインでユーザーに強制するアクション。IdManagement.RequiredAction を公開契約として写したもので、`set_required_action` または `clear_required_action` のアクションを指定するためだけに使う。")
enum RequiredAction {
  UpdatePassword: "update_password",
  VerifyEmail: "verify_email",
  ConfigureTotp: "configure_totp",
  UpdateProfile: "update_profile",
  TermsAndConditions: "terms_and_conditions",
}


@doc("割り当ての表示範囲。Application モジュールが所有する定義を公開契約として写したもので、`assign_application` のアクションの `visibility` だけに使う。")
enum AssignmentVisibility {
  visible: "visible",
  hidden: "hidden",
}


@doc("LifecycleWorkflow のライフサイクル。`archived` は、削除した定義を内部に残して実行の履歴の参照整合性を保つための実装上の状態で、管理コンソールにも通常の API にも公開しない。")
enum LifecycleWorkflowStatus {
  draft: "draft",
  enabled: "enabled",
  disabled: "disabled",
  archived: "archived",
}


@doc("WorkflowTrigger が起動する条件の種類。最初のバージョンには、User の変更を起点とする三つだけがある。日付、cron、待機のステップは範囲外である。")
enum WorkflowTriggerKind {
  user_created: "user_created",
  user_attributes_changed: "user_attributes_changed",
  user_status_changed: "user_status_changed",
}


@doc("WorkflowFilter で使える比較演算子。`eq` と `not_eq` は完全一致と不一致、`in` は値の集合のいずれかとの一致、`exists` は値がそもそもあるかどうかの判定である。")
enum WorkflowFilterOperator {
  eq: "eq",
  not_eq: "not_eq",
  in: "in",
  exists: "exists",
}


@doc("WorkflowAction の種類。どれも望ましい状態を示すので冪等であり、状態がすでに成り立っていれば no_op になる。")
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


@doc("WorkflowRunLifecycle の状態。何もしなかった場合も成功とみなす。")
enum WorkflowRunStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  partially_failed: "partially_failed",
  failed: "failed",
  canceled: "canceled",
}


@doc("一つの WorkflowStep の結果。`pending` は、何も試みる前の初期値である。")
enum WorkflowStepOutcome {
  pending: "pending",
  changed: "changed",
  no_op: "no_op",
  failed: "failed",
  canceled: "canceled",
}


@doc("UpdateLifecycleWorkflow が、`expected_revision` が現在の `LifecycleWorkflow.current_revision` と一致しないことを見つけ、更新の消失を拒否した。呼び出し元は最新のリビジョンを読み直して再試行する。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workflow_revision_conflict` である。")
model WorkflowRevisionConflictError is ProblemDetails;


@doc("求めた LifecycleWorkflow の `name` は、テナントの削除されていないワークフローがすでに使っている。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:workflow_name_conflict` である。")
model WorkflowNameConflictError is ProblemDetails;


@doc("`WorkflowTrigger.filters` の一項。保存できるのは、TenantUserAttributeSchema に存在し、型が一致するフィールドだけである。個人情報に対する自由入力の正規表現と未知のフィールドは、どちらも保存時に拒否する。複数の絞り込み条件は AND で結ぶ。")
model WorkflowFilter {
  @doc("`TenantUserAttributeSchema.key`、または User の組み込みの属性のキー。")
  @minLength(1)
  field: string;
  operator: WorkflowFilterOperator;
  @doc("型が `field` の AttributeType（文字列、数値、真偽値、日付の文字列、文字列の配列）に従うので、任意の JSON 値とする。`in` 演算子はその型の配列をとり、`exists` は値をとらない。この依存は一つの TypeSpec の型では表せないので、契約は値を制約なしとして記録し、サーバーがテナントの属性のスキーマに照らして検査する。")
  value?: unknown;
}


@doc("LifecycleWorkflowRevision が固定するトリガーの定義。種類に応じて `watched_attributes`、または `from_status` と `to_status` を使う。`filters` は AND で結ぶ 0 から 20 の項からなる。")
model WorkflowTriggerDef {
  kind: WorkflowTriggerKind;
  @doc("`kind` が `user_attributes_changed` のとき必須である。載せたフィールドのどれかが実際に変わったときだけ起動し、値の変わらないフィールドがリクエストに含まれているだけでは起動しない。")
  watched_attributes?: string[];
  @doc("`kind` が `user_status_changed` のとき必須である。許された UserLifecycle の遷移の `from` である。")
  from_status?: UserStatus;
  @doc("`kind` が `user_status_changed` のとき必須である。許された UserLifecycle の遷移の `to` である。")
  to_status?: UserStatus;
  filters: WorkflowFilter[];
}


@doc("LifecycleWorkflowRevision が固定する一つのアクションの定義。どのフィールドを使うかは種類によって決まり、参照先の存在は作成と更新のとき、および有効化のときの両方で検査する。")
model WorkflowActionDef {
  kind: WorkflowActionKind;
  @doc("`kind` が `add_group_member` または `remove_group_member` のとき必須である。動的グループを指定すると、保存時の検証エラーになる。")
  group_id?: string;
  @doc("`kind` が `assign_application` または `unassign_application` のとき必須である。")
  application_id?: string;
  @doc("`kind` が `assign_application` のとき任意で、省くと `visible` になる。Application モジュールが所有する定義を公開契約として写したもの。")
  visibility?: AssignmentVisibility;
  @doc("`kind` が `set_required_action` または `clear_required_action` のとき必須である。")
  required_action?: RequiredAction;
  @doc("`kind` が `enable_user` または `disable_user` のときに記録する、任意の理由。")
  reason?: string;
  @doc("`kind` が `send_email` のとき必須である。ワークフローの作成者が通知に付けるラベル。文面は Tenancy の通知のテンプレートのカタログの LifecycleWorkflowNotification から解決し、この値を置換変数として本文に渡す。ワークフローごとに別のテンプレートを割り当てることは、このモジュールの範囲外である。")
  template_key?: string;
}


@doc("LifecycleWorkflow の定義の変わらないスナップショット。意味を変える更新のたびにリビジョンを 1 増やし、過去のリビジョンは変わらないまま残る。新しい WorkflowRun を作るのは、有効なリビジョンだけである。")
model LifecycleWorkflowRevision {
  workflow_id: string;
  tenant_id: string;
  revision: int64;
  trigger: WorkflowTriggerDef;
  actions: WorkflowActionDef[];
  created_at: utcDateTime;
}


@doc("ワークフローの定義の集約で、`name` はテナントの中で一意である。作成するとリビジョン 1 を下書きとして作る。トリガーが評価するのは有効なリビジョンだけで、意味を変える更新は `current_revision` を増やすが、`status` と `enabled_revision` は変えない。管理者は状態にかかわらず削除でき、削除した定義は一覧、読み取り、名前の一意性の検査から外れる。既存の実行と監査イベントは、定義とは独立して残す。")
model LifecycleWorkflow {
  @doc("生成した識別子（UUID v4）。変わらない。")
  @minLength(1)
  @maxLength(64)
  id: string;
  @doc("属するテナント。テナントをまたぐ参照は禁止する。")
  tenant_id: string = "default";
  @doc("表示名。テナントの中で一意である。")
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  status: LifecycleWorkflowStatus = LifecycleWorkflowStatus.draft;
  @doc("直近に作成したリビジョンの番号。")
  current_revision: int64;
  @doc("トリガーが現在評価するリビジョン。ワークフローが下書き、無効、またはアーカイブ済みの間は設定されないことがある。")
  enabled_revision?: int64;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("トリガーの起動の要約で、WorkflowRun を作成したときに固定する。履歴には変わったフィールドの名前と各述語が成り立ったかどうかだけを残し、個人情報を守るため、属性値そのものは残さない。")
model WorkflowRunTriggerSnapshot {
  kind: WorkflowTriggerKind;
  @doc("起動した User の変更のドメインイベントの id。同じ発生の再配送は同じ実行に収束するので、重複を除く。")
  source_occurrence_id: string;
  @doc("`kind` が `user_attributes_changed` のときの、実際に変わったフィールドの名前。値は含まない。")
  changed_fields: string[];
  from_status?: UserStatus;
  to_status?: UserStatus;
}


@doc("WorkflowTrigger の一回の起動が生む作業の単位。workflow_id、リビジョン、トリガーのスナップショット、target_user_id、展開したアクションの一覧はすべて作成時に固定するので、後で定義を変えても既存の実行の意味は変わらない。ディスパッチャーはキューへ入れた後に `job_id` を付ける。アクションを実行する直前に、対象のユーザーとリソースをテナントの中で検査し直し、同じ target_user_id に対する実行は `triggered_at` の順に直列化する。`(tenant_id, workflow_id, revision, source_occurrence_id, target_user_id)` が一意性の制約である。")
model WorkflowRun {
  @doc("生成した識別子（UUID v4）。")
  id: string;
  tenant_id: string;
  workflow_id: string;
  revision: int64;
  trigger: WorkflowRunTriggerSnapshot;
  target_user_id: string;
  @doc("リビジョンから展開し、ここで固定したアクションの一覧。後で定義を変えても、この実行の意味は変わらない。")
  actions: WorkflowActionDef[];
  status: WorkflowRunStatus;
  @doc("ディスパッチャーがキューへ入れた後に付ける `Jobs.JobRef.id`。それまでは設定しない。")
  job_id?: string;
  triggered_at: utcDateTime;
  started_at?: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("WorkflowRun の中の一つのアクションの実行の記録。定義の順での位置を保持し、結果と無害化したエラーコードをチェックポイントとして記録する。属性値、メールの本文、シークレットは残さない。")
model WorkflowStep {
  run_id: string;
  tenant_id: string;
  @doc("`WorkflowRun.actions` の中での、0 から数えた位置。")
  step_index: int64;
  action_kind: WorkflowActionKind;
  outcome: WorkflowStepOutcome = WorkflowStepOutcome.pending;
  @doc("無害化したエラーコード。属性値もメールの本文も含まない。")
  error_code?: string;
  attempted_at?: utcDateTime;
}


@doc("`send_email` のアクションの配送の冪等キー。ある `(run_id, step_index)` についてメッセージをちょうど一通送ったことを記録する。")
model LifecycleNotificationDelivery {
  run_id: string;
  step_index: int64;
  tenant_id: string;
  template_key: string;
  delivered_at: utcDateTime;
}


@doc("ドライランの対象のユーザーを指定する入力。使うリビジョンは `enabled_revision`、指定がなければ `current_revision` である。")
model LifecycleWorkflowDryRunRequest {
  target_user_id: string;
}


@doc("ドライランの一つのアクションに対する判定。対象の User の現在の状態（グループのメンバーシップ、アプリケーションの割り当て、必須のアクション、状態、メールの確認状態）を読み、実際のアクションの実行器が下すのと同じ判定（would_change、no_op、blocked）を、副作用なしに返す。属性値もメールアドレスも含まない。")
model LifecycleWorkflowDryRunStepResult {
  action_kind: WorkflowActionKind;
  @doc("would_change、no_op、blocked のいずれか。")
  would_change: string;
  @doc("blocked になった理由。resource_not_found、dependency_unavailable、notification_unavailable、trigger_not_matched など。would_change と no_op では省く。")
  reason?: string;
}


@doc("ドライランの結果。何も作成も変更もしない。WorkflowRun、Job、メンバーシップ、割り当て、必須のアクション、状態、メールのどれも変わらない。`evaluated_at` とリビジョンを含み、実際の実行についての保証としては示さない。")
model LifecycleWorkflowDryRunResponse {
  workflow_id: string;
  revision: int64;
  target_user_id: string;
  evaluated_at: utcDateTime;
  steps: LifecycleWorkflowDryRunStepResult[];
}


@doc("ワークフローを作成するリクエスト。作成するとリビジョン 1 を下書きとして作る。")
model LifecycleWorkflowCreateRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  trigger: WorkflowTriggerDef;
  actions: WorkflowActionDef[];
}


@doc("ワークフローを更新するリクエスト。`expected_revision` が現在の `current_revision` と一致しないときは WorkflowRevisionConflictError で拒否するので、更新の消失は起こらない。")
model LifecycleWorkflowUpdateRequest {
  expected_revision: int64;
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  trigger?: WorkflowTriggerDef;
  actions?: WorkflowActionDef[];
}


@doc("`current_revision` の内容を含む、ワークフローの定義の表現。")
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


@doc("ワークフローの一覧のレスポンス。")
model LifecycleWorkflowListResponse {
  workflows: LifecycleWorkflowResponse[];
}


@doc("実行の詳細の中のステップの表現。シークレット、メールの本文、属性値は含まない。")
model WorkflowStepResponse {
  step_index: int64;
  action_kind: WorkflowActionKind;
  outcome: WorkflowStepOutcome;
  error_code?: string;
  attempted_at?: utcDateTime;
}


@doc("実行の詳細のレスポンス。トリガー、固定したリビジョン、対象のユーザー、Job の試行と、各ステップの結果、エラーコード、タイムスタンプを示す。シークレット、メールの本文、属性値は含まない。")
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


@doc("ワークフローの実行の一覧のレスポンス。ページングはしない。エンドポイントはカーソルのパラメーターをとらず、サーバー側の上限までの直近の実行を返す。以前は、何も生成せず送り返すパラメーターもない `next_after` のカーソルを宣言しており、エンドポイントが行わないページングを約束していた。")
model WorkflowRunListResponse {
  runs: WorkflowRunResponse[];
}


@doc("管理者が、LifecycleWorkflow をリビジョン 1 の下書きとして作成した。")
model LifecycleWorkflowCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("管理者が LifecycleWorkflow を更新した。トリガーまたはアクションが変わってリビジョンが増えたときは、`newRevision` を含む。")
model LifecycleWorkflowUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
  newRevision?: int64;
}


@doc("管理者が `current_revision` を検証に通し、`enabled_revision` として有効にした。")
model LifecycleWorkflowEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
  revision: int64;
}


@doc("管理者が LifecycleWorkflow を無効にした。新しい実行は起動せず、開始していないキュー上の実行は取り消す。")
model LifecycleWorkflowDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("管理者が LifecycleWorkflow を削除した。それまでの状態にかかわらず発行し、監査証跡として残す。")
model LifecycleWorkflowDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  workflowId: string;
}


@doc("ディスパッチャーが WorkflowRun をワーカーへ渡し、最初のステップが始まった。")
model LifecycleWorkflowRunStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("WorkflowRun のすべてのステップが changed または no_op で終わった。")
model LifecycleWorkflowRunSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("WorkflowRun が試行の上限に達し、成功と失敗が混在したまま終わった。")
model LifecycleWorkflowRunPartiallyFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("WorkflowRun が試行の上限に達し、一つも成功したステップがないまま終わった。")
model LifecycleWorkflowRunFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("無効化によって、開始していないキュー上の実行が止まったか、実行中の実行が現在のステップのチェックポイントの後で止まった。")
model LifecycleWorkflowRunCanceled {
  occurredAt: utcDateTime;
  tenantId: string;
  workflowId: string;
  runId: string;
  targetUserId: string;
}


@doc("一つの WorkflowStep が失敗の結果で確定した。監査の検索で見つけられるように、workflow_id、run_id、step_id、error_code を残す。")
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
