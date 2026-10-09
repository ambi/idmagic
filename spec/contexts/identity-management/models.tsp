namespace IdMagic.Contract {

@doc("属性値の直和型の判別子。Tenancy モジュールが所有する定義を、公開契約として写したもの。")
enum AttributeType {
  String: "string",
  Number: "number",
  Boolean: "boolean",
  Date: "date",
  StringArray: "string_array",
}


@doc("Group のメンバーシップの管理の方法。作成後は変えられない。")
enum GroupMembershipType {
  manual: "manual",
  dynamic: "dynamic",
}


@doc("GroupMember を作った唯一の経路。CSV では読み取り専用である。メンバーシップの行は `membership_state` で望むメンバーシップを示し、既存のものを変換することはないので、`dynamic_rule` のメンバーシップは、ローカルのファイルが解除したり所有者を付け替えたりせず、拒否する。")
enum GroupMembershipSource {
  manual: "manual",
  dynamic_rule: "dynamic_rule",
}

@doc("テナントに属するロールの束の集約。`roles` をすべてのメンバーの User に一度に与える。`id` は生成した変わらない識別子、`name` はテナントの中で一意な編集できるラベルである。CSV の行はまず `id` で対象を解決し、なければ `name` を使うので、既知の `id` と別の Group が保持する `name` の両方を含む行は、解決せずに拒否する。`email` は連絡先のアドレスで認証には使わず、`attributes` はテナントの TenantGroupAttributeSchema に照らして検証するカスタム属性のまばらな集まりである。どちらも CSV で往復し、`attributes` は `custom:<key>` の列になる。削除は即時で取り消せない。Group には予約の状態がないので、CSV の行は目標の状態の列ではなく、明示的な `lifecycle_action` の列で削除を求める。")
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
  @doc("グループの連絡先のメールアドレス。`User.email` と違い、認証や確認のフローはない。")
  @maxLength(254)
  email?: string;
  @doc("テナントの TenantGroupAttributeSchema に照らして検証するカスタム属性。")
  attributes?: Record<AttributeValue>;
  roles: string[];
  membership_type: GroupMembershipType = GroupMembershipType.manual;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
}


@doc("Group の一つのカスタム属性の定義。User の UserAttributeDef と違い、Group には組み込みの OIDC や SCIM の一覧も、セルフサービスの編集も、クレームとしての公開もないので、型と必須かどうかだけを含む縮めた形である。")
model GroupAttributeDef {
  @minLength(1)
  key: string;
  @maxLength(100)
  label?: string;
  type: AttributeType;
  multi_valued: boolean = false;
  required: boolean = false;
}


@doc("一つの規則のバージョンに対する全体の再評価の状態。")
enum DynamicMembershipEvaluationStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  failed: "failed",
}


@doc("Agent の運用上の状態。`active` は通常どおり動き、`disabled` は元に戻せる一時停止、`killed` は一方向の終端の緊急停止である。`active` 以外では、フェイルクローズで新しいトークンを発行しない。")
enum AgentStatus {
  Active: "active",
  Disabled: "disabled",
  Killed: "killed",
}


@doc("Agent の自律性。`autonomous` は自ら動き、`supervised` は人の監督の下で動いて、人の承認を記録する発行の経路でだけトークンを得る（REQ-OAUTH2-050）。要否は `supervised` があることではなく `autonomous` がないことで決めるので、サーバーが認識しない値も承認を求める。")
enum AgentKind {
  Autonomous: "autonomous",
  Supervised: "supervised",
}


@doc("`roles` に、サーバーが保存を拒否するロールが含まれる。空の名前か、保持してはならない対象に新たに割り当てる予約済みのロール `system_admin` である。`system_admin` は制御面に予約しているので、書き込みでこれを加えられるのは制御面テナントの User または Group だけで、どのテナントでも Agent には加えられない。判断は呼び出し元自身のロールではなく対象の種類とテナントを読み、結果の集合ではなく書き込みが加えるものを読むので、対象にすでに保存されている値は再び送ってよく、明示的な `roles` の更新でそれを外すこともできる。拒否した書き込みは、同じリクエストのほかのフィールドを含め、何も保存しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_role` であり、`detail` が二つの理由のどちらに当たるかを示す。")
model InvalidRoleError is ProblemDetails;


@doc("求めた `preferred_username` は、テナントの別の User がすでに使っている。名前は前後の空白を除き Unicode の大文字と小文字の畳み込みをしてから比べるので、大文字と小文字だけが違う名前も衝突する。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:username_conflict` である。")
model UsernameConflictError is ProblemDetails;


@doc("求めた `email` は、テナントの削除されていない別の User がすでに使っている。アドレスは前後の空白を除き Unicode の大文字と小文字の畳み込みをしてから比べる。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:email_taken` である。")
model EmailTakenError is ProblemDetails;


@doc("求めた Group の `name` は、テナントの別の Group がすでに使っている。名前は前後の空白を除き Unicode の大文字と小文字の畳み込みをしてから比べる。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:group_name_conflict` である。")
model GroupNameConflictError is ProblemDetails;


@doc("求めた Agent の `name` は、テナントの別の Agent がすでに使っている。名前は前後の空白を除き Unicode の大文字と小文字の畳み込みをしてから比べる。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_name_conflict` である。")
model AgentNameConflictError is ProblemDetails;


@doc("操作者が自身の User を削除しようとした。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:self_delete_forbidden` である。")
model SelfDeleteForbiddenError is ProblemDetails;


@doc("操作者が自身の User を無効にしようとした。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:self_disable_forbidden` である。")
model SelfDisableForbiddenError is ProblemDetails;


@doc("RestoreAdminUser の対象の User が `pending_deletion` の状態にない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:not_pending_deletion` である。")
model UserNotPendingDeletionError is ProblemDetails;


@doc("DisableAdminUser または EnableAdminUser の対象の User が `pending_deletion` の状態にある。予定した削除を取り消せるのは RestoreAdminUser だけである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:user_pending_deletion` である。")
model UserPendingDeletionError is ProblemDetails;


@doc("削除の猶予期間を過ぎてから RestoreAdminUser を呼んだ。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:restore_grace_expired` である。")
model RestoreGracePeriodExpiredError is ProblemDetails;


@doc("カスタム属性の値がテナントの属性のスキーマを満たさない。RFC 9457 の Problem Details で、`type` はグループの属性の拒否と同じコードの `urn:idmagic:error:invalid_attribute` であり、両者はエンドポイントで区別する。")
model InvalidUserAttributeError is ProblemDetails;


@doc("Group のカスタム属性の値が、テナントの Group の属性のスキーマ（TenantGroupAttributeSchema）を満たさない。RFC 9457 の Problem Details で、`type` はユーザーの属性の拒否と同じコードの `urn:idmagic:error:invalid_attribute` であり、両者はエンドポイントで区別する。")
model InvalidGroupAttributeError is ProblemDetails;


@doc("SetUserRequiredAction に、サーバーが知らない RequiredAction の値を与えた。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_required_action` である。")
model InvalidRequiredActionError is ProblemDetails;


@doc("Group の `name` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:group_name_required` である。")
model GroupNameRequiredError is ProblemDetails;


@doc("動的グループの規則（CEL の式）のコンパイルまたは検証に失敗した。未知の属性、真偽値でない結果、構文の誤りなどである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_dynamic_group_rule` である。")
model InvalidDynamicGroupRuleError is ProblemDetails;


@doc("Agent の `name` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_name_required` である。")
model AgentNameRequiredError is ProblemDetails;


@doc("所有者を求める Agent の種類で、`owner_user_id` がない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_owner_required` である。")
model AgentOwnerRequiredError is ProblemDetails;


@doc("指定した `owner_user_id` が、テナントの User へ解決しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_owner_not_found` である。")
model AgentOwnerNotFoundError is ProblemDetails;


@doc("リクエストの対象の Agent がこのテナントに存在しない。他のテナントの Agent は存在しないものとして扱い、同じように答える。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_not_found` である。")
model AgentNotFoundError is ProblemDetails;


@doc("Agent は一方向の終端の操作で強制停止されているので、更新、無効化、有効化、再度の強制停止、削除、資格情報の付与のどれもできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_killed` である。")
model AgentKilledError is ProblemDetails;


@doc("EnableAgent の対象の Agent の所有者の User が有効でない。先に所有者を稼働に戻すか、Agent を有効な所有者へ引き渡す。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_owner_inactive` である。")
model AgentOwnerInactiveError is ProblemDetails;


@doc("指定した OAuth2Client は、テナントの別の Agent にすでに結び付いている。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_client_already_bound` である。")
model AgentClientAlreadyBoundError is ProblemDetails;


@doc("Agent の登録に `kind` がない。種類は、Agent が人の承認なしにトークンを得られるかを実行時に決めるので、デフォルト値で代えない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:agent_kind_required` である。")
model AgentKindRequiredError is ProblemDetails;


@doc("`kind` が AgentKind の宣言する値でない。既知の値へ丸めると承認の要否を呼び出し元に代わって決めることになるので、丸めずに拒否する。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_agent_kind` である。")
model InvalidAgentKindError is ProblemDetails;


@doc("指定したメールアドレスの形式が正しくない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_email` である。")
model InvalidEmailError is ProblemDetails;


@doc("指定した新しいメールアドレスが現在のものと同じである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:email_unchanged` である。")
model EmailUnchangedError is ProblemDetails;


@doc("リクエストが指定したエクスポートの列が、対象の許可リストにない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_columns` である。")
model InvalidExportColumnsError is ProblemDetails;


@doc("エクスポートの対象の種類に対応していない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_target` である。")
model InvalidExportTargetError is ProblemDetails;


@doc("指定した状態の絞り込み条件は、エクスポートの対象に許されていない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_filter` である。")
model InvalidExportFilterError is ProblemDetails;


@doc("User の運用上のライフサイクルの属性。`status` が唯一の情報源で、状態の節の UserLifecycle の状態機械に一致する。以前の `disabled_at` と `deleted_at` は、`status` と `status_changed_at` に統合した。墓標は `status == deleted` である。")
model UserLifecycle {
  status: UserStatus = UserStatus.Active;
  @doc("現在の状態に達した時刻。詳細な遷移の履歴は監査イベントにある。")
  status_changed_at?: utcDateTime;
  last_login_at?: utcDateTime;
  password_changed_at?: utcDateTime;
  required_actions?: RequiredAction[];
}


@doc("一つの属性の値の直和型。`type` が示すフィールドだけを設定する。組み込みの OIDC の属性とテナント定義のカスタム属性が共有する。")
model AttributeValue {
  type: AttributeType;
  string?: string;
  number?: float64;
  boolean?: boolean;
  date?: string;
  string_array?: string[];
}

@doc("IdP の正規のユーザーの集約。プロトコルのサブジェクトはこの id から射影し、プロフィールの拡張はまばらな `attributes` に保持する。")
model User {
  @minLength(1)
  id: string;
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(100)
  preferred_username: string;
  password_hash?: string;
  @maxLength(200)
  name?: string;
  @maxLength(100)
  given_name?: string;
  @maxLength(100)
  family_name?: string;
  @maxLength(254)
  email?: string;
  email_verified: boolean = false;
  mfa_enrolled: boolean = false;
  roles: string[] = #[];
  lifecycle: UserLifecycle;
  attributes?: Record<AttributeValue>;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("管理者がユーザーを作成する JSON のリクエスト。初期パスワードには Authentication モジュールの PasswordPolicy を適用する。")
model AdminUserCreateRequest {
  @minLength(1)
  @maxLength(100)
  preferred_username: string;
  password: string;
  @maxLength(200)
  name?: string;
  @maxLength(254)
  email?: string;
  email_verified: boolean = false;
  roles: string[] = #[];
}


@doc("管理者がユーザーの変更可能なプロフィールとロールを更新する JSON のリクエスト。`id` と `password_hash` は変えない。")
model AdminUserUpdateRequest {
  @minLength(1)
  @maxLength(100)
  preferred_username?: string;
  @maxLength(200)
  name?: string;
  @maxLength(200)
  given_name?: string;
  @maxLength(200)
  family_name?: string;
  @maxLength(254)
  email?: string;
  email_verified?: boolean;
  roles?: string[];
  @doc("あれば属性の集まり全体を置き換え、テナントの TenantUserAttributeSchema に照らして検証する。セルフサービスの UpdateUserProfile と違い、管理者はエンドユーザーが編集できない属性も設定できる。")
  attributes?: Record<AttributeValue>;
}


@doc("管理 API のユーザーの一覧。削除したものを除き、`query` と `status` に左右されない、テナントのユーザーの正確な総数とともに返す。")
model AdminUserListResponse {
  users: AdminUserResponse[];
  @doc("ライフサイクルの状態が `deleted` でない、テナントの User の正確な数。`query` と `status` に左右されない。")
  total_users?: int64;
}


@doc("管理 API でのユーザーの表現。`password_hash` は意図して契約に含めない。")
model AdminUserResponse {
  id: string;
  @doc("属するテナント。")
  tenant_id: string;
  preferred_username: string;
  name?: string;
  email?: string;
  email_verified: boolean;
  mfa_enrolled: boolean;
  roles: string[];
  @doc("次のサインインで強制するアクション。")
  required_actions?: RequiredAction[];
  @doc("認証が最後に完了した時刻。")
  last_login_at?: utcDateTime;
  @doc("パスワードを最後に変更した時刻。")
  password_changed_at?: utcDateTime;
  @doc("ユーザーを無効にした時刻を保持する旧世代のフィールド。`User.lifecycle.status` と `status_changed_at` が唯一の情報源で、このフィールドは現在のレスポンスの後方互換のためだけに残る。この移行の範囲外の、個別の整理が必要な既存の隙間である。")
  disabled_at?: utcDateTime;
  created_at: utcDateTime;
  @doc("作成時は `created_at` と等しく、以降は管理操作のたびに更新する。")
  updated_at: utcDateTime;
}


@doc("SetUserRequiredAction の JSON のリクエスト。`action` は RequiredAction の列挙の送受信の値である。")
model AdminRequiredActionRequest {
  action: RequiredAction;
}


@doc("`/api/account/profile` が返すもの（セルフサービス）で、呼び出したユーザー自身の表示用の射影である。`attributes` は自身について読んでよいもの（`self_readable` または `claim_exposed`）だけを含み、`editable_attributes` は `editable_by_user=true` の定義を並べ、UI はそれでフォームを描画する。")
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
  attributes: Record<AttributeValue>;
  editable_attributes: UserAttributeDef[];
}


@doc("セルフサービスのプロフィールの更新のリクエスト。表示名と、`editable_by_user=true` の印を付けた属性だけを受け付ける。`attributes` はキーごとにマージし、指定したキーは作成または更新して、それ以外は変えない。管理者が管理する属性（状態、ロール、組織）には届かない。")
model AccountProfileUpdateRequest {
  name?: string;
  given_name?: string;
  family_name?: string;
  attributes?: Record<AttributeValue>;
}


@doc("CSV のユーザーのインポートのエンドポイントが、プレビューまたは適用を受け付けたときに返すもの。ジョブは非同期に動くので、進捗と結果は `id` で GetAdminUserImport を問い合わせる。")
model UserImportJobRef {
  @doc("ジョブの id。GetAdminUserImport の問い合わせと、プレビューからの適用の開始に使う。")
  id: string;
  @doc("Jobs モジュールのライフサイクルによるジョブの状態。")
  status: string;
  mode: UserImportMode;
}


@doc("CSV のユーザーのインポートのジョブの公開の表示。プレビューは副作用がなく、適用は同じテナントの成功したプレビューに `source_sha256` で結び付く。行のエラーは列と安定したコードを示し、値は示さない。パスワードもハッシュも読まず、保存しない。")
model UserImportJob {
  @doc("ジョブの id。")
  id: string;
  @doc("Jobs モジュールのライフサイクルによるジョブの状態。")
  status: string;
  mode: UserImportMode;
  @doc("ジョブが生成した件数とダイジェスト。ジョブが実行されるまでは省く。")
  result?: UserImportResult;
  @doc("行のエラーの一ページ。標準のカーソルのヘッダーでページングする。ジョブの実行中は空である。")
  errors?: UserImportRowError[];
}


@doc("完了した CSV のユーザーのインポートのジョブが生成した件数とダイジェスト。")
model UserImportResult {
  @doc("アップロードした CSV の SHA-256。適用は、同じダイジェストのプレビューを指定しなければならない。")
  source_sha256: string;
  @doc("適用の場合、開始の元になったプレビューのジョブ。")
  preview_job_id?: string;
  total_rows: int64;
  created_rows: int64;
  updated_rows: int64;
  unchanged_rows: int64;
  @doc("古いクライアントのための互換のフィールドで、created_rows + updated_rows + unchanged_rows である。新しいクライアントは操作ごとの件数を使う。")
  accepted_rows?: int64;
  rejected_rows: int64;
  @doc("ジョブがエラーの成果物を生成したときの、保存したその成果物への参照。")
  error_artifact_ref?: string;
  @doc("保存したエラーの成果物の SHA-256。")
  error_artifact_sha256?: string;
  @doc("このページに含まれる数ではなく、ジョブが生成した行のエラーの総数。")
  error_total: int64;
}


@doc("一つの CSV の行に対する、安全な検証または適用のエラー。値は返さず、列の名前と安定したコードだけを返す。制御面テナントの外で `roles` の列が予約済みのロール `system_admin` を新たに加える行は、プレビューでも適用時の再計画でも、その列の `invalid_roles` で拒否する。対象がすでに保持する値を送り直すだけの行は `unchanged` のままである。")
model UserImportRowError {
  row: int64;
  column?: string;
  code: string;
}


@doc("CSV のユーザーのインポートのジョブの段階。プレビューは読み取り専用の計画で、適用は成功したプレビューのペイロードを現在の状態に対して計画し直して適用する。")
enum UserImportMode {
  dry_run: "dry_run",
  preview: "preview",
  apply: "apply",
}


@doc("User と Group を問わず、すべての CSV のエクスポートとインポートが共有する設定可能な安全のポリシー。デフォルトは、データ行 100,000 行、ファイルあたり 64 MiB、フィールドあたり 64 KiB である。成功したエクスポートは常に同じポリシーの内に収まるので、そのままプレビューできる。上限はテナントの容量を表すものではなく、一つの非同期の成果物に対する運用上の境界である。")
model CsvTransferPolicy {
  max_rows: int64;
  max_bytes: int64;
  max_field_bytes: int64;
}


@doc("Group の CSV が作成または更新のほかに求める行の操作。書き込み専用の列で、Group のエクスポートは常に空で書き出すので、編集していないエクスポートを再び適用すると、すべての行を変更なしとして計画する。語彙は閉じており、Group には予約の状態がないので値は `delete` だけで、未知の値は既知の値へ丸めずに行を拒否する。")
enum GroupCsvLifecycleAction {
  delete: "delete",
}


@doc("CSV のグループのインポートのエンドポイントが、プレビューまたは適用を受け付けたときに返すもの。ジョブは非同期に動くので、進捗と結果は `id` で GetAdminGroupImport を問い合わせる。")
model GroupImportJobRef {
  @doc("ジョブの id。GetAdminGroupImport の問い合わせと、プレビューからの適用の開始に使う。")
  id: string;
  @doc("Jobs モジュールのライフサイクルによるジョブの状態。")
  status: string;
  mode: GroupImportMode;
}


@doc("CSV のグループのインポートのジョブの公開の表示。プレビューは副作用がなく、適用は同じテナントの成功したプレビューに `source_sha256` で結び付く。行のエラーは列と安定したコードを示し、値は示さない。")
model GroupImportJob {
  @doc("ジョブの id。")
  id: string;
  @doc("Jobs モジュールのライフサイクルによるジョブの状態。")
  status: string;
  mode: GroupImportMode;
  @doc("ジョブが生成した件数とダイジェスト。ジョブが実行されるまでは省く。")
  result?: GroupImportResult;
  @doc("行のエラーの一ページ。標準のカーソルのヘッダーでページングする。ジョブの実行中は空である。")
  errors?: GroupImportRowError[];
}


@doc("完了した CSV のグループのインポートのジョブが生成した件数とダイジェスト。削除は取り消せず、Group のすべてのメンバーシップへ連鎖するので、`deleted_rows` と `deleted_memberships` はほかの操作とは別に報告する。")
model GroupImportResult {
  @doc("アップロードした CSV の SHA-256。適用は、同じダイジェストのプレビューを指定しなければならない。")
  source_sha256: string;
  @doc("適用の場合、開始の元になったプレビューのジョブ。")
  preview_job_id?: string;
  total_rows: int64;
  created_rows: int64;
  updated_rows: int64;
  unchanged_rows: int64;
  @doc("Group の削除を求める行、または削除した行の数。")
  deleted_rows: int64;
  @doc("それらの削除が解除するメンバーシップの数。CSV の行ではなく、削除にともなう連鎖を数える。")
  deleted_memberships: int64;
  rejected_rows: int64;
  @doc("ジョブがエラーの成果物を生成したときの、保存したその成果物への参照。")
  error_artifact_ref?: string;
  @doc("保存したエラーの成果物の SHA-256。")
  error_artifact_sha256?: string;
  @doc("このページに含まれる数ではなく、ジョブが生成した行のエラーの総数。")
  error_total: int64;
}


@doc("一つの CSV のグループの行に対する、安全な検証または適用のエラー。値は返さず、列の名前と安定したコードだけを返す。制御面テナントの外で `roles` の列が予約済みのロール `system_admin` を新たに加える行は、プレビューでも適用時の再計画でも、その列の `invalid_roles` で拒否する。対象がすでに保持する値を送り直すだけの行は `unchanged` のままである。")
model GroupImportRowError {
  row: int64;
  column?: string;
  code: string;
}


@doc("CSV のグループのインポートのジョブの段階。プレビューは読み取り専用の計画で、適用は成功したプレビューのペイロードを現在の状態に対して計画し直して適用する。")
enum GroupImportMode {
  preview: "preview",
  apply: "apply",
}


@doc("Group の CSV の行が、一つの User についてグループに保持させたいメンバーシップ。メンバーシップの書式の唯一の変更の意図であり、語彙は閉じている。`present` はグループがまだ User を保持していなければ加え、`absent` はグループが保持している手動のメンバーシップを解除し、グループがすでに一致していればどちらも何もしない。空のセルと未知の値は、既知の値へ丸めずにそれぞれ行を拒否する。どちらへ丸めても、ファイルが下していない追加か削除かの判断をすることになるためである。メンバーシップのエクスポートはすべての行に `present` を書き出すので、編集していないエクスポートを再び適用すると、すべての行を変更なしとして計画する。ファイルが指定しない User には触れない。これは完全な権威のある同期ではないので、エクスポートを複数のファイルに分けても一括の削除にはならない。")
enum GroupMembershipCsvState {
  present: "present",
  absent: "absent",
}


@doc("CSV のグループのメンバーシップのインポートのエンドポイントが、プレビューまたは適用を受け付けたときに返すもの。ジョブは非同期に動くので、進捗と結果は `id` で GetAdminGroupMemberImport を問い合わせる。ジョブはパスで指定した一つのグループに結び付き、どちらのエンドポイントも別のグループに属するジョブを拒否する。")
model GroupMembershipImportJobRef {
  @doc("ジョブの id。GetAdminGroupMemberImport の問い合わせと、プレビューからの適用の開始に使う。")
  id: string;
  @doc("Jobs モジュールのライフサイクルによるジョブの状態。")
  status: string;
  mode: GroupMembershipImportMode;
}


@doc("CSV のグループのメンバーシップのインポートのジョブの公開の表示。プレビューは副作用がなく、適用は同じテナントと同じグループの成功したプレビューに `source_sha256` で結び付く。行のエラーは列と安定したコードを示し、値は示さない。")
model GroupMembershipImportJob {
  @doc("ジョブの id。")
  id: string;
  @doc("Jobs モジュールのライフサイクルによるジョブの状態。")
  status: string;
  mode: GroupMembershipImportMode;
  @doc("ジョブが生成した件数とダイジェスト。ジョブが実行されるまでは省く。")
  result?: GroupMembershipImportResult;
  @doc("行のエラーの一ページ。標準のカーソルのヘッダーでページングする。ジョブの実行中は空である。")
  errors?: GroupMembershipImportRowError[];
}


@doc("完了した CSV のグループのメンバーシップのインポートのジョブが生成した件数とダイジェスト。メンバーシップの解除はその User の有効ロールから Group のロールを外すので、`removed_rows` はほかの操作とは別に報告する。")
model GroupMembershipImportResult {
  @doc("アップロードした CSV の SHA-256。適用は、同じダイジェストのプレビューを指定しなければならない。")
  source_sha256: string;
  @doc("適用の場合、開始の元になったプレビューのジョブ。")
  preview_job_id?: string;
  @doc("このジョブのすべての行が限られる Group。リクエストのパスから得る。")
  group_id: string;
  total_rows: int64;
  @doc("グループが保持していないメンバーシップの追加を求める行、または追加した行の数。")
  added_rows: int64;
  @doc("グループが保持している手動のメンバーシップの解除を求める行、または解除した行の数。")
  removed_rows: int64;
  @doc("求めるメンバーシップにすでに一致している行の数。編集していないエクスポートはすべてこれになる。")
  unchanged_rows: int64;
  rejected_rows: int64;
  @doc("ジョブがエラーの成果物を生成したときの、保存したその成果物への参照。")
  error_artifact_ref?: string;
  @doc("保存したエラーの成果物の SHA-256。")
  error_artifact_sha256?: string;
  @doc("このページに含まれる数ではなく、ジョブが生成した行のエラーの総数。")
  error_total: int64;
}


@doc("一つの CSV のグループのメンバーシップの行に対する、安全な検証または適用のエラー。値は返さず、列の名前と安定したコードだけを返す。動的な対象の Group のような、ファイル全体に当てはまる拒否は、行のない一つのエラーとして報告する。")
model GroupMembershipImportRowError {
  row: int64;
  column?: string;
  code: string;
}


@doc("CSV のグループのメンバーシップのインポートのジョブの段階。プレビューは読み取り専用の計画で、適用は成功したプレビューのペイロードを現在のメンバーシップに対して計画し直して適用する。")
enum GroupMembershipImportMode {
  preview: "preview",
  apply: "apply",
}


@doc("CSV のエクスポートの対象のリソースの種類。列の定義は種類ごとの許可リストの中で閉じているので、レビューしていない属性が勝手に出ていくことはない。")
enum DataExportTargetKind {
  user: "user",
  group: "group",
  group_membership: "group_membership",
}


@doc("エクスポートの出力の形式。最初は CSV（RFC 4180）だけである。")
enum DataExportFormat {
  csv: "csv",
}


@doc("DataExportLifecycle に一致する、リソースのエクスポートのジョブの状態。`succeeded` はファイルをダウンロードできることを、`expired` はファイルそのものが TTL で消去され、メタデータと監査証跡だけが残ることを意味する。")
enum DataExportStatus {
  queued: "queued",
  running: "running",
  succeeded: "succeeded",
  failed: "failed",
  canceled: "canceled",
  expired: "expired",
}


@doc("エクスポートの種類ごとの許可リストの一つの列。`key` は安定した機械名で、CSV のヘッダーでもある。`label` は表示のためだけのもので、`pii` は個人情報を含むかどうかを示す。User の許可リストは、13 の組み込みのコアの列と、`attr:<key>`（OIDC の profile、phone、address のクレームと SCIM の enterprise の組織の属性から取った、すべてのテナントが共有する 27 の組み込みの拡張属性）と、`custom:<key>`（有効な TenantUserAttributeSchema からテナントごとに解決するテナント定義の属性）からなる。Group の許可リストは、インポートと互換の列（`id`、`name`、`description`、`email`、`membership_type`、`roles`、`dynamic_rule_expression`、`dynamic_rule_enabled`、`lifecycle_action`、`created_at`、`updated_at`）と、`custom:<key>`（TenantGroupAttributeSchema からテナントごとに解決するテナント定義の属性）からなる。Group には合わせる組み込みの属性の一覧がないので、`attr:<key>` に当たるものはない。編集していない Group のエクスポートは、そのままプレビューできる。`password_hash`、シークレット、トークン、リカバリーコード、MFA のシークレットのような機微な値は、どの許可リストにもない。")
model DataExportColumn {
  key: string;
  label: string;
  pii: boolean = false;
}


@doc("エクスポートを開始する入力。対象の種類はエンドポイントのパスから決まるので、ボディには含まない。出力の形式、列の選択（許可リストの部分集合）、一覧の画面から引き継いだ絞り込み条件を含む。許可リストの外の列のキーは拒否する。")
model DataExportRequest {
  format: DataExportFormat = DataExportFormat.csv;
  columns: string[];
  filter?: Record<string>;
}


@doc("リソースのエクスポートのジョブの公開の表示で、監査できる記録を兼ねる。CSV そのものも個人情報の値も含まず、対象の種類、選んだ列のキー、件数、バイト数、状態、有効期限、依頼者だけを含む。")
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


@doc("ダウンロードで返す完成したエクスポートのファイル。`content` は CSV そのもので個人情報を含みうるので、認可された管理者にだけ、成功してまだ期限切れでないジョブについてだけ、`Content-Disposition: attachment` で返す。")
model DataExportFile {
  filename: string;
  content_type: string;
  byte_size: int64;
  content: string;
}


@doc("エクスポートが失敗した理由。値は返さず、安定したコードだけを返す。")
model DataExportError {
  code: string;
}


@doc("管理者がリソースのエクスポートを求め、ジョブを受け付けた。CSV の値は含まず、対象の種類、選んだ列のキー、依頼者だけを含む。")
model DataExportRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
  target: DataExportTargetKind;
  requestedColumns: string[];
}


@doc("ワーカーがリソースのエクスポートの生成を始めた。")
model DataExportStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
}


@doc("リソースのエクスポートの生成が終わり、ファイルをダウンロードできるようになった。記録するのは行数とバイト数だけで、値は記録しない。")
model DataExportSucceeded {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
  totalRows: int64;
  byteSize: int64;
}


@doc("リソースのエクスポートの生成が失敗し、不完全なファイルはダウンロードできないまま終わった。")
model DataExportFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
  target: DataExportTargetKind;
  errorCode: string;
}


@doc("管理者が、終わる前のリソースのエクスポートを取り消した。")
model DataExportCanceled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
}


@doc("管理者が、完成したエクスポートのファイルをダウンロードした。個人情報を一括で取り出す高リスクの操作として監査証跡に残す。")
model DataExportDownloaded {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  exportId: string;
  target: DataExportTargetKind;
}


@doc("保持期間が過ぎ、エクスポートのファイルそのものを消去した。メタデータと監査証跡は残る。")
model DataExportExpired {
  occurredAt: utcDateTime;
  tenantId: string;
  exportId: string;
}


@doc("エンドユーザーのアカウントポータルのホームである `/api/account/summary` が返すもの。呼び出したユーザー自身のアカウントの概要で、管理シェルの AccountContext（`/api/auth/account`）とは別の契約である。ロールは含まず、ポータルのホームの概要カードのために、最後のサインイン、パスワードの変更時刻、MFA の状態、未処理の必須のアクションを返す。")
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


@doc("主たるメールアドレスの変更を申し出るリクエスト（セルフサービス）。新しいアドレスへ確認のリンクを送るだけで、ConfirmEmailChange が完了するまで `User.email` は変わらない。")
model AccountEmailChangeRequest {
  new_email: string;
}


@doc("新しいアドレスへ送った一回限りのリンクを確認するリクエスト（セルフサービス）。トークンそのものが新しいアドレスの所有の証明なので、パスワードリセットと同じ考え方で、認証済みのセッションは不要である。")
model AccountEmailChangeConfirm {
  @minLength(1)
  token: string;
}


@doc("エンドユーザーによる自身の個人データのエクスポート（GDPR 第 15 条、アクセス権）。現段階では同期的に生成し、現在の API から得られるプロフィールと接続済みのアプリケーション（同意）を集める。")
model AccountDataExport {
  exported_at: utcDateTime;
  profile: AccountProfileResponse;
  consents: AccountConsentResponse[];
}


@doc("User と Group のメンバーシップの関係。`group_id` と `user_id` で一意である。CSV のメンバーシップの行はまず `user_id` で User を解決し、なければ `preferred_username` を使うので、既知の `user_id` と別の User が保持するユーザー名の両方を含む行は、解決せずに拒否する。そこでは `source` と `created_at` は読み取り専用である。")
model GroupMember {
  group_id: string;
  user_id: string;
  source: GroupMembershipSource = GroupMembershipSource.manual;
  @doc("`dynamic_rule` のとき、評価した `DynamicGroupRule.version`。")
  rule_version?: int64;
  created_at: utcDateTime;
}


@doc("動的な Group と一対一に組になる制限付きの CEL の規則。式は真偽値を返し、未知の属性、許可する集合の外の関数、大きすぎる式は、いずれも保存できない。")
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


@doc("規則のバージョンの全体の再評価と、それが変えたメンバーシップの、監査できる結果。")
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


@doc("CEL の規則を作成または更新する入力。")
model DynamicGroupRuleRequest {
  @minLength(1)
  @maxLength(4096)
  expression: string;
}


@doc("保存していない CEL を最大 100 のユーザーに対して読み取り専用でプレビューするための入力。")
model DynamicGroupPreviewRequest {
  expression: string;
  user_ids: string[];
}


@doc("User ごとに、規則が一致したかどうかと、それが現在のメンバーシップとどう違うか。属性値は含まない。")
model DynamicGroupPreviewResult {
  user_id: string;
  matched: boolean;
  change: string;
  error_code?: string;
}


@doc("管理者がグループを作成する JSON のリクエスト。")
model GroupCreateRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  @maxLength(254)
  email?: string;
  attributes?: Record<AttributeValue>;
  roles?: string[];
  membership_type?: GroupMembershipType;
  dynamic_rule?: DynamicGroupRuleRequest;
}


@doc("管理者がグループの名前、説明、メールアドレス、属性、またはロールを更新する JSON のリクエスト。省いたフィールドは現在の値を保つ。")
model GroupUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  @maxLength(254)
  email?: string;
  attributes?: Record<AttributeValue>;
  roles?: string[];
}


@doc("グループの表現。管理 UI の一覧と詳細の表示は同じ表現を返す。")
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


@doc("管理 API のグループの一覧。")
model GroupListResponse {
  groups: GroupSummaryResponse[];
}


@doc("グループのメンバーの表現。")
model GroupMemberResponse {
  user_id: string;
  preferred_username: string;
  created_at: utcDateTime;
}


@doc("ユーザーの詳細の表示のような画面からメンバーシップを作るときに、`group_id` を指定する任意のリクエストボディ。")
model GroupAddMemberRequest {
  group_id: string;
}


@doc("指定した User が属するグループと、その有効ロールの内訳。管理 UI が有効ロールを追いやすいように、直接のロール（`user.roles`）とグループから継承したロールを、その和集合と並べて別々に返す。`effective_roles` は `direct_roles` と `group_roles` の和集合であり、User と Group の両方にまたがって導くので、User のモデルの制約ではなく、このレスポンスのフィールドとして計算して公開する。")
model UserGroupsResponse {
  groups: GroupSummaryResponse[];
  @doc("`user.roles`。")
  direct_roles: string[];
  @doc("属するグループから継承したロールの和集合。")
  group_roles: string[];
  @doc("`direct_roles` と `group_roles` の和集合。")
  effective_roles: string[];
}


@doc("テナントに属する、人でない ID の主体。Agent は自身の資格情報を保持せず、AgentCredentialBinding が結び付ける OAuth2Client を通じてトークンを得る。`owner_user_id` は責任を負う User を指し、発行は結果を `status` に固定せず、リクエストのたびにその所有者を解決する（REQ-OAUTH2-046）。`status` は運用の面で、`kind` は監督の面で発行を制限する。`active` 以外は新しいトークンを発行せず、`autonomous` 以外は人の承認を記録する経路でだけ発行する（REQ-OAUTH2-050）。どちらの制限もフェイルクローズで互いに独立しており、`active` で `supervised` の Agent にも承認が必要である。")
model Agent {
  @minLength(1)
  @maxLength(64)
  id: string;
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  kind: AgentKind;
  @doc("Agent に責任を負う User。その User をオフボーディングすると、`status` を変えずに新たな発行が止まる。")
  owner_user_id: string;
  status: AgentStatus = AgentStatus.Active;
  roles: string[] = #[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
  @doc("元に戻せる一時停止が有効になった時刻。`status` が `disabled` でなければ省く。")
  disabled_at?: utcDateTime;
  @doc("緊急停止が有効になった時刻。`status` が `killed` でなければ省く。一度設定すると消えない。")
  killed_at?: utcDateTime;
}


@doc("Agent と同じテナントの OAuth2Client の結び付きで、`agent_id` と `client_id` で一意である。Agent は結び付いた Client の資格情報でトークンを得る。`client_id` は、`tenant_id` が Agent のものと一致する OAuth2Client を指さなければならない（AgentReferencesStayTenantLocal）。一つの `client_id` を複数の Agent に結び付けることはできず、BindAgentCredential が事前条件で確かめる。")
model AgentCredentialBinding {
  agent_id: string;
  client_id: string;
  created_at: utcDateTime;
}


@doc("管理者が Agent を登録する JSON のリクエスト。`owner_user_id` を省くと呼び出し元が所有者になる。`kind` は必須でデフォルトはない。Agent がトークンを得るのに人の承認が必要かを決めるもので、どちらのデフォルト値も誤りになる。`supervised` では黙ってトークンを得られない Agent を登録してしまい、`autonomous` では省略によって監督なしの発行を許してしまう。")
model AgentRegisterRequest {
  @minLength(1)
  @maxLength(100)
  name: string;
  @maxLength(500)
  description?: string;
  @doc("必須である。省くと AgentKindRequiredError、未知の値は InvalidAgentKindError になり、どちらもデフォルトへは戻さない。")
  kind: AgentKind;
  @doc("省くと呼び出し元の id が所有者になる。")
  owner_user_id?: string;
  roles?: string[];
}


@doc("管理者が Agent の名前、説明、種類、`owner_user_id`、またはロールを更新する JSON のリクエスト。省いたフィールドは現在の値を保つ。`owner_user_id` の変更は所有権の移転である。")
model AgentUpdateRequest {
  @minLength(1)
  @maxLength(100)
  name?: string;
  @maxLength(500)
  description?: string;
  @doc("省くと現在の種類を保つ。未知の値は InvalidAgentKindError になる。")
  kind?: AgentKind;
  @doc("変更すると所有権を移転する（AgentOwnerChanged）。")
  owner_user_id?: string;
  roles?: string[];
}


@doc("OAuth2Client を Agent に結び付ける JSON のリクエスト。")
model AgentCredentialBindRequest {
  @minLength(1)
  client_id: string;
}


@doc("Agent の表現。管理 UI の一覧と詳細の表示は同じ表現を返す。")
model AgentSummaryResponse {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  kind: AgentKind;
  owner_user_id: string;
  status: AgentStatus;
  roles: string[];
  @doc("結び付いた OAuth2Client の id。")
  client_ids: string[];
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
  killed_at?: utcDateTime;
}


@doc("管理 API の Agent の一覧。")
model AgentListResponse {
  agents: AgentSummaryResponse[];
}


@doc("外部の IdP の ID をローカルのユーザーに連携した（アカウントの連携）。")
model FederationLinked {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  provider: string;
}


@doc("外部の IdP の ID とローカルのユーザーの連携を外した。")
model FederationUnlinked {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  provider: string;
}


@doc("エンドユーザーが主たるメールアドレスの変更を申し出て、新しいアドレスへ確認のリンクを送った（セルフサービス）。ConfirmEmailChange までメールアドレスそのものは変わらない。`newEmailHash` は SHA-256 なので、平文が監査ログに届くことはない。")
model EmailChangeRequested {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
  @doc("新しいメールアドレスの SHA-256 の 16 進数。")
  newEmailHash: string;
}


@doc("新しいアドレスへ送った一回限りのリンクを確認し、主たるメールアドレスを確定した（セルフサービス）。`email_verified` が true になり、`verify_email` の必須のアクションがあれば自動で解除する。")
model EmailChanged {
  occurredAt: utcDateTime;
  @doc("対象のユーザーが属するテナント。")
  tenantId: string;
  userId: string;
}


@doc("管理者がユーザーを作成した。初期パスワードも `password_hash` も監査イベントには含めない。")
model UserCreated {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者がユーザーのプロフィールまたはロールを更新した。")
model UserUpdated {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  changedFields: string[];
}


@doc("管理者がユーザーを無効にし、そのユーザーの新しいサインインと既存のセッションの利用の両方を拒否した。")
model UserDisabled {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者が、無効にしたユーザーを再び有効にした。")
model UserEnabled {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者がユーザーを論理削除した。状態は `pending_deletion` へ移り、個人情報、Consent、RefreshToken、Session は残った。`reason` は管理者が残せる自由入力のメモである。")
model UserSoftDeleted {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  reason?: string;
}


@doc("管理者が `pending_deletion` のユーザーを復元した。状態は `active` に戻り、個人情報と資格情報を保ったまま再びサインインできる。")
model UserRestored {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者による、または猶予期間を過ぎた後の自動の完全削除によって、ユーザーを完全削除した。個人情報は匿名化し、関連する集約は連鎖して削除した。`reason` は管理者が残せる自由入力のメモで、自動のときは `auto_purge` である。")
model UserDeleted {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
  reason?: string;
}


@doc("管理者が、ユーザーに次のサインインで強制するアクション（RequiredAction）を与えた。`action` は列挙値なので、平文で監査証跡に残して安全である。")
model UserRequiredActionSet {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  actorUserId: string;
  targetUserId: string;
}


@doc("強制するアクションを解除した。管理者が明示的に解除したときのほか、ユーザーが自身のパスワードを変更して `update_password` が自動で解除されたときにも発行し、その場合の操作した主体の id は対象自身のものである。")
model UserRequiredActionCleared {
  occurredAt: utcDateTime;
  @doc("操作が行われたテナント。操作した主体と対象の両方に一致する。")
  tenantId: string;
  @doc("解除した主体。自動の解除では対象自身である。")
  actorUserId: string;
  targetUserId: string;
}


@doc("管理者がグループを作成した。")
model GroupCreated {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
}


@doc("管理者がグループの名前、説明、またはロールを更新した。")
model GroupUpdated {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  changedFields: string[];
}


@doc("管理者がグループを削除した。メンバーシップは連鎖して解除し、それぞれ個別の GroupMemberRemoved を発行した。")
model GroupDeleted {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
}


@doc("管理者が User をグループのメンバーにした。すでにメンバーだったときは発行しないので、操作は冪等である。")
model GroupMemberAdded {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  userId: string;
}


@doc("管理者が User をグループから外したか、グループの削除によってメンバーシップが解除された。メンバーでなかったときは発行しないので、操作は冪等である。")
model GroupMemberRemoved {
  occurredAt: utcDateTime;
  @doc("グループが属するテナント。")
  tenantId: string;
  actorUserId: string;
  groupId: string;
  userId: string;
}


@doc("管理者が、動的グループの CEL の規則を新しいバージョンへ更新した。")
model DynamicGroupRuleUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("管理者が検証済みの規則を有効にし、全体の再評価を予定した。")
model DynamicGroupRuleEnabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("管理者が規則を無効にし、古い動的なメンバーシップを即座に無効にした。")
model DynamicGroupRuleDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("規則のバージョンの全体の再評価が終わった。属性値は含まず、変わったメンバーシップの数だけを含む。")
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


@doc("ワーカーが動的なメンバーシップの全体の再評価を始めた。")
model DynamicMembershipEvaluationStarted {
  occurredAt: utcDateTime;
  tenantId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("動的なメンバーシップの全体の再評価が失敗し、規則のバージョンはフェイルクローズのままになった。")
model DynamicMembershipEvaluationFailed {
  occurredAt: utcDateTime;
  tenantId: string;
  groupId: string;
  ruleVersion: int64;
}


@doc("管理者が Agent を登録した。")
model AgentRegistered {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が Agent の名前、説明、種類、またはロールを更新した。")
model AgentUpdated {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  changedFields: string[];
}


@doc("管理者が Agent を元に戻せる形で一時停止した。")
model AgentDisabled {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が一時停止した Agent を稼働に戻した。")
model AgentEnabled {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が Agent を強制停止した。一方向の終端の操作で、元には戻せない。")
model AgentKilled {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が Agent を削除した。そのバインディングは連鎖して解除した。")
model AgentDeleted {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
}


@doc("管理者が Agent の所有者（`owner_user_id`）を移転した。")
model AgentOwnerChanged {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  previousOwnerUserId: string;
  newOwnerUserId: string;
}


@doc("管理者が OAuth2Client を Agent に結び付けた。")
model AgentCredentialBound {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  clientId: string;
}


@doc("管理者が OAuth2Client と Agent の結び付きを外した。")
model AgentCredentialUnbound {
  occurredAt: utcDateTime;
  @doc("Agent が属するテナント。")
  tenantId: string;
  actorUserId: string;
  agentId: string;
  clientId: string;
}


@doc("リクエストの対象の User がこのテナントに存在しない。他のテナントの User は存在しないものとして扱う。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:user_not_found` である。")
model UserNotFoundError is ProblemDetails;


@doc("テナントはすでに同時に実行してよい数のエクスポートのジョブを実行しているので、もう一つは開始できない。実行中のジョブが終わった後に再試行すれば成功するので、レスポンスは QuotaExceededError の 422 ではなく 429 である。RFC 9457 の Problem Details で、`type` は固有のコードの `urn:idmagic:error:active_job_quota_exceeded` である。同時実行の上限とテナントのリソースのクォータは別の概念であり、`quota_exceeded` しか見ないクライアントは再試行がいつか成功するかを判断できないためである。")
model ActiveJobQuotaExceededError is ProblemDetails;


@doc("リクエストが指定したユーザーのインポートのジョブが、このテナントに存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:user_import_not_found` である。")
model UserImportNotFoundError is ProblemDetails;


@doc("ユーザーのインポートのサービスが設定されていないので、インポートを開始することも読むこともできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:user_import_unavailable` である。")
model UserImportUnavailableError is ProblemDetails;


@doc("リクエストが指定したグループのインポートのジョブが、このテナントに存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:group_import_not_found` である。")
model GroupImportNotFoundError is ProblemDetails;


@doc("グループのインポートのサービスが設定されていないので、インポートを開始することも読むこともできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:group_import_unavailable` である。")
model GroupImportUnavailableError is ProblemDetails;


@doc("リクエストが指定したグループのメンバーシップのインポートのジョブが、このテナントに存在しないか、パスの Group とは別の Group に属している。一つのグループが別のグループのジョブの id を漏らさないように、二つを一つの答えにまとめる。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:group_membership_import_not_found` である。")
model GroupMembershipImportNotFoundError is ProblemDetails;


@doc("グループのメンバーシップのインポートのサービスが設定されていないので、インポートを開始することも読むこともできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:group_membership_import_unavailable` である。")
model GroupMembershipImportUnavailableError is ProblemDetails;

}
