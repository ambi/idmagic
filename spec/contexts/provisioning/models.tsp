namespace IdMagic.Contract {

@doc("Application の Context が所有するものを公表した写し。ProvisioningConnection は Application ごとに一つまでである。")
model Application {
  id: string;
  tenant_id: string;
}


@doc("Application の Context が所有する定義を、公表された言語として写したもの。")
enum AssignmentSubjectType {
  user: "user",
  group: "group",
}


@doc("Application の Context が所有するものを公表した写し。`scope` が `assigned_only` のときに対象の集合を解決するために使う。")
model ApplicationAssignment {
  application_id: string;
  subject_type: AssignmentSubjectType;
  subject_id: string;
}


@doc("指定した `application_id` に有効な ProvisioningConnection が存在しないか、他のテナントに属している（存在しないものとして扱う）。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:provisioning_not_found` である。")
model ProvisioningConnectionNotFoundError is ProblemDetails;


@doc("Application にはすでに ProvisioningConnection がある。接続は Application ごとに一つまでである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:provisioning_conflict` である。")
model ProvisioningConnectionAlreadyExistsError is ProblemDetails;


@doc("指定した `task_id` の ProvisioningTask が存在しないか、指定した接続に属していない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:provisioning_not_found` である。")
model ProvisioningTaskNotFoundError is ProblemDetails;


@doc("状態が `dead_letter` でない ProvisioningTask に対して RetryProvisioningTask を呼んだ。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:provisioning_conflict` である。")
model ProvisioningTaskNotRetryableError is ProblemDetails;


@doc("ProvisionOnDemand が指定したサブジェクトが接続の範囲の外にある。`assigned_only` では、サブジェクトが割り当てられていないことを意味する。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:provisioning_conflict` である。")
model ProvisioningSubjectNotInScopeError is ProblemDetails;


@doc("下流の SCIM のサービスプロバイダーへの認証の方法。どちらの方法でも送信するリクエストは `Authorization: Bearer` を含み、方法はその値の取得元を決める。`bearer_token` は保存した資格情報をそのまま提示する。`oauth2_client_credentials` は、保存したクライアントシークレットを接続のトークンエンドポイントでアクセストークンと交換し（RFC 6749 の 4.4 節）、それを提示する。トークンは期限の少し前まで再利用する。")
enum ProvisioningAuthMethod {
  bearer_token: "bearer_token",
  oauth2_client_credentials: "oauth2_client_credentials",
}


@doc("プロビジョニングの対象のサブジェクトの選び方。`assigned_only` は ApplicationAssignment を読む。`all_users` はテナントのすべての User を対象にする高度なオプションである。")
enum ProvisioningScope {
  assigned_only: "assigned_only",
  all_users: "all_users",
}


@doc("管理者が切り替える有効の状態。ApplicationStatus と同じく、削除は状態ではない。行を削除し、履歴は ProvisioningConnectionDeleted のイベントに残る。")
enum ProvisioningConnectionStatus {
  active: "active",
  disabled: "disabled",
}


@doc("プロビジョニングのエンジンから見た接続の健全性。`quarantined` は、ResumeProvisioningConnection までプロビジョニングを止める。")
enum ProvisioningHealth {
  ok: "ok",
  degraded: "degraded",
  quarantined: "quarantined",
}


@doc("`push_groups` が有効な間の、対象のグループの選び方。")
enum ProvisioningGroupSelection {
  assigned_groups: "assigned_groups",
  explicit: "explicit",
}


@doc("下流の `displayName` へ射影する IdMagic 側の Group の属性。Group に値のない属性を選んだプロビジョニングタスクは名前を使う。表示名はフェイルクローズで判断するものではない。")
enum ProvisioningGroupDisplayNameSource {
  name: "name",
  description: "description",
  email: "email",
}


@doc("AttributeMappingRule の値の取得元。`expression`（関数の評価）は最初のバージョンの範囲外で、直接の参照と固定値だけを許す。")
enum AttributeSourceKind {
  attribute: "attribute",
  constant: "constant",
}


@doc("AttributeMappingRule を適用する時点。下流での手作業の変更を尊重すべきフィールドには `create_only` を使う。")
enum AttributeApplyOn {
  create_and_update: "create_and_update",
  create_only: "create_only",
}


@doc("プロビジョニング解除のトリガーを下流向けにどう変換するか。")
enum ProvisioningDeprovisionAction {
  deactivate: "deactivate",
  delete: "delete",
  none: "none",
}


@doc("ProvisioningTask または RemoteResourceLink が参照する内部の集約。")
enum ProvisioningSourceType {
  user: "user",
  group: "group",
}


@doc("ProvisioningTask が下流へ適用する操作。")
enum ProvisioningOperation {
  create: "create",
  update: "update",
  deactivate: "deactivate",
  delete: "delete",
  membership_add: "membership_add",
  membership_remove: "membership_remove",
}


@doc("`ProvisioningTask.status` の値の集合。`in_flight` は、ディスパッチャーが Jobs.Job を付けた時点から終端の状態に達するまでのすべてを含み、Jobs の層での試行の再試行の間も `in_flight` のままである。WorkflowRunLifecycle と同じく、一つ下の層の再試行のループは、この層の状態に反映しない。`failed` という終端でない中間の状態はない。`dead_letter` は、Job が `max_attempts` を使い切ったときにちょうど一度だけ達する終端の状態である。")
enum ProvisioningTaskStatus {
  pending: "pending",
  in_flight: "in_flight",
  succeeded: "succeeded",
  dead_letter: "dead_letter",
}


@doc("Okta にならった、接続が下流で行ってよい操作の切り替え。`sync_password` は範囲外である。")
model ProvisioningFeatureFlags {
  create_users: boolean = true;
  update_users: boolean = true;
  deactivate_users: boolean = true;
  delete_users: boolean = false;
  push_groups: boolean = false;
}


@doc("TestProvisioningConnection のディスカバリーが、下流の /ServiceProviderConfig からキャッシュする読み取り専用の対応機能。PATCH に対応しない対象で PUT へ切り替えるかどうかを決める。")
model ProvisioningCapabilities {
  supports_patch: boolean;
  supports_bulk: boolean;
  supports_filter: boolean;
  supports_etag: boolean;
  supports_sort: boolean;
  discovered_at: utcDateTime;
}


@doc("Application のエディター向けに射影する、機微でない資格情報のメタデータ。ApplicationClientSecretCredentialMetadata と同じく、平文もハッシュも公開しない。")
model ProvisioningConnectionCredentialMetadata {
  credential_id: string;
  auth_method: ProvisioningAuthMethod;
  @doc("クライアントの資格情報を交換するトークンエンドポイント。`auth_method` が `oauth2_client_credentials` のときだけ含む。シークレットではない。シークレットはクライアントシークレットであり、この射影には含めない。")
  oauth2_token_url?: string;
  @doc("トークンエンドポイントで提示するクライアントの識別子。`auth_method` が `oauth2_client_credentials` のときだけ含む。")
  oauth2_client_id?: string;
  @doc("トークンエンドポイントで求めるスコープ。`auth_method` が `oauth2_client_credentials` で、接続がスコープを設定しているときだけ含む。")
  oauth2_scope?: string;
  created_at: utcDateTime;
  rotated_at?: utcDateTime;
}


@doc("RegisterProvisioningConnection と UpdateProvisioningConnection が受け取る、書き込み専用の資格情報の入力。値はレスポンス、監査ログ、エラーのどこにも現れない。")
model ProvisioningCredentialInput {
  auth_method: ProvisioningAuthMethod;
  @doc("`auth_method` が `bearer_token` のとき必須である。")
  bearer_token?: string;
  @doc("`auth_method` が `oauth2_client_credentials` のとき必須である。")
  oauth2_token_url?: url;
  oauth2_client_id?: string;
  oauth2_client_secret?: string;
  oauth2_scope?: string;
}


@doc("`push_groups` が有効な間の、対象のグループの選び方と、その表示名の取得元。")
model GroupPushConfig {
  selection: ProvisioningGroupSelection = ProvisioningGroupSelection.assigned_groups;
  @doc("`selection` が `explicit` のときに使う、`Group.id` の値の明示的な一覧。")
  explicit_group_ids?: string[];
  @doc("下流の `displayName` の取得元。選べるのは `ProvisioningGroupDisplayNameSource` が列挙する属性だけで、デフォルトは Group の名前である。")
  display_name_source: ProvisioningGroupDisplayNameSource = ProvisioningGroupDisplayNameSource.name;
}


@doc("一つの出力属性の宣言的な対応付け。`target_path` は SCIM の属性のパスで、単純な属性、複合属性、複数値の絞り込み、enterprise 拡張のいずれでもよい。")
model AttributeMappingRule {
  @minLength(1)
  target_path: string;
  source_kind: AttributeSourceKind;
  @doc("`source_kind` が `attribute` のとき必須である。IdMagic 側の属性のキーである。")
  source_key?: string;
  @doc("`source_kind` が `constant` のとき必須である。定数は対象の属性へそのまま書き込み、その属性が保持できるものはリモートの SCIM のスキーマが決めるので、任意の JSON 値とする。")
  constant_value?: unknown;
  apply_on: AttributeApplyOn = AttributeApplyOn.create_and_update;
  @doc("true のとき、解決できない値はフェイルクローズでタスクを失敗させる。")
  required: boolean = false;
  @doc("取得元が何にも解決しないときに使う。`constant_value` と同じ理由で、対象の属性が保持できるものはリモートの SCIM のスキーマが決めるので、任意の JSON 値とする。")
  default_value?: unknown;
}


@doc("409 の衝突のときに、下流の既存のリソースを採用するために比べる属性。`externalId`（IdMagic の id）による関連付けが主な経路で、この規則は、`externalId` を設定していない既存のリソースとの衝突を解決するための副次的な経路である。")
model MatchingRule {
  conflict_match_attribute: string = "userName";
}


@doc("プロビジョニング解除のトリガーを下流の操作へどう変換するか。ユーザーの無効化は常に deactivate に対応し、設定できない。グループのメンバーシップの削除も同じく、members を削除する PATCH に対応する。デフォルトは破壊的でない方の選択肢に寄せている。")
model DeprovisionPolicy {
  @doc("Application から来る割り当て解除のトリガー。")
  on_unassign: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.deactivate;
  @doc("ユーザーの論理削除または削除のトリガー。`delete` を選ぶと、`grace_period_days` の後に完全削除する。")
  on_delete: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.deactivate;
  @doc("グループの削除または割り当て解除のトリガー。グループには deactivate に当たるものがないので、`delete` と `none` だけを許す。")
  on_group_deleted_or_unassigned: ProvisioningDeprovisionAction = ProvisioningDeprovisionAction.none;
  @minValue(0)
  grace_period_days: int64 = 0;
  @doc("一回の同期で生じる deactivate と delete の合計がこれを超えるときは、同期を実行せず、代わりに接続を隔離する。")
  accidental_deletion_count_threshold?: int64;
  @doc("同じガードを、件数ではなく接続のサブジェクトに対する割合で表したもの。")
  @minValue(1)
  @maxValue(100)
  accidental_deletion_percent_threshold?: int64;
}


@doc("送信のプロビジョニングの設定で、Application ごとに一つまでである。マスターデータは IdMagic 側の User と Group の集約で、下流はその写しである。`scope` は独自の割り当てのモデルを持たず、既存の ApplicationAssignment を再利用する。`application_id` を同一性にすることで、Application ごとに一つの接続という制約を構造で表す。")
model ProvisioningConnection {
  application_id: string;
  tenant_id: string;
  status: ProvisioningConnectionStatus = ProvisioningConnectionStatus.active;
  @doc("下流の SCIM v2 のベース URL。https でなければならず、内部またはリンクローカルの IP と、任意のリダイレクトは拒否する。")
  base_url: url;
  credential: ProvisioningConnectionCredentialMetadata;
  @doc("TestProvisioningConnection のディスカバリーがキャッシュする。それを実行するまでは設定しない。")
  capabilities?: ProvisioningCapabilities;
  feature_flags: ProvisioningFeatureFlags;
  scope: ProvisioningScope = ProvisioningScope.assigned_only;
  @doc("`feature_flags.push_groups` が有効な間だけ意味がある。")
  group_push?: GroupPushConfig;
  @doc("登録時に、SCIM のコアの User のデフォルトの対応付けを入れる。")
  attribute_mappings: AttributeMappingRule[];
  matching: MatchingRule;
  deprovision_policy: DeprovisionPolicy;
  @minValue(1)
  rate_limit_per_minute: int64 = 60;
  @minValue(1)
  max_attempts: int64 = 8;
  @doc("失敗と隔離の通知先。")
  notification_email?: string;
  @minValue(1)
  quarantine_after_consecutive_failures: int64 = 10;
  health: ProvisioningHealth = ProvisioningHealth.ok;
  @doc("隔離を決める内部のカウンター。成功したタスクが 0 に戻す。")
  consecutive_failure_count: int64 = 0;
  last_full_sync_at?: utcDateTime;
  quarantined_at?: utcDateTime;
  quarantine_reason?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("IdMagic の User または Group と下流のリソースを関連付ける永続的な記録。409 または 404 を冪等に解決できるように保持する。")
model RemoteResourceLink {
  connection_id: string;
  tenant_id: string;
  source_type: ProvisioningSourceType;
  @doc("IdMagic 側の `User.id` または `Group.id`。")
  source_id: string;
  @doc("下流の SCIM のリソースの id。")
  remote_id: string;
  @doc("下流へ送る `externalId`。デフォルトは IdMagic の `source_id` である。")
  external_id: string;
  @doc("`capabilities.supports_etag` が true のときだけ意味がある。")
  etag?: string;
  @doc("直近に適用した `source_version`。順序の入れ替わったタスクの解決に使う。")
  last_synced_version: int64;
  updated_at: utcDateTime;
}


@doc("一つの内部のライフサイクルのイベントに対する、一つのプロビジョニングタスク。冪等キー `(tenant_id, connection_id, source_type, source_id, source_version)` は一意性の制約であり、同じキーで再びキューへ入れても既存の行に対して何もしない。ディスパッチャーは Jobs へキューを入れた後に `job_id` を付け、それまでは設定しない。")
model ProvisioningTask {
  id: string;
  tenant_id: string;
  connection_id: string;
  source_type: ProvisioningSourceType;
  source_id: string;
  @doc("同じトランザクションでの捕捉が出力する、単調に増えるカウンター。順序の入れ替わったタスクの解決に使う。")
  source_version: int64;
  operation: ProvisioningOperation;
  status: ProvisioningTaskStatus = ProvisioningTaskStatus.pending;
  @doc("ディスパッチャーがキューへ入れた後に付ける `Jobs.JobRef.id`。")
  job_id?: string;
  @doc("直近の失敗の、無害化したエラー。シークレットも属性値も含まない。")
  last_error?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("管理者が、Application に新しい ProvisioningConnection を登録した。")
model ProvisioningConnectionRegistered {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("管理者が、資格情報を除く ProvisioningConnection の設定を更新した。")
model ProvisioningConnectionUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("管理者が `ProvisioningConnection.status` を disabled にした。新しいタスクは作らない。")
model ProvisioningConnectionDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("管理者が ProvisioningConnection を削除した。ApplicationStatus と同じく、行そのものを削除する。")
model ProvisioningConnectionDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("管理者が接続の資格情報を更新した。")
model ProvisioningCredentialRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  credentialId: string;
}


@doc("ディスパッチャーが ProvisioningTask に Jobs.Job を付け、タスクが始まった。LifecycleWorkflowRunStarted と同じ形である。")
model ProvisioningTaskStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  taskId: string;
  jobId: string;
}


@doc("ユーザーの作成または更新が下流に届き、タスクが succeeded として確定した。")
model UserProvisioned {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  taskId: string;
  userId: string;
  remoteId: string;
}


@doc("ユーザーの deactivate または削除が下流に届き、タスクが succeeded として確定した。")
model UserDeprovisioned {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  taskId: string;
  userId: string;
  action: ProvisioningDeprovisionAction;
}


@doc("ユーザーのタスクが `max_attempts` を使い切り、dead_letter として確定した。試行ごとには発行しない。デッドレターになったグループのタスクも同じイベントを使う。")
model UserProvisioningFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  taskId: string;
  sourceType: ProvisioningSourceType;
  sourceId: string;
  error: string;
}


@doc("グループの作成、更新、deactivate、または削除が下流に届き、タスクが succeeded として確定した。")
model GroupPushed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  taskId: string;
  groupId: string;
  remoteId: string;
}


@doc("グループのメンバーシップの変更が members への PATCH として下流に届き、タスクが succeeded として確定した。")
model GroupMembershipPushed {
  occurredAt: utcDateTime;
  tenantId: string;
  connectionId: string;
  taskId: string;
  groupId: string;
}


@doc("連続した失敗、または誤削除のガードを超えたことにより、`ProvisioningConnection.health` が quarantined になった。以降のタスクは作らない。")
model ConnectionQuarantined {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  reason: string;
  consecutiveFailures: int64;
}


@doc("管理者が ResumeProvisioningConnection で隔離を解除した。隔離から抜ける方法がなければ回復できないので、このイベントを加えた。")
model ProvisioningConnectionQuarantineCleared {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
}


@doc("StartFullResync が対象とした範囲内のすべてのサブジェクトが収束した。")
model FullResyncCompleted {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  totalSubjects: int64;
  succeededCount: int64;
  failedCount: int64;
}

}
