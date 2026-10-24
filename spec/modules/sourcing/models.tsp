namespace IdMagic.Contract {

@doc("RFC 7644 の 3.12 節の SCIM のプロトコルエラーのレスポンス。SCIM は形を標準が定める接点なので Problem Details を返さず、ボディは `application/json` である。")
model ScimProtocolError {
  @doc("常に `urn:ietf:params:scim:api:messages:2.0:Error` の一要素だけである。")
  schemas: string[];
  @doc("HTTP のステータスコード。10 進数の文字列で表す。")
  status: string;
  @doc("発生した事象の、人が読む説明。常に英語で、ローカライズしない。")
  detail?: string;
  @doc("RFC 7644 の 3.12 節の表 9 にある SCIM の詳細エラーのキーワード。`invalidFilter`、`invalidValue`、`uniqueness` など。当てはまるキーワードがないときは省く。")
  scimType?: string;
}


@doc("SCIM 2.0 の id を、それが表す内部の User（`sub`）に対応付ける。")
model ScimUserRef {
  tenant_id: string;
  scim_id: string;
  user_id: string;
}


@doc("SCIM 2.0 の id を、それが表す内部の Group（グループの id）に対応付ける。")
model ScimGroupRef {
  tenant_id: string;
  scim_id: string;
  group_id: string;
}


@doc("/Users と /Groups のエンドポイントへ送り、そこから返る SCIM のリソース（RFC 7643）。含まれる属性はリソースが `schemas` で宣言するスキーマによって決まるので、契約は一つの属性の集合に固定せず、開いた JSON オブジェクトとして記録する。サーバーが従う属性の定義は GetScimSchemas が報告する。IdMagic は User と Group のコアスキーマと、User の enterprise 拡張に対応する。")
model ScimResource is Record<unknown>;


@doc("RFC 7644 の 3.4.2 節の SCIM の検索結果のエンベロープ。")
model ScimListResponse {
  @doc("常に `urn:ietf:params:scim:api:messages:2.0:ListResponse` の一要素だけである。")
  schemas: string[];
  @doc("このページに含まれる数ではなく、検索に一致したリソースの総数。")
  totalResults: int32;
  @doc("このページの最初のリソースの、1 から数えた位置。")
  startIndex: int32;
  @doc("このページに含まれるリソースの数。")
  itemsPerPage: int32;
  @doc("一致したリソースのページ。RFC 7644 の 3.4.2 節の綴りに合わせて大文字で始める。")
  Resources: ScimResource[];
}


@doc("SCIM の任意の機能の一つに対応しているかどうか（RFC 7643 の 5 節）。")
model ScimFeatureSupport {
  supported: boolean;
}


@doc("ServiceProviderConfig が宣言する、一括操作への対応（RFC 7643 の 5 節）。")
model ScimBulkConfig {
  supported: boolean;
  maxOperations: int32;
  maxPayloadSize: int32;
}


@doc("ServiceProviderConfig が宣言する、絞り込みへの対応（RFC 7643 の 5 節）。")
model ScimFilterConfig {
  supported: boolean;
  @doc("検索の `count` にかかわらず、サーバーが返すページの最大の大きさ。")
  maxResults: int32;
}


@doc("サービスプロバイダーが受け付ける認証方式の一つ（RFC 7643 の 5 節）。")
model ScimAuthenticationScheme {
  name: string;
  @doc("方式の説明。常に英語で、ローカライズしない。")
  description: string;
  @doc("方式を定める仕様の場所。")
  specUri?: string;
  @doc("方式の種類。`oauthbearertoken` など。")
  type: string;
}


@doc("RFC 7643 の 5 節の SCIM のサービスプロバイダーの設定。このデプロイメントが対応する任意の機能を報告する。")
model ScimServiceProviderConfig {
  @doc("常に `urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig` の一要素だけである。")
  schemas: string[];
  documentationUri?: string;
  patch: ScimFeatureSupport;
  bulk: ScimBulkConfig;
  filter: ScimFilterConfig;
  changePassword: ScimFeatureSupport;
  sort: ScimFeatureSupport;
  etag: ScimFeatureSupport;
  authenticationSchemes: ScimAuthenticationScheme[];
}


@doc("リソースタイプがコアスキーマの外に拡張するスキーマ（RFC 7643 の 6 節）。User の enterprise 拡張など。")
model ScimSchemaExtension {
  @doc("拡張するスキーマの URN。")
  schema: string;
  @doc("このリソースタイプのリソースがこの拡張を含まなければならないかどうか。")
  required: boolean;
}


@doc("サービスプロバイダーが公開するリソースタイプの一つ（RFC 7643 の 6 節）。")
model ScimResourceType {
  @doc("常に `urn:ietf:params:scim:schemas:core:2.0:ResourceType` の一要素だけである。")
  schemas: string[];
  id: string;
  name: string;
  @doc("リソースタイプを提供するパス。SCIM のベース URL からの相対パスである。")
  endpoint: string;
  @doc("リソースタイプの説明。常に英語で、ローカライズしない。")
  description?: string;
  @doc("このリソースタイプのコアスキーマの URN。")
  schema: string;
  schemaExtensions?: ScimSchemaExtension[];
}


@doc("SCIM のスキーマの中の属性の定義の一つ（RFC 7643 の 7 節）。")
model ScimSchemaAttribute {
  name: string;
  @doc("属性の型。`string`、`boolean`、`reference`、`complex` など。")
  type: string;
  multiValued: boolean;
  @doc("属性が保持するものの説明。常に英語で、ローカライズしない。")
  description?: string;
  required: boolean;
  caseExact: boolean;
  @doc("`readOnly`、`readWrite`、`immutable`、`writeOnly` のいずれか。")
  mutability: string;
  @doc("`always`、`never`、`default`、`request` のいずれか。")
  returned: string;
  @doc("`none`、`server`、`global` のいずれか。")
  uniqueness: string;
  @doc("`reference` の属性で、参照が指してよいリソースタイプ。")
  referenceTypes?: string[];
  @doc("`complex` の属性で、副属性の定義。")
  subAttributes?: ScimSchemaAttribute[];
}


@doc("サービスプロバイダーが従うスキーマの一つ（RFC 7643 の 7 節）。IdMagic は User のコアスキーマ、User の enterprise 拡張、Group のコアスキーマを報告する。")
model ScimSchema {
  @doc("常に `urn:ietf:params:scim:schemas:core:2.0:Schema` の一要素だけである。")
  schemas: string[];
  @doc("この文書が定義するスキーマの URN。")
  id: string;
  name: string;
  @doc("スキーマの用途の説明。常に英語で、ローカライズしない。")
  description?: string;
  attributes: ScimSchemaAttribute[];
}


@doc("Directory Connector の状態。`awaiting_enrollment` は公開鍵の登録を待ち、`active` は登録した鍵で呼び出せ、`disabled` は管理者が無効化した終端である。")
enum DirectoryConnectorStatus {
  awaiting_enrollment,
  active,
  disabled,
}


@doc("閉域の AD または LDAP から取り込む Directory Connector の登録。登録トークンもそのハッシュも含まない。")
model DirectoryConnector {
  @doc("IdMagic が採番する ID。Connector の JWT の `iss` と `sub` になる。")
  id: string;
  @doc("管理者が付けた名前。")
  display_name: string;
  status: DirectoryConnectorStatus;
  created_at: utcDateTime;
  @doc("`awaiting_enrollment` の間だけ、未使用の登録トークンの有効期限を返す。")
  enrollment_expires_at?: utcDateTime;
  @doc("公開鍵を登録した時刻。")
  enrolled_at?: utcDateTime;
  @doc("登録した公開鍵の RFC 7638 のサムプリント。`active` の間だけ返す。")
  key_id?: string;
  @doc("Connector が最後に認証された呼び出しを行った時刻。")
  last_seen_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("Directory Connector を作成するリクエスト。")
model CreateDirectoryConnectorRequest {
  @doc("前後の空白を除いて保存する。空白だけの値は拒否する。")
  @maxLength(200)
  display_name: string;
}


@doc("登録トークンを一度だけ返す応答。トークンは再表示できない。")
model DirectoryConnectorEnrollmentGrant {
  connector: DirectoryConnector;
  @doc("`idmagic_dce_` で始まる、一回だけ使える登録トークン。発行の 1 時間後に期限が切れる。")
  enrollment_token: string;
}


@doc("Connector が公開鍵を登録するリクエスト。")
model EnrollDirectoryConnectorRequest {
  @doc("管理者から受け取った登録トークン。")
  enrollment_token: string;
  @doc("Connector が手元で作った鍵対の公開鍵の JWK（RFC 7517）。P-256 の EC 鍵か 2048 ビット以上の RSA 鍵であり、秘密鍵の成分を含まない。")
  jwk: Record<unknown>;
}


@doc("登録が済んだ Connector が、以後の JWT に使う値。")
model DirectoryConnectorEnrollment {
  @doc("JWT の `iss` と `sub` に使う Connector の ID。")
  connector_id: string;
  @doc("JWT のヘッダーの `kid` に使える、登録した公開鍵の RFC 7638 のサムプリント。")
  key_id: string;
  @doc("JWT の `aud` に使う値。テナントの発行者識別子に `/directory-connector/v1` を続けたものである。")
  audience: string;
}


@doc("Directory Connector が見つからない。存在しないか、別のテナントのものである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:directory_connector_not_found` である。")
model DirectoryConnectorNotFoundError is ProblemDetails;


@doc("無効化した Directory Connector には登録トークンを再発行できない。新しい Connector を作る。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:directory_connector_disabled` である。")
model DirectoryConnectorDisabledError is ProblemDetails;


@doc("登録トークンまたは Connector の JWT を認証できない。トークンの不一致、期限切れ、使用済み、`jti` の再利用、署名、`aud`、無効化した Connector を区別しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_token` である。")
model DirectoryConnectorInvalidTokenError is ProblemDetails;


@doc("テナントの管理者が Directory Connector を作成した。登録トークンを含まない。")
model DirectoryConnectorCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  connectorId: string;
}


@doc("Directory Connector が公開鍵を登録し、呼び出せるようになった。")
model DirectoryConnectorEnrolled {
  occurredAt: utcDateTime;
  tenantId: string;
  connectorId: string;
  @doc("登録した公開鍵の RFC 7638 のサムプリント。")
  keyId: string;
}


@doc("テナントの管理者が登録トークンを再発行し、登録済みの公開鍵を破棄した。登録トークンを含まない。")
model DirectoryConnectorEnrollmentReissued {
  occurredAt: utcDateTime;
  tenantId: string;
  connectorId: string;
}


@doc("テナントの管理者が Directory Connector を無効化した。")
model DirectoryConnectorDisabled {
  occurredAt: utcDateTime;
  tenantId: string;
  connectorId: string;
}


@doc("取り込みの実行の種類。`full` は取り込み元の全件を送る完全な再同期、`incremental` は前回からの差分である。")
enum IngestionRunMode {
  full,
  incremental,
}


@doc("取り込みの実行の状態。`running` はバッチを受け付け、`completed` は Connector が完了を報告し、`superseded` は同じ Connector の新しい実行に打ち切られた。")
enum IngestionRunStatus {
  running,
  completed,
  superseded,
}


@doc("Directory Connector が取り込みの実行を始めるリクエスト。")
model StartIngestionRunRequest {
  mode: IngestionRunMode;
}


@doc("取り込みの実行。")
model IngestionRun {
  id: string;
  mode: IngestionRunMode;
  status: IngestionRunStatus;
  started_at: utcDateTime;
  completed_at?: utcDateTime;
}


@doc("Directory Connector が送る User の項目。属性は Connector が正規化済みである。")
model DirectoryUserItem {
  @doc("`objectGUID` や `entryUUID` のような、名前の変更で変わらない ID。バッチの中で重複できない。")
  @minLength(1)
  @maxLength(256)
  external_id: string;
  @doc("`true` なら User の削除を予約する。ほかの属性を読まない。")
  deleted?: boolean = false;
  @doc("`false` なら User を `disabled` にする。")
  enabled?: boolean = true;
  @doc("削除でない項目では必須である。")
  preferred_username?: string;
  name?: string;
  given_name?: string;
  family_name?: string;
  email?: string;
  employee_number?: string;
  department?: string;
}


@doc("User の項目のバッチ。")
model DirectoryUserBatchRequest {
  @doc("Connector が付けるバッチの ID。同じ実行で同じ ID と同じ項目の再送は、最初の結果を返す。")
  @minLength(1)
  @maxLength(128)
  batch_id: string;
  @minItems(1)
  @maxItems(100)
  items: DirectoryUserItem[];
}


@doc("Directory Connector が送る AD のコンピューターアカウントの項目。属性は Connector が正規化済みである。")
model DirectoryComputerItem {
  @doc("`objectGUID` の 16 バイトを標準の base64 にした値。バッチの中で重複できない。")
  @minLength(1)
  @maxLength(256)
  external_id: string;
  @doc("`false` ならコンピューターを無効にする。")
  enabled?: boolean = true;
  @doc("`sAMAccountName`。`$` で終わる。")
  @minLength(2)
  @maxLength(256)
  @pattern("\\$$")
  sam_account_name: string;
  @doc("`objectSid` の文字列表現（`S-1-5-21-...`）。")
  @maxLength(184)
  @pattern("^S-1-[0-9]+(-[0-9]+)+$")
  object_sid: string;
  @doc("コンピューターのドメインの DNS 名。Kerberos の主体名 `<sam_account_name>@<domain>` の realm になる。")
  @minLength(1)
  @maxLength(253)
  domain: string;
  @maxLength(253)
  dns_host_name?: string;
}


@doc("コンピューターの項目のバッチ。バッチの ID は同じ実行の User のバッチと同じ空間を使う。")
model DirectoryComputerBatchRequest {
  @doc("Connector が付けるバッチの ID。同じ実行で同じ ID と同じ項目の再送は、最初の結果を返す。")
  @minLength(1)
  @maxLength(128)
  batch_id: string;
  @minItems(1)
  @maxItems(100)
  items: DirectoryComputerItem[];
}


@doc("項目の適用の結果。")
enum DirectoryItemOutcome {
  created,
  updated,
  unchanged,
  deleted,
  failed,
}


@doc("`failed` の項目のエラー。")
enum DirectoryItemError {
  username_conflict,
  email_conflict,
  invalid_attribute,
  quota_exceeded,
  user_deleted,
  @doc("別の Connector の有効なコンピューターが同じ `object_sid` か主体名を持つ。")
  computer_conflict,
}


@doc("一つの項目の結果。")
model DirectoryItemResult {
  external_id: string;
  outcome: DirectoryItemOutcome;
  @doc("`outcome` が `failed` のときだけ返す。")
  error?: DirectoryItemError;
}


@doc("バッチの項目ごとの結果。項目を送られた順に並ぶ。")
model DirectoryUserBatchResult {
  batch_id: string;
  results: DirectoryItemResult[];
}


@doc("Directory Connector が取り込みの実行の完了を報告するリクエスト。")
model CompleteIngestionRunRequest {
  @doc("この実行で送ったバッチの項目の数の和。記録した数と一致しなければ完了を拒否する。")
  @minValue(0)
  item_count: int32;
}


@doc("取り込みの実行の結果ごとの項目の数。")
model IngestionRunCounts {
  batches: int32;
  items: int32;
  created: int32;
  updated: int32;
  unchanged: int32;
  deleted: int32;
  failed: int32;
  @doc("完全な再同期の完了で無効にした User の数。")
  disabled: int32;
  @doc("完全な再同期の完了で無効にしたコンピューターの数。")
  disabled_computers: int32;
}


@doc("管理者が読む取り込みの実行の要約。")
model IngestionRunSummary {
  ...IngestionRun;
  counts: IngestionRunCounts;
}


@doc("`failed` の項目。")
model IngestionRunFailure {
  batch_id: string;
  external_id: string;
  error: DirectoryItemError;
}


@doc("管理者が読む取り込みの実行の詳細。")
model IngestionRunDetail {
  ...IngestionRunSummary;
  @doc("バッチを記録した順に最大 100 件。")
  failures: IngestionRunFailure[];
}


@doc("取り込みの実行が見つからない。存在しないか、別の Connector か別のテナントのものである。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ingestion_run_not_found` である。")
model IngestionRunNotFoundError is ProblemDetails;


@doc("取り込みの実行は `completed` か `superseded` であり、記録していないバッチと `superseded` の完了を受け付けない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ingestion_run_closed` である。")
model IngestionRunClosedError is ProblemDetails;


@doc("報告した項目の数が、記録したバッチの項目の数の和と一致しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:ingestion_run_incomplete` である。")
model IngestionRunIncompleteError is ProblemDetails;


@doc("同じ ID で記録したバッチと項目が異なる。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:batch_conflict` である。")
model BatchConflictError is ProblemDetails;

}
