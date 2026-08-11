namespace IdMagic.Contract {

@doc("Application context が所有する published stub。ProvisioningConnection は Application 1 件に対し最大 1 件。")
model Application {
  id: string;
  tenant_id: string;
}


@doc("Application context が所有する定義の published language stub。")
enum AssignmentSubjectType {
  user: "user",
  group: "group",
}


@doc("Application context が所有する published stub。scope=assigned_only の対象範囲解決に用いる。")
model ApplicationAssignment {
  application_id: string;
  subject_type: AssignmentSubjectType;
  subject_id: string;
}


@doc("指定 application_id に有効な ProvisioningConnection が存在しない、または別テナントに属する (別テナントは未存在として扱う)。")
model ProvisioningConnectionNotFoundError {

}


@doc("指定 Application に既に ProvisioningConnection が存在する (1 Application 1 connection の一意制約)。")
model ProvisioningConnectionAlreadyExistsError {

}


@doc("指定 delivery_id の ProvisioningDelivery が存在しない、または指定 connection に属さない。")
model ProvisioningDeliveryNotFoundError {

}


@doc("dead_letter 以外の状態の ProvisioningDelivery に対して RetryProvisioningDelivery が呼ばれた。")
model ProvisioningDeliveryNotRetryableError {

}


@doc("ProvisionOnDemand が指定した subject が connection の scope (assigned_only の場合は割当) に含まれない。")
model ProvisioningSubjectNotInScopeError {

}


@doc("下流 SCIM service provider への認証方式。")
enum ProvisioningAuthMethod {
  bearer_token: "bearer_token",
  oauth2_client_credentials: "oauth2_client_credentials",
}


@doc("配送対象 subject の決め方。assigned_only は ApplicationAssignment を参照し、all_users はテナント全 User を対象にする上級オプション。")
enum ProvisioningScope {
  assigned_only: "assigned_only",
  all_users: "all_users",
}


@doc("管理者が切り替える有効状態。ApplicationStatus と同様、削除は状態を持たず行を除去する (履歴は ProvisioningConnectionDeleted イベントに残る)。")
enum ProvisioningConnectionStatus {
  active: "active",
  disabled: "disabled",
}


@doc("配送エンジンから見た connection の健全性。quarantined は ResumeProvisioningConnection まで配送を停止する。")
enum ProvisioningHealth {
  ok: "ok",
  degraded: "degraded",
  quarantined: "quarantined",
}


@doc("push_groups 有効時に対象グループをどう選ぶか。")
enum ProvisioningGroupSelection {
  assigned_groups: "assigned_groups",
  explicit: "explicit",
}


@doc("AttributeMappingRule の値の供給元。expression (関数評価) は初期版の対象外とし、直参照と固定値のみ許す。")
enum AttributeSourceKind {
  attribute: "attribute",
  constant: "constant",
}


@doc("AttributeMappingRule をいつ適用するか。create_only は下流での手動変更を尊重したい項目に使う。")
enum AttributeApplyOn {
  create_and_update: "create_and_update",
  create_only: "create_only",
}


@doc("deprovision trigger を下流へどう翻訳するか。")
enum ProvisioningDeprovisionAction {
  deactivate: "deactivate",
  delete: "delete",
  none: "none",
}


@doc("ProvisioningDelivery / RemoteResourceLink が指す内部集約の種別。")
enum ProvisioningSourceType {
  user: "user",
  group: "group",
}


@doc("ProvisioningDelivery が下流へ適用する操作。")
enum ProvisioningOperation {
  create: "create",
  update: "update",
  deactivate: "deactivate",
  delete: "delete",
  membership_add: "membership_add",
  membership_remove: "membership_remove",
}


@doc("ProvisioningDelivery.status の値集合。in_flight は dispatcher が Jobs.Job を関連付けてから\n終端に至るまでを表し、Jobs レベルの attempt retry 中も in_flight のまま保持する\n(WorkflowRunLifecycle と同じく、下位の retry loop をこの層の状態には反映しない)。\nfailed という非終端の中間状態は持たない — dead_letter は Job が max_attempts を\n使い切った時点で一度だけ到達する終端状態。")
enum ProvisioningDeliveryStatus {
  pending: "pending",
  in_flight: "in_flight",
  succeeded: "succeeded",
  dead_letter: "dead_letter",
}


@doc("connection が下流へ行ってよい操作のトグル (Okta 準拠)。sync_password は out of scope。")
model ProvisioningFeatureFlags {
  create_users: boolean = true;
  update_users: boolean = true;
  deactivate_users: boolean = true;
  delete_users: boolean = false;
  push_groups: boolean = false;
}


@doc("TestProvisioningConnection の discovery (下流 /ServiceProviderConfig) がキャッシュする read-only capability。PATCH 非対応先は PUT にフォールバックする判断に使う。")
model ProvisioningCapabilities {
  supports_patch: boolean;
  supports_bulk: boolean;
  supports_filter: boolean;
  supports_etag: boolean;
  supports_sort: boolean;
  discovered_at: utcDateTime;
}


@doc("Application editor 向けに credential を写像した非機密メタデータ。平文・hash を露出しない (ApplicationClientSecretCredentialMetadata と同型)。")
model ProvisioningConnectionCredentialMetadata {
  credential_id: unknown;
  auth_method: ProvisioningAuthMethod;
  created_at: utcDateTime;
  rotated_at?: utcDateTime;
}


@doc("RegisterProvisioningConnection / UpdateProvisioningConnection が受け取る書き込み専用 credential 入力。応答・監査ログ・エラーに値をそのまま含めない。")
model ProvisioningCredentialInput {
  auth_method: ProvisioningAuthMethod;
  @doc("auth_method=bearer_token のとき必須。")
  bearer_token?: string;
  @doc("auth_method=oauth2_client_credentials のとき必須。")
  oauth2_token_url?: url;
  oauth2_client_id?: string;
  oauth2_client_secret?: string;
  oauth2_scope?: string;
}


@doc("push_groups 有効時の対象グループ選択と表示名の供給元。")
model GroupPushConfig {
  selection: ProvisioningGroupSelection = ProvisioningGroupSelection.assigned_groups;
  @doc("selection=explicit のとき使う明示的な Group.id 一覧。")
  explicit_group_ids?: string[];
  @doc("下流 displayName に写す idmagic Group 属性キー。")
  display_name_source: string = "name";
}


@doc("1 出力属性の宣言的写像。target_path は SCIM 属性パス (単純・複合・多値フィルタ・enterprise 拡張を許す)。")
model AttributeMappingRule {
  @minLength(1)
  target_path: string;
  source_kind: AttributeSourceKind;
  @doc("source_kind=attribute のとき必須。idmagic 側の属性キー。")
  source_key?: string;
  @doc("source_kind=constant のとき必須。")
  constant_value?: unknown;
  apply_on: AttributeApplyOn = AttributeApplyOn.create_and_update;
  @doc("true のとき値が解決不能なら配送を fail-closed で失敗させる。")
  required: boolean = false;
  @doc("source が空のとき使う既定値。")
  default_value?: unknown;
}


@doc("409 (既存衝突) 時に下流の既存 resource を養子縁組するための比較属性。externalId 相関 (idmagic の id) が主で、このルールは externalId 未設定の既存 resource との衝突解決に使う二次手段。")
model MatchingRule {
  conflict_match_attribute: string = "userName";
}


@doc("deprovision trigger を下流アクションへ翻訳する規則。user disable → deactivate は固定 (設定不可)。\ngroup membership 削除 → PATCH members remove も固定。既定は破壊性が低い側に倒す。")
model DeprovisionPolicy {
  @doc("Application からの unassign trigger。")
  on_unassign: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.deactivate;
  @doc("user soft-delete/delete trigger。delete を選ぶと grace_period_days 後に purge する。")
  on_delete: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.deactivate;
  @doc("group 削除/unassign trigger。group に deactivate 相当は無いため delete/none のみ許す。")
  on_group_deleted_or_unassigned: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.none;
  @minValue(0)
  grace_period_days: int64 = 0;
  @doc("1 回の sync でこの件数を超える deactivate/delete が生じたら実行せず quarantine する。")
  accidental_deletion_count_threshold?: int64;
  @minValue(1)
  @maxValue(100)
  accidental_deletion_percentage_threshold?: int64;
}


@doc("Application 1 件に対し最大 1 件だけ存在する outbound provisioning 設定。真実源は\nidmagic 側の User/Group aggregate、下流は mirror。scope は既存 ApplicationAssignment を再利用し、\n専用の割当 model は持たない。identity を application_id とすることで 1 Application 1 connection の\n一意制約を構造的に表す。")
model ProvisioningConnection {
  application_id: unknown;
  tenant_id: string;
  status: ProvisioningConnectionStatus = ProvisioningConnectionStatus.active;
  @doc("下流 SCIM v2 base URL。https 必須、内部/リンクローカル IP・任意 redirect を拒否する。")
  base_url: url;
  credential: ProvisioningConnectionCredentialMetadata;
  @doc("TestProvisioningConnection の discovery でキャッシュする。未実行の間は未設定。")
  capabilities?: ProvisioningCapabilities;
  feature_flags: ProvisioningFeatureFlags;
  scope: ProvisioningScope = ProvisioningScope.assigned_only;
  @doc("feature_flags.push_groups が有効なときのみ意味を持つ。")
  group_push?: GroupPushConfig;
  @doc("登録時に SCIM core User の既定マッピングを seed する。")
  attribute_mappings: AttributeMappingRule[];
  matching: MatchingRule;
  deprovision_policy: DeprovisionPolicy;
  @minValue(1)
  rate_limit_per_minute: int64 = 60;
  @minValue(1)
  max_attempts: int64 = 8;
  @doc("失敗・quarantine の通知先。")
  notification_email?: string;
  @minValue(1)
  quarantine_after_consecutive_failures: int64 = 10;
  health: ProvisioningHealth = ProvisioningHealth.ok;
  @doc("quarantine 判定用の内部カウンタ。配送成功でゼロへ戻る。")
  consecutive_failure_count: int64 = 0;
  last_full_sync_at?: utcDateTime;
  quarantined_at?: utcDateTime;
  quarantine_reason?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("idmagic の User/Group と下流 resource の相関の durable な記録。409/404 を冪等に解決するために保持する。")
model RemoteResourceLink {
  connection_id: unknown;
  tenant_id: string;
  source_type: ProvisioningSourceType;
  @doc("idmagic 側の User.id または Group.id。")
  source_id: string;
  @doc("下流 SCIM resource の id。")
  remote_id: string;
  @doc("下流へ送出した externalId (既定は idmagic の source_id)。")
  external_id: string;
  @doc("capabilities.supports_etag が true のときだけ意味を持つ。")
  etag?: string;
  @doc("最後に反映した source_version。out-of-order な配送をここで解決する。")
  last_synced_version: int64;
  updated_at: utcDateTime;
}


@doc("内部 lifecycle event 1 件分の配送単位。idempotency key\n(tenant_id, connection_id, source_type, source_id, source_version) は一意制約であり、\n同一 key の重複 enqueue は既存行への no-op とする。job_id は dispatcher が Jobs\nへ enqueue した後に関連付ける (未 enqueue の間は未設定)。")
model ProvisioningDelivery {
  id: unknown;
  tenant_id: string;
  connection_id: unknown;
  source_type: ProvisioningSourceType;
  source_id: string;
  @doc("same-Tx capture が発行する単調増加カウンタ。out-of-order 配送の解決に使う。")
  source_version: int64;
  operation: ProvisioningOperation;
  status: ProvisioningDeliveryStatus = ProvisioningDeliveryStatus.pending;
  @doc("dispatcher が enqueue 後に関連付ける Jobs.JobRef.id。")
  job_id?: string;
  @doc("直近の失敗の sanitised error。secret や属性値は含めない。")
  last_error?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("管理者が Application へ ProvisioningConnection を新規登録した。")
model ProvisioningConnectionRegistered {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("管理者が ProvisioningConnection の設定 (credential 以外) を更新した。")
model ProvisioningConnectionUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("管理者が ProvisioningConnection.status を disabled にした。配送を新規に生成しなくなる。")
model ProvisioningConnectionDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("管理者が ProvisioningConnection を削除した (ApplicationStatus と同じく行そのものを除去する hard delete)。")
model ProvisioningConnectionDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("管理者が connection の credential を更新した。")
model ProvisioningCredentialRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  credentialId: unknown;
}


@doc("dispatcher が ProvisioningDelivery に Jobs.Job を関連付け配送を開始した (LifecycleWorkflowRunStarted と同型)。")
model ProvisioningDeliveryStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  jobId: string;
}


@doc("user の create/update が下流へ反映され配送が succeeded に確定した。")
model UserProvisioned {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  userId: string;
  remoteId: string;
}


@doc("user の deactivate/delete が下流へ反映され配送が succeeded に確定した。")
model UserDeprovisioned {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  userId: string;
  action: ProvisioningDeprovisionAction;
}


@doc("user 向け配送が max_attempts を使い切り dead_letter に確定した (attempt ごとには発行しない)。group 向け配送の dead-letter でも同一イベントを用いる。")
model UserProvisioningFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  sourceType: ProvisioningSourceType;
  sourceId: string;
  error: string;
}


@doc("group の create/update/deactivate/delete が下流へ反映され配送が succeeded に確定した。")
model GroupPushed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  groupId: string;
  remoteId: string;
}


@doc("group membership の変更が下流へ PATCH members として反映され配送が succeeded に確定した。")
model GroupMembershipPushed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  deliveryId: string;
  groupId: string;
}


@doc("連続失敗または誤削除ガード超過により ProvisioningConnection.health が quarantined になった。以後の配送を生成しない。")
model ConnectionQuarantined {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  reason: string;
  consecutiveFailures: int64;
}


@doc("管理者が ResumeProvisioningConnection で quarantine を解除した。quarantine を解除する経路が無いと復旧不能になるため、イベント一覧へ追加した。")
model ProvisioningConnectionQuarantineCleared {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("StartFullResync が対象にした scope 内 subject 全件の収束が完了した。")
model FullResyncCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  totalSubjects: int64;
  succeededCount: int64;
  failedCount: int64;
}

}
