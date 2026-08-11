namespace IdMagic.Contract {

@doc("属性値の sum type discriminator。Tenancy context が所有する定義の published language stub。")
enum AttributeType {
  String: "String",
  Number: "Number",
  Boolean: "Boolean",
  Date: "Date",
  StringArray: "StringArray",
}


@doc("Group の所属管理方式。作成後は変更できない。")
enum GroupMembershipType {
  manual: "manual",
  dynamic: "dynamic",
}


@doc("GroupMember を作成した唯一の経路。")
enum GroupMembershipSource {
  manual: "manual",
  dynamic_rule: "dynamic_rule",
}

@doc("tenant-scoped ロール束集約。所属 User に roles を一斉付与する。id は不変の生成識別子、name はテナント内で一意な編集可能ラベル。email は連絡先メールアドレス (認証には使わない)、attributes はテナント定義の TenantGroupAttributeSchema に対して検証される sparse な custom 属性 bag。")
model Group {
  @minLength(1)
  @maxLength(64)
  id: string;
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  @doc("グループ宛の連絡先メールアドレス。User.email と異なり認証・検証フローは持たない。")
  email?: string;
  @doc("テナント定義の TenantGroupAttributeSchema に対して検証される custom 属性。")
  attributes?: Record<AttributeValue>;
  roles: string[];
  membership_type: GroupMembershipType = GroupMembershipType.manual;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("Group の custom 属性 1 件の定義。User の UserAttributeDef と異なり、Group には builtin OIDC/SCIM カタログ・self-service 編集主体・claim 露出が存在しないため、型と要否だけを持つ簡略版。")
model GroupAttributeDef {
  @minLength(1)
  key: string;
  label?: string;
  type: AttributeType;
  multi_valued: boolean = false;
  required: boolean = false;
}


@doc("rule version 単位の全件再評価状態。")
enum DynamicMembershipEvaluationStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  failed: "failed",
}


@doc("Agent の運用状態。Active は通常稼働、Disabled は可逆な運用停止、Killed は緊急停止 (一方向終端)。Active 以外は新規トークンを発行しない (fail-closed)。")
enum AgentStatus {
  Active: "Active",
  Disabled: "Disabled",
  Killed: "Killed",
}


@doc("Agent の自律性区分。Autonomous は自律実行、Supervised は人間の監督下で実行する。")
enum AgentKind {
  Autonomous: "Autonomous",
  Supervised: "Supervised",
}


@doc("roles に未知のロールが含まれる。")
model InvalidRoleError {

}


@doc("操作者が自分自身の User を削除しようとした。")
model SelfDeleteForbiddenError {

}


@doc("操作者が自分自身の User を無効化しようとした。")
model SelfDisableForbiddenError {

}


@doc("RestoreAdminUser の対象 User が PendingDeletion 状態でない。")
model UserNotPendingDeletionError {

}


@doc("RestoreAdminUser が削除猶予期間の経過後に呼ばれた。")
model RestoreGracePeriodExpiredError {

}


@doc("カスタム属性値がテナントの属性スキーマを満たさない。")
model InvalidUserAttributeError {

}


@doc("Group のカスタム属性値がテナントの Group 属性スキーマ (TenantGroupAttributeSchema) を満たさない。")
model InvalidGroupAttributeError {

}


@doc("SetUserRequiredAction に未知の RequiredAction 値が指定された。")
model InvalidRequiredActionError {

}


@doc("Group の name が空である。")
model GroupNameRequiredError {

}


@doc("dynamic group rule (CEL 式) がコンパイル・検証に失敗した (未知の属性・非Boolean・構文エラー等)。")
model InvalidDynamicGroupRuleError {

}


@doc("Agent の name が空である。")
model AgentNameRequiredError {

}


@doc("owner が必須な Agent 種別で owner_user_id が指定されていない。")
model AgentOwnerRequiredError {

}


@doc("指定した owner_user_id がテナント内の User に解決できない。")
model AgentOwnerNotFoundError {

}


@doc("指定された email がメールアドレスの形式を満たさない。")
model InvalidEmailError {

}


@doc("指定された新しい email が現在の email と同一である。")
model EmailUnchangedError {

}


@doc("指定した export column が対象の許可リストに含まれない。")
model InvalidExportColumnsError {

}


@doc("export target の種別がサポートされていない。")
model InvalidExportTargetError {

}


@doc("指定した status filter が対象の export target に許可されていない。")
model InvalidExportFilterError {

}


@doc("User の運用ライフサイクル属性。status は状態機械 UserLifecycle\n(states セクション) と一致する唯一の真実で、旧 disabled_at / deleted_at は\nstatus + status_changed_at に統合した。tombstone は status == Deleted。")
model UserLifecycle {
  status: UserStatus = UserStatus.Active;
  @doc("現在の status になった時刻。詳細な遷移履歴は監査イベントが持つ。")
  status_changed_at?: utcDateTime;
  last_login_at?: utcDateTime;
  password_changed_at?: utcDateTime;
  required_actions?: RequiredAction[];
}


@doc("属性 1 件の値 (sum type)。type が示すフィールドだけが設定される。OIDC 組み込み属性と tenant 定義 custom で共通。")
model AttributeValue {
  type: AttributeType;
  string?: string;
  number?: unknown;
  boolean?: boolean;
  date?: string;
  string_array?: string[];
}

@doc("IdP の canonical user aggregate。protocol subject はこの id から写像し、プロフィール拡張は sparse attributes に保持する。")
model User {
  @minLength(1)
  id: string;
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(100)
  preferred_username: string;
  password_hash?: string;
  name?: string;
  given_name?: string;
  family_name?: string;
  email?: string;
  email_verified: boolean = false;
  mfa_enrolled: boolean = false;
  roles: string[] = #[];
  lifecycle: UserLifecycle;
  attributes?: Record<AttributeValue>;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("管理者がユーザーを作成する JSON リクエスト。初期パスワードには Authentication context の PasswordPolicy を適用する。")
model AdminUserCreateRequest {
  @minLength(1)
  @maxLength(100)
  preferred_username: string;
  password: string;
  name?: string;
  email?: string;
  email_verified: boolean = false;
  roles: string[] = #[];
}


@doc("管理者がユーザーの可変プロフィールとロールを更新する JSON リクエスト。id と password_hash は変更しない。")
model AdminUserUpdateRequest {
  @minLength(1)
  @maxLength(100)
  preferred_username?: string;
  name?: string;
  email?: string;
  email_verified?: boolean;
  roles?: string[];
}


@doc("管理 API のユーザー一覧と、query / status に依存せず削除済みを除いた tenant 内の正確な総ユーザー数。")
model AdminUserListResponse {
  users: AdminUserResponse[];
  @doc("tenant 内で lifecycle status が deleted ではない User の正確な件数。query / status の影響を受けない。")
  total_users?: int64;
}


@doc("管理 API が返すユーザー表現。password_hash は契約上含めない。")
model AdminUserResponse {
  id: string;
  @doc("所属テナント ID。")
  tenant_id: string;
  preferred_username: string;
  name?: string;
  email?: string;
  email_verified: boolean;
  mfa_enrolled: boolean;
  roles: string[];
  @doc("次回ログイン時の強制アクション。")
  required_actions?: RequiredAction[];
  @doc("直近の認証完了時刻。")
  last_login_at?: utcDateTime;
  @doc("直近のパスワード変更時刻。")
  password_changed_at?: utcDateTime;
  @doc("旧世代の無効化時刻フィールド。User.lifecycle.status + status_changed_at が唯一の真実であり、本フィールドは現状レスポンス後方互換のためだけに残る (別途整理が必要な既存ギャップ、本移行のスコープ外)。")
  disabled_at?: utcDateTime;
  created_at: utcDateTime;
  @doc("作成時は created_at と同じ値を持ち、以後管理操作ごとに更新する。")
  updated_at: utcDateTime;
}


@doc("SetUserRequiredAction の JSON リクエスト。action は RequiredAction enum の wire 値。")
model AdminRequiredActionRequest {
  action: RequiredAction;
}


@doc("`/api/account/profile` の返り値 (self-service)。呼び出しユーザ自身の表示用\nprojection。attributes は self が読める分 (self_readable / claim_exposed) のみ、\neditable_attributes は editable_by_user=true の定義一覧で、UI のフォーム描画に使う。")
model AccountProfileResponse {
  id: string;
  preferred_username: string;
  name?: string;
  given_name?: string;
  family_name?: string;
  email?: string;
  email_verified: boolean;
  mfa_enrolled: boolean;
  status: UserStatus;
  attributes: unknown;
  editable_attributes: UserAttributeDef[];
}


@doc("self-service プロフィール更新リクエスト。表示名と editable_by_user=true の属性\nのみを受け付ける。attributes は key 単位の merge (指定 key のみ upsert、未指定は\n現状維持) で、admin 管理属性 (status / roles / organization) は対象外。")
model AccountProfileUpdateRequest {
  name?: string;
  given_name?: string;
  family_name?: string;
  attributes?: unknown;
}


@doc("CSV ユーザーインポートジョブの公開ビュー。preview は副作用を起こさず、apply は同一 tenant の成功済み preview と source_sha256 に結合する。結果は操作別件数と値を含まない stable row error を持ち、password/hash は入力・保存しない。")
model UserImportJob {
  id: string;
  status: string;
  mode: UserImportMode;
  source_sha256?: string;
  preview_job_id?: string;
  total_rows: int64;
  created_rows?: int64;
  updated_rows?: int64;
  unchanged_rows?: int64;
  @doc("旧クライアント移行用の互換フィールド。created_rows + updated_rows + unchanged_rows。新規クライアントは操作別件数を使う。")
  accepted_rows?: int64;
  rejected_rows: int64;
  errors?: UserImportRowError[];
  error_total?: int64;
}


@doc("CSV の 1 行に対する安全な検証または適用エラー。値そのものは返さず、列名と stable code のみを返す。")
model UserImportRowError {
  row: int64;
  column?: string;
  code: string;
}


@doc("CSV ユーザーインポートジョブの段階。preview は read-only な計画、apply は成功済み preview payload の現在状態に対する再計画と適用。")
enum UserImportMode {
  dry_run: "dry_run",
  preview: "preview",
  apply: "apply",
}


@doc("User export/import が共有する configurable safety policy。既定は 100,000 data rows・64 MiB/file・64 KiB/field。成功した User export は常に同じ policy 内にあり、そのまま preview できる。上限は tenant 容量の意味ではなく、一つの非同期 artifact に対する運用上の境界である。")
model UserCsvTransferPolicy {
  max_rows: int64;
  max_bytes: int64;
  max_field_bytes: int64;
}


@doc("CSV エクスポートの対象リソース種別。列定義は種別ごとの allowlist に閉じ、未レビュー属性が自動で外へ出ないようにする。")
enum DataExportTargetKind {
  user: "user",
  group: "group",
  group_membership: "group_membership",
}


@doc("エクスポートの出力形式。初期は CSV (RFC 4180) のみ。")
enum DataExportFormat {
  csv: "csv",
}


@doc("リソースエクスポートジョブの状態。DataExportLifecycle と一致する。succeeded はファイルをダウンロード可能な状態、expired はファイル本体が TTL 経過で purge され metadata と監査だけが残る状態。")
enum DataExportStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  failed: "failed",
  canceled: "canceled",
  expired: "expired",
}


@doc("エクスポート対象種別の allowlist に含まれる 1 列。key は CSV ヘッダーにも使う安定した機械名、label は画面表示専用、pii は個人情報を含むか。User の allowlist は 13 個の組み込みコア列に加え、attr:<key> (OIDC profile/phone/address 相当と SCIM enterprise 相当の組織属性からなる、全 tenant 共通の組み込み拡張属性 27 種) と custom:<key> (実効 TenantUserAttributeSchema から tenant ごとに解決するテナント定義属性) を含む。password_hash・secret・token・recovery code・MFA secret などの sensitive 値は allowlist に一切含めない。")
model DataExportColumn {
  key: string;
  label: string;
  pii: boolean = false;
}


@doc("管理者がエクスポートを開始する入力。対象種別は endpoint (path) が決めるため body には持たない。出力形式、列選択 (allowlist の部分集合)、一覧画面から引き継いだフィルタを持つ。列に allowlist 外の key を含む場合は拒否する。")
model DataExportRequest {
  format: DataExportFormat = DataExportFormat.csv;
  columns: string[];
  filter?: unknown;
}


@doc("リソースエクスポートジョブの公開ビュー兼監査可能な記録。CSV 本体や PII の値は含まず、対象種別・選択列 key・件数・バイト数・状態・期限・要求者だけを持つ。")
model DataExportJob {
  id: string;
  status: DataExportStatus;
  target: DataExportTargetKind;
  format: DataExportFormat;
  requested_columns: string[];
  filename?: string;
  total_rows?: int64;
  byte_size?: int64;
  error_code?: string;
  requested_by: string;
  created_at: utcDateTime;
  completed_at?: utcDateTime;
  expires_at?: utcDateTime;
  downloadable: boolean = false;
}


@doc("ダウンロード時に返す完成済みエクスポートファイル。content は CSV 本体で PII を含みうるため、succeeded かつ未期限のジョブに対してのみ、認可済み管理者へ content-disposition attachment として返す。")
model DataExportFile {
  filename: string;
  content_type: string;
  byte_size: int64;
  content: string;
}


@doc("エクスポートの失敗理由。値そのものは返さず stable code のみを返す。")
model DataExportError {
  code: string;
}


@doc("管理者がリソースエクスポートを要求し、ジョブが受理された。CSV 値は持たず、対象種別・選択列 key・要求者だけを記録する。")
model DataExportRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
  target: DataExportTargetKind;
  requestedColumns: string[];
}


@doc("worker がリソースエクスポートの生成を開始した。")
model DataExportStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
}


@doc("リソースエクスポートの生成が完了し、ファイルがダウンロード可能になった。行数とバイト数だけを記録し、値は持たない。")
model DataExportSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
  totalRows: int64;
  byteSize: int64;
}


@doc("リソースエクスポートの生成が失敗し、不完全ファイルはダウンロード不可のまま終端した。")
model DataExportFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
  errorCode: string;
}


@doc("終端前のリソースエクスポートが管理者によって取消された。")
model DataExportCanceled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
}


@doc("管理者が完成済みエクスポートファイルをダウンロードした。PII を大量に取り出す高リスク操作として監査に残す。")
model DataExportDownloaded {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
  target: DataExportTargetKind;
}


@doc("保持期限を経過し、エクスポートファイル本体が purge された。metadata と監査は残る。")
model DataExportExpired {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
}


@doc("`/api/account/summary` の返り値 (end-user account portal のホーム)。呼び出し\nユーザ自身のアカウント概要で、admin shell 用の AccountContext (/api/auth/account)\nとは別契約。roles は含めず、最終ログイン・パスワード変更日時・MFA 状態・未対応の\nrequired actions を portal home の summary card 用に返す。")
model AccountSummary {
  id: string;
  preferred_username: string;
  name?: string;
  email?: string;
  email_verified: boolean;
  mfa_enrolled: boolean;
  status: UserStatus;
  last_login_at?: utcDateTime;
  password_changed_at?: utcDateTime;
  required_actions: RequiredAction[];
}


@doc("primary email 変更の起票リクエスト (self-service)。新アドレスへ確認リンクを\n送るだけで、User.email の反映は確認完了 (ConfirmEmailChange) まで起きない。")
model AccountEmailChangeRequest {
  new_email: string;
}


@doc("新アドレスへ送ったワンタイムリンクの確認リクエスト (self-service)。token が\n新アドレス所有の証左なので認証済みセッションは要求しない (reset password と同方針)。")
model AccountEmailChangeConfirm {
  @minLength(1)
  token: string;
}


@doc("end-user 自身の個人データエクスポート (GDPR 第15条 right of access)。本ステージは\n同期生成で、現状 API から得られる profile と接続済みアプリ (consents) をまとめる。")
model AccountDataExport {
  exported_at: utcDateTime;
  profile: AccountProfileResponse;
  consents: AccountConsentResponse[];
}


@doc("User と Group の所属関係。group_id × user_id で一意。")
model GroupMember {
  group_id: string;
  user_id: string;
  source: GroupMembershipSource = GroupMembershipSource.manual;
  @doc("dynamic_rule の場合に評価した DynamicGroupRule.version。")
  rule_version?: int64;
  created_at: utcDateTime;
}


@doc("dynamic Group と一対一の制限 CEL rule。expression は Boolean を返し、未知属性・許可外関数・過大な式を保存できない。")
model DynamicGroupRule {
  group_id: string;
  tenant_id: string;
  @minLength(1)
  @maxLength(4096)
  expression: string;
  enabled: boolean = false;
  version: int64;
  referenced_attributes: string[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("rule version の全件再評価と所属差分の監査可能な結果。")
model DynamicMembershipEvaluation {
  id: string;
  tenant_id: string;
  group_id: string;
  rule_version: int64;
  job_id?: string;
  status: DynamicMembershipEvaluationStatus;
  added_count: int64 = 0;
  removed_count: int64 = 0;
  unchanged_count: int64 = 0;
  error_count: int64 = 0;
  created_at: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("CEL rule の作成・更新入力。")
model DynamicGroupRuleRequest {
  @minLength(1)
  @maxLength(4096)
  expression: string;
}


@doc("保存前 CEL と最大100 userの read-only preview入力。")
model DynamicGroupPreviewRequest {
  expression: string;
  user_ids: string[];
}


@doc("Userごとの一致と現在所属との差分。属性値は含めない。")
model DynamicGroupPreviewResult {
  user_id: string;
  matched: boolean;
  change: string;
  error_code?: string;
}


@doc("管理者がグループを作成する JSON リクエスト。")
model GroupCreateRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  email?: string;
  attributes?: Record<AttributeValue>;
  roles?: string[];
  membership_type?: GroupMembershipType;
  dynamic_rule?: DynamicGroupRuleRequest;
}


@doc("管理者がグループの name / description / email / attributes / roles を更新する JSON リクエスト。省略フィールドは現状維持。")
model GroupUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  email?: string;
  attributes?: Record<AttributeValue>;
  roles?: string[];
}


@doc("グループ表現。管理 UI の一覧と詳細で同じ表現を返す。")
model GroupSummaryResponse {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  email?: string;
  attributes?: Record<AttributeValue>;
  roles: string[];
  member_count: int64;
  membership_type: GroupMembershipType;
  dynamic_rule?: DynamicGroupRule;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("管理 API のグループ一覧。")
model GroupListResponse {
  groups: GroupSummaryResponse[];
}


@doc("グループ所属メンバー表現。")
model GroupMemberResponse {
  user_id: string;
  preferred_username: string;
  created_at: utcDateTime;
}


@doc("User 詳細画面など group_id を body で指定して所属させる際の任意リクエスト。")
model GroupAddMemberRequest {
  group_id: string;
}


@doc("指定 User の所属グループと有効ロール内訳。明示ロール (user.roles) と\nグループ由来ロール、その union を分けて返し、管理 UI が effective roles を\n理解しやすくする。effective_roles = direct_roles ∪ group_roles で、User と Group\n両方にまたがる導出値であるため models.User の constraint ではなく本 response の\nフィールドとして計算・公開する。")
model UserGroupsResponse {
  groups: GroupSummaryResponse[];
  @doc("user.roles。")
  direct_roles: string[];
  @doc("所属グループ由来ロールの union。")
  group_roles: string[];
  @doc("direct_roles ∪ group_roles。")
  effective_roles: string[];
}


@doc("Agent と同一テナントの OAuth2Client の束縛関係。agent_id × client_id で一意。Agent は束縛した Client の資格情報を用いてトークンを得る。束縛先 client_id は Agent と同一 tenant_id の OAuth2Client でなければならない (AgentReferencesStayTenantLocal)。client_id は他の Agent と重複して束縛できない (1 client_id につき Agent は最大 1 つ。BindAgentCredential の requires で検証)。")
model AgentCredentialBinding {
  agent_id: string;
  client_id: string;
  created_at: utcDateTime;
}


@doc("管理者が Agent を登録する JSON リクエスト。owner_user_id 省略時は actor を所有者とする。")
model AgentRegisterRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  kind?: AgentKind;
  @doc("省略時は actor の id を所有者とする。")
  owner_user_id?: string;
  roles?: string[];
}


@doc("管理者が Agent の name / description / kind / owner_user_id / roles を更新する JSON リクエスト。省略フィールドは現状維持。owner_user_id の変更は所有権移転。")
model AgentUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  kind?: AgentKind;
  @doc("変更は所有権移転 (AgentOwnerChanged)。")
  owner_user_id?: string;
  roles?: string[];
}


@doc("Agent に OAuth2Client を束縛する JSON リクエスト。")
model AgentCredentialBindRequest {
  @minLength(1)
  client_id: string;
}


@doc("Agent 表現。管理 UI の一覧と詳細で同じ表現を返す。")
model AgentSummaryResponse {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  kind: AgentKind;
  owner_user_id: string;
  status: AgentStatus;
  roles: string[];
  @doc("束縛済み OAuth2Client id。")
  client_ids: string[];
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
  killed_at?: utcDateTime;
}


@doc("管理 API の Agent 一覧。")
model AgentListResponse {
  agents: AgentSummaryResponse[];
}


@doc("ローカル user に外部 IdP の identity を紐付けた (account linking)。")
model FederationLinked {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  provider: string;
}


@doc("ローカル user から外部 IdP の identity を解除した。")
model FederationUnlinked {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  provider: string;
}


@doc("end-user が primary email の変更を起票し、新アドレスへ確認リンクを送った\n(self-service)。実際の email 反映は ConfirmEmailChange まで起きない。\nnewEmailHash は SHA-256 で、平文を audit log に流さない。")
model EmailChangeRequested {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
  @doc("新 email の SHA-256 hex")
  newEmailHash: string;
}


@doc("新アドレスへ送ったワンタイムリンクの確認が完了し、primary email を確定した\n(self-service)。email_verified=true に更新し、verify_email の\nrequired action があれば自動解除する。")
model EmailChanged {
  occurredAt: utcDateTime;
  @doc("対象 user の所属テナント ID。")
  tenantId: string;
  userId: string;
}


@doc("管理者がユーザーを作成した。初期パスワードと password_hash は監査イベントに含めない。")
model UserCreated {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者がユーザーのプロフィールまたは roles を更新した。")
model UserUpdated {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  changedFields: string[];
}


@doc("管理者がユーザーを無効化した。対象ユーザーの新規ログインと既存セッション利用を拒否する。")
model UserDisabled {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者が無効化済みユーザーを再有効化した。")
model UserEnabled {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者がユーザーを SoftDelete した。status は PendingDeletion に遷移し、PII / Consent / RefreshToken / Session は温存される。reason は admin が任意に残せる自由記述。")
model UserSoftDeleted {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  reason?: string;
}


@doc("管理者が PendingDeletion のユーザーを Restore した。status は Active に戻り、PII / credential は温存されたままでログインが再開する。")
model UserRestored {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者または 猶予期間経過後の自動 purge がユーザーを Purge した。PII は anonymize 済みで、関連 aggregate は cascade 削除されている。reason は admin が任意に残せる自由記述 (自動 purge では auto_purge)。")
model UserDeleted {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  reason?: string;
}


@doc("管理者が次回ログイン時の強制アクション (RequiredAction) を付与した。action は enum 値で、監査に平文で残しても安全。")
model UserRequiredActionSet {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("強制アクションが解除された。admin の明示解除のほか、本人がパスワードを変更した結果 update_password が自動解除される場合も発火し、その際 actor の id は対象本人の id になる。")
model UserRequiredActionCleared {
  occurredAt: utcDateTime;
  @doc("操作対象テナント ID。actor / target と一致する。")
  tenantId: string;
  @doc("解除を行った主体。自動解除時は対象本人。")
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者がグループを作成した。")
model GroupCreated {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
}


@doc("管理者がグループの name / description / roles を更新した。")
model GroupUpdated {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  changedFields: string[];
}


@doc("管理者がグループを削除した。所属 membership は cascade で解除され、各々 GroupMemberRemoved を伴う。")
model GroupDeleted {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
}


@doc("管理者が User をグループに所属させた。既に所属済みの場合は emit されない (冪等)。")
model GroupMemberAdded {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  userId: string;
}


@doc("管理者が User をグループから外した、またはグループ削除に伴い解除された。所属していない場合は emit されない (冪等)。")
model GroupMemberRemoved {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  userId: string;
}


@doc("管理者が dynamic group の CEL rule を新しい version へ更新した。")
model DynamicGroupRuleUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("管理者が検証済み rule を有効化し全件再評価を予約した。")
model DynamicGroupRuleEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("管理者が rule を無効化し旧 dynamic membership を即時無効にした。")
model DynamicGroupRuleDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("rule version の全件再評価が完了した。属性値は持たず差分件数だけを記録する。")
model DynamicMembershipEvaluated {
  occurredAt: utcDateTime;
  tenantId: string;
  groupId: string;
  ruleVersion: int64;
  addedCount: int64;
  removedCount: int64;
  unchangedCount: int64;
  errorCount: int64;
}


@doc("worker が dynamic membership 全件再評価を開始した。")
model DynamicMembershipEvaluationStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("dynamic membership 全件再評価が失敗し、対象 rule version は fail-closed のままになった。")
model DynamicMembershipEvaluationFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("管理者が Agent を登録した。")
model AgentRegistered {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が Agent の name / description / kind / roles を更新した。")
model AgentUpdated {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  changedFields: string[];
}


@doc("管理者が Agent を可逆に運用停止した。")
model AgentDisabled {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が運用停止中の Agent を再稼働した。")
model AgentEnabled {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が Agent を緊急停止した (一方向終端)。以後復帰不能。")
model AgentKilled {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が Agent を削除した。束縛は cascade で解除される。")
model AgentDeleted {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が Agent の所有者 (owner_user_id) を移転した。")
model AgentOwnerChanged {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  previousOwnerUserId: string;
  newOwnerUserId: string;
}


@doc("管理者が Agent に OAuth2Client を束縛した。")
model AgentCredentialBound {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  clientId: string;
}


@doc("管理者が Agent から OAuth2Client の束縛を解除した。")
model AgentCredentialUnbound {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント ID。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  clientId: string;
}

}
