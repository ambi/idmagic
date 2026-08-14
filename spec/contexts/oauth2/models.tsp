namespace IdMagic.Contract {

@doc("end-user の account portal が返す「接続済みアプリ」1 件 (self-service)。\n自分の sub に固定されるため sub / tenant_id は出さない。client_id・表示名・付与 scope・\n付与/失効日時を表示用に返す。")
model AccountConsentResponse {
  client_id: string;
  @doc("client_id を解決した表示名。OAuth2Client.client_name → Application\nカタログ名の順で解決し、いずれも無ければ client_id をそのまま返す。")
  client_name: string;
  scopes: string[];
  state: ConsentState;
  granted_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("EndpointRateLimitPolicy の閾値を超えた。login throttle (per-account/per-IP の失敗回数ベース) とは別の、要求到達時ベースの汎用 endpoint rate limit。fail-closed (共有カウンタストア到達不能時も拒否)。")
model RateLimitedError {

}


@doc("SPA の認証操作後に表示する次画面またはクライアント redirect URI。")
model BrowserFlowResponse {
  next?: string;
  redirect_to?: url;
}


@doc("認可レスポンスのパラメータをクライアントへ返す方式 (OIDC Multiple Response Types §3)。")
enum ResponseMode {
  Query: "Query",
  FormPost: "FormPost",
}


@doc("/introspect / /revoke の token_type_hint で指定されるトークン種別。")
enum TokenType {
  AccessToken: "AccessToken",
  RefreshToken: "RefreshToken",
}


@doc("トークンを所有者にバインドする方式 (RFC 7800)。Dpop は DPoP 公開鍵サムプリント、Mtls は mTLS クライアント証明書サムプリント。")
enum SenderConstraintKind {
  Dpop: "Dpop",
  Mtls: "Mtls",
}


@doc("発行された AccessToken に適用されているセンダー制約。None は Bearer トークン。")
enum AccessTokenSenderConstraint {
  None: "None",
  Dpop: "Dpop",
  Mtls: "Mtls",
}


@doc("PKCE (RFC 7636) の code_challenge ハッシュ方式。本アプリでは S256 のみ許容（plain は脆弱なため拒否）。")
enum CodeChallengeMethod {
  S256: "S256",
}


@doc("AuthorizationCodeFlow 状態機械の取りうる状態列挙。状態機械セクションも参照。")
enum AuthorizationCodeState {
  Received: "Received",
  AuthenticationPending: "AuthenticationPending",
  Authenticated: "Authenticated",
  ConsentPending: "ConsentPending",
  Consented: "Consented",
  CodeIssued: "CodeIssued",
  Exchanged: "Exchanged",
  Rejected: "Rejected",
  Expired: "Expired",
}


@doc("DeviceCodeFlow (RFC 8628) 状態機械の取りうる状態列挙。状態機械セクションも参照。")
enum DeviceCodeState {
  Issued: "Issued",
  UserCodeEntered: "UserCodeEntered",
  Approved: "Approved",
  Denied: "Denied",
  Exchanged: "Exchanged",
  Expired: "Expired",
}


@doc("ApprovalRequestLifecycle 状態機械の取りうる状態列挙。CIBA の輸送語彙ではなく「前提条件がまだ充足されていない」一般形として持つ。Consumed は Approved からのみ到達する、一度きりのトークン化を表す終端。状態機械セクションも参照。")
enum ApprovalRequestState {
  Pending: "Pending",
  Approved: "Approved",
  Denied: "Denied",
  Expired: "Expired",
  Consumed: "Consumed",
}


@doc("RefreshTokenLifecycle 状態機械の取りうる状態列挙。Rotated は子トークンへ消費済み、Revoked は失効済み、Expired は期限切れ。")
enum RefreshTokenState {
  Active: "Active",
  Rotated: "Rotated",
  Revoked: "Revoked",
  Expired: "Expired",
}


@doc("AuthorizationCodeRecordLifecycle 状態機械の取りうる状態列挙。AuthorizationCodeFlow とは別軸でコード本体のライフサイクルを表す。")
enum AuthorizationCodeRecordState {
  Issued: "Issued",
  Redeemed: "Redeemed",
  Expired: "Expired",
}


@doc("ConsentLifecycle 状態機械の取りうる状態列挙。GDPR Art.7(3) 撤回権により Granted → Revoked が可能。")
enum ConsentState {
  Granted: "Granted",
  Revoked: "Revoked",
  Expired: "Expired",
}


@doc("PARRecordLifecycle 状態機械の取りうる状態列挙。Stored から /authorize での参照で Used、期限切れで Expired。")
enum PARRecordState {
  Stored: "Stored",
  Used: "Used",
  Expired: "Expired",
}


@doc("authorization_details の登録スキーマで各フィールドが担うダウンスコープ半順序。 set は集合包含 (要求 ⊆ 同意/元)、at_most は単調減少 (要求 ≤ 同意/元)、enum は許可列挙からの完全一致、 exact は不透明値の完全一致。判定不能はすべて拒否側へ倒す (fail-closed)。")
enum AuthorizationDetailFieldSemantics {
  set: "set",
  at_most: "at_most",
  enum_: "enum",
  exact: "exact",
}


@doc("AuthorizationDetailType の運用状態列挙。Enabled のみ受理対象、Disabled は新規要求を拒否する (既存トークンは失効まで有効)。")
enum AuthorizationDetailTypeState {
  Enabled: "Enabled",
  Disabled: "Disabled",
}


@doc("OAuth 2.0 / OIDC / DPoP / FAPI / RFC 8628 で規定された標準エラーコード一式。HTTP レスポンスの error フィールド値となる。")
enum OAuthErrorCode {
  InvalidRequest: "InvalidRequest",
  InvalidClient: "InvalidClient",
  InvalidGrant: "InvalidGrant",
  UnauthorizedClient: "UnauthorizedClient",
  UnsupportedGrantType: "UnsupportedGrantType",
  InvalidScope: "InvalidScope",
  InvalidToken: "InvalidToken",
  ServerError: "ServerError",
  InvalidDpopProof: "InvalidDpopProof",
  AccessDenied: "AccessDenied",
  ExpiredToken: "ExpiredToken",
  SlowDown: "SlowDown",
  InsufficientScope: "InsufficientScope",
  AuthorizationPending: "AuthorizationPending",
  InvalidTarget: "InvalidTarget",
}


@doc("RFC 7591 + FAPI 2.0 拡張のクライアント。application_id が NULL の catalog 外 client も許容する。非 NULL の場合は同一 tenant の oidc Application を参照し、Application 1件に最大1 client だけが紐づく。")
model OAuth2Client {
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(128)
  client_id: string;
  @doc("対応する Application の ID。通常 CRUD では変更せず、Application 一括作成・削除だけが関係を管理する。")
  application_id?: unknown;
  client_secret_hash?: string;
  @maxLength(200)
  client_name?: string;
  client_type: ClientType;
  @doc("redirect 系グラント (authorization_code) のとき必須。client_credentials のみの M2M クライアントは持たない (RFC 6749 §3.1.2)。")
  redirect_uris?: url[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  jwks_uri?: url;
  jwks?: unknown;
  tls_client_auth_subject_dn?: string;
  id_token_signed_response_alg: SignatureAlgorithm = SignatureAlgorithm.PS256;
  require_pushed_authorization_requests: boolean = false;
  require_pkce?: boolean;
  dpop_bound_access_tokens: boolean = false;
  fapi_profile: FapiProfile = FapiProfile.None;
  first_party: boolean = false;
  @doc("OpenID Connect Back-Channel Logout 1.0 の通知先。設定時、session revoke ごとに\n署名済み logout token をこの URI へ POST する。")
  backchannel_logout_uri?: url;
  @doc("RP が logout token に sid claim を要求するか。true の場合 sid 欠落の対象セッションへは配送しない (browser session を持たない発行のため送るべき対象が無い)。")
  backchannel_logout_session_required: boolean = false;
  @doc("OpenID Connect Front-Channel Logout 1.0 の通知先。/end_session 応答にこの URI への\niframe を含める。")
  frontchannel_logout_uri?: url;
  @doc("iframe URI に iss / sid クエリパラメータを付与するか。")
  frontchannel_logout_session_required: boolean = false;
  @doc("この client 向けの claim 発行 policy (アプリ単位の release 上書き)。\nNameID 相当の source 属性を sub の source として使う。scope/consent による ClaimExposed 属性の\n開示可否判定 (openid/profile/email 等の標準 scope) はこの policy とは独立に先に適用され、\n本 policy はその範囲に対する追加の絞り込み・属性写像として ID Token / UserInfo にマージされる。")
  claim_policy: ClaimMappingPolicy;
  created_at: utcDateTime;
  @doc("作成時は created_at と同じ値を持ち、以後 client metadata の管理操作ごとに更新する。")
  updated_at: utcDateTime;
}


@doc("secret-based OAuth2 client の認証 credential。平文は発行応答だけで返し、保存・イベント・一覧には含めない。新規発行は既存 credential を変更せず、個別 revoke または expiry まで照合対象となる。既存の期限なし credential は無停止移行のため許容する。")
model ClientSecretCredential {
  id: unknown;
  client_id: string;
  secret_hash: string;
  created_at: utcDateTime;
  @doc("credential の有効期限。新規追加発行では必須、null は移行前から存在する期限なし legacy credential または互換 rotation の current credential を表す。")
  expires_at?: utcDateTime;
  @doc("即時 revoke 時刻。revoked_at がある credential は expires_at に関わらず認証に使わない。")
  revoked_at?: utcDateTime;
}


@doc("GDPR 同意取得記録。状態は ConsentLifecycle 状態機械に従う。user_id と client_id は既存 User と OAuth2Client を指す。")
model Consent {
  user_id: string;
  client_id: string;
  scopes: string[];
  state: ConsentState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  granted_at: utcDateTime;
  @doc("通常 granted_at + 365 日")
  expires_at: utcDateTime;
  @doc("state == Revoked になった時刻（監査）。")
  revoked_at?: utcDateTime;
  @doc("RFC 9396 — ユーザーが同意した構造化詳細。発行・交換時の部分集合判定の基準。")
  authorization_details?: AuthorizationDetail[];
}


@doc("RFC 9396 authorization_details の 1 要素。type で識別される構造化された細粒度権限を表す。 要求・同意・トークン反映を通じて一貫して扱い、登録済み AuthorizationDetailType に対し fail-closed に検証する。")
model AuthorizationDetail {
  @doc("登録済み AuthorizationDetailType の識別子 (例: payment_initiation)。")
  @minLength(1)
  type: string;
  @doc("RFC 9396 — 詳細が向く resource server の URI 群。")
  locations?: string[];
  @doc("RFC 9396 — 対象に対して許可する操作群。")
  actions?: string[];
  @doc("RFC 9396 — アクセスするデータ種別群。")
  datatypes?: string[];
  @doc("RFC 9396 — 対象リソースの識別子。")
  identifier?: string;
  @doc("RFC 9396 — 対象に対する権限レベル群。")
  privileges?: string[];
  @doc("type 固有の構造化フィールド (例: instructedAmount, creditorAccount)。登録スキーマで意味論を束縛する。")
  fields?: unknown;
}


@doc("AuthorizationDetailsSchema の 1 フィールド規則。ダウンスコープ半順序と許可値を定義する。")
model AuthorizationDetailFieldRule {
  @doc("対象フィールド名。標準フィールド (actions 等) または fields マップ内のキー。")
  @minLength(1)
  name: string;
  @doc("部分集合判定に使う半順序。")
  semantics: AuthorizationDetailFieldSemantics;
  @doc("true のとき詳細にこのフィールドが必須。")
  required: boolean = false;
  @doc("semantics が set / enum のとき許可される値の閉集合 (空なら無制限)。")
  allowed?: string[];
}


@doc("ある type の構造的スキーマ。受理するフィールドと各フィールドのダウンスコープ半順序を列挙する。 未登録フィールド・スキーマ不適合・許可外値はすべて拒否する (fail-closed)。")
model AuthorizationDetailsSchema {
  @doc("フィールド規則の集合。")
  rules: AuthorizationDetailFieldRule[];
}


@doc("テナントが登録する authorization_details の type 定義。type 識別子・検証スキーマ・同意 UI の 表示テンプレートを束ねる。Enabled な type のみ受理する。管理は AdminAuthorizationDetailTypesManage で保護する。")
model AuthorizationDetailType {
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("authorization_details の type 識別子。")
  @minLength(1)
  type: string;
  description?: string;
  schema: AuthorizationDetailsSchema;
  @doc("同意 UI で詳細を人間可読に提示するテンプレート (例: '{identifier} へ最大 {fields.instructedAmount} まで')。")
  display_template: string;
  state: AuthorizationDetailTypeState = AuthorizationDetailTypeState.Enabled;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("McpResourceServer の運用状態列挙。Active のみ resource indicator 検証を通過する。Disabled は新規発行を拒否する (既存トークンは失効まで有効)。")
enum McpResourceServerState {
  Active: "Active",
  Disabled: "Disabled",
}


@doc("MCP resource server (ツール/データソース) の tenant-scoped 登録。canonical resource URI と許可 scope を所有し、Protected Resource Metadata (RFC 9728) と resource indicator (RFC 8707) 検証の基準になる。管理は AdminMcpResourcesManage で保護する。")
model McpResourceServer {
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("登録の一意識別子 (UUID)。tenant を越えて globally unique。")
  @minLength(1)
  id: string;
  @doc("発行トークンの aud に厳格束縛される canonical resource URI。tenant 内で一意。")
  resource: ResourceIndicator;
  @doc("管理 UI 表示名。")
  @minLength(1)
  @maxLength(200)
  name: string;
  @doc("この resource へのアクセスで許可する scope の allowlist。resource 指定時の要求 scope はこの部分集合でなければならない。")
  scopes: string[];
  state: McpResourceServerState = McpResourceServerState.Active;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("RFC 8707 の resource パラメータ。要求 audience を表す絶対 URI で fragment を含まない。1 トークンにつき 1 個のみを許可し、複数指定は invalid_target で拒否する。")
model ResourceIndicator {
  @minLength(1)
  value: url;
}


@doc("RFC 9728 Protected Resource Metadata。realm の IdMagic API は ApiTokenScope、外部 resource は McpResourceServer 登録から導出する。")
model ProtectedResourceMetadata {
  resource: url;
  @doc("この resource を保護する Authorization Server (= idmagic issuer) の一覧。")
  authorization_servers: url[];
  scopes_supported: string[];
  bearer_methods_supported: string[] = #["header"];
  dpop_signing_alg_values_supported?: string[];
}


@doc("管理者が McpResourceServer を登録・更新する JSON リクエスト。resource は作成時のみ。")
model McpResourceServerRequest {
  @doc("作成時必須。更新では変更不可 (発行済みトークンの aud 意味が変わるため)。")
  resource?: string;
  @minLength(1)
  @maxLength(200)
  name: string;
  scopes: string[];
  state?: McpResourceServerState;
}


@doc("McpResourceServer 表現。管理 UI の一覧と詳細で同じ表現を返す。")
model McpResourceServerResponse {
  tenant_id: string;
  id: string;
  resource: url;
  name: string;
  scopes: string[];
  state: McpResourceServerState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("管理 API の McpResourceServer 一覧。")
model McpResourceServerListResponse {
  resource_servers: McpResourceServerResponse[];
}


@doc("リフレッシュトークンのストアレコード。プレーン値は保存しない。状態は RefreshTokenLifecycle 状態機械に従う。tenant は global ID の User / OAuth2Client から導出し、token 行に冗長保持しない。use case は両者の tenant-local invariant を検証する。")
model RefreshTokenRecord {
  id: unknown;
  @doc("SHA-256 ハッシュ。")
  hash: string;
  @doc("認可コード交換から派生したすべてのトークンが共有する識別子。")
  family_id: unknown;
  parent_id?: unknown;
  client_id: string;
  user_id: string;
  scopes: string[];
  state: RefreshTokenState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  @doc("ローテーションされても延長されない絶対期限。")
  absolute_expires_at: utcDateTime;
  sender_constraint?: RefreshTokenSenderConstraint;
  @doc("発行元 AuthorizationCodeRecord.sid (OIDC session id)。Rotate では親の値を\nそのまま引き継ぐ。null は client_credentials 等 browser session を持たない発行。\nAuthentication の SessionEnded.sessionId と一致する全 family (複数 client を含みうる)\nを本フィールドで解決し、session revoke から一括で RevokeToken へ遷移させる。")
  sid?: string;
  @doc("発行元 (AuthorizationCodeRecord.resource) の resource indicator。\nRotate では親の値をそのまま引き継ぎ、ローテーション後も同じ McpResourceServer へ\naudience を厳格限定する。null は resource 未指定の発行。")
  resource?: ResourceIndicator;
}


@doc("ある browser session (sid) が特定の OAuth2Client (RP) に対して認証結果を\n発行した記録 (RP participation)。sid は Authentication の LoginSession.id と同値で\nRP 間で共通。session 状態そのものはここに複製せず、session revoke 時にどの RP へ\nfront/back-channel logout を通知するかを解決する索引としてのみ使う。")
model ClientSession {
  @doc("LoginSession.id と同値の OIDC session id。")
  sid: string;
  client_id: string;
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  first_issued_at: utcDateTime;
  @doc("同一 RP への再発行 (再認可) ごとに更新する。通知先解決には影響しない。")
  last_issued_at: utcDateTime;
}


@doc("back-channel logout の1配送試行を表すレコード。ローカル session/refresh token\nの失効は本レコード作成前に確定済みであり、配送の成否で失効を取り消さない。\n配送実行は Jobs bounded context の durable job\n(kind=backchannel_logout_delivery) に委譲する。")
model LogoutNotification {
  id: unknown;
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("通知対象の OIDC session id。")
  sid: string;
  @doc("通知先 RP。")
  client_id: string;
  @doc("発行した logout token の jti。RP 側の replay 検出と idmagic 側の監査相関に使う。")
  logout_token_jti: string;
  @doc("配送先 (client の backchannel_logout_uri を配送開始時点でスナップショットした値)。")
  target_uri: url;
  state: LogoutNotificationState;
  attempts: int64 = 0;
  last_error?: string;
  @doc("Jobs context の JobRef.id (配送実行の相関)。")
  job_id?: string;
  created_at: utcDateTime;
  delivered_at?: utcDateTime;
}


@doc("LogoutNotification の配送状態。Jobs の JobStatus と1:1ではなく、配送固有の成否のみを表す (job 自体の retry/lease は Jobs 側が保持する)。")
enum LogoutNotificationState {
  Pending: "Pending",
  Delivered: "Delivered",
  Failed: "Failed",
}


@doc("フロー追跡用のサーバー内部中間ステート。")
model AuthorizationRequest {
  id: unknown;
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  state: AuthorizationCodeState;
  client_id: string;
  redirect_uri: url;
  response_type: ResponseType;
  scope: string;
  state_param?: string;
  nonce?: string;
  code_challenge: string;
  code_challenge_method: CodeChallengeMethod;
  prompt?: string;
  max_age?: int64;
  par_request_uri?: string;
  user_id?: string;
  auth_time?: int64;
  @doc("認証成功時のメソッド列。advance(authenticate_user) で AuthenticationContext から伝搬。")
  amr?: string[];
  @doc("amr から AuthenticationContextPolicy に従って導出した URN。")
  acr?: string;
  @doc("クライアントが /authorize に与えた acr_values パラメータ (RFC 8176)。step-up 判定に使う。")
  acr_values?: string;
  @doc("advance(authenticate_user) で AuthenticationContext.session_id から伝搬する\nOIDC session id (Authentication context の LoginSession.id と同値)。AuthorizationCodeRecord /\nRefreshTokenRecord / IdTokenClaims へ引き継ぎ、session revoke 起点の family 失効と\nfront/back-channel logout の相関に使う。")
  sid?: string;
  @doc("RFC 9396 — クライアントが要求した構造化詳細。検証済み・登録 type 適合済みのものを保持する。")
  authorization_details?: AuthorizationDetail[];
  created_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("発行済み認可コードの永続レコード。code 本体は短命。状態は AuthorizationCodeRecordLifecycle に従う（AuthorizationCodeFlow とは別軸でコード自体のライフサイクルを表す）。")
model AuthorizationCodeRecord {
  code: string;
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  authorization_request_id: unknown;
  client_id: string;
  user_id: string;
  scopes: string[];
  redirect_uri: url;
  code_challenge: string;
  code_challenge_method: CodeChallengeMethod;
  nonce?: string;
  auth_time: int64;
  @doc("認可リクエストから引き継いだ認証メソッド列。id_token / access_token に emit する。")
  amr?: string[];
  @doc("認可リクエストから引き継いだ ACR URN。id_token / access_token に emit する。")
  acr?: string;
  @doc("認可リクエストから引き継いだ OIDC session id。id_token の sid claim と発行 RefreshTokenRecord.sid に伝播する。")
  sid?: string;
  @doc("RFC 9396 — 同意済み構造化詳細。交換で発行トークンへ反映する。")
  authorization_details?: AuthorizationDetail[];
  state: AuthorizationCodeRecordState;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  @doc("state == Redeemed になった時刻（監査）。")
  redeemed_at?: utcDateTime;
  @doc("成功交換で発行された refresh family（不正リプレイ検知時のファミリー失効に使う）")
  issued_family_id?: unknown;
}


@doc("Pushed Authorization Request の格納レコード。状態は PARRecordLifecycle に従う。")
model PARRecord {
  request_uri: string;
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  client_id: string;
  parameters: unknown;
  state: PARRecordState;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("RFC 8628 デバイス認可レコード。device_code はハッシュのみで保存する。")
model DeviceAuthorization {
  device_code_hash: string;
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("紛らわしい文字を避けた短いコード。")
  user_code: string;
  client_id: string;
  scopes: string[];
  state: DeviceCodeState;
  user_id?: string;
  auth_time?: int64;
  interval_seconds: int64;
  last_polled_at?: utcDateTime;
  issued_family_id?: unknown;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("人間の承認が成立するまで client の要求を保留する承認要求。判断の状態と内容は CIBA の\n輸送 bookkeeping から分離するが、競合する判断と poll を一つの store 境界で直列化するため同じ\nrecord に併置する。id は UUID で、auth_req_id は SHA-256 ハッシュだけを索引として保存する。")
model ApprovalRequest {
  id: string;
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("承認要求を起票した client。")
  client_id: string;
  @doc("client に束縛された Agent (存在する場合)。承認画面と発行トークンの agent_id claim に使う。")
  agent_id?: string;
  @doc("承認判断を行う User の sub。login_hint / id_token_hint から解決し、以後この User だけが判断できる。")
  user_id: string;
  @doc("承認されたときに発行するトークンの scope。")
  scopes: string[];
  @doc("RFC 9396 — 承認対象の構造化された権限要求。承認画面はこれを人間可読に描画する。")
  authorization_details?: AuthorizationDetail[];
  @doc("要求の取り違え承認を防ぐため承認画面に表示する短い識別文 (CIBA Core §7.1 binding_message)。")
  binding_message?: string;
  state: ApprovalRequestState;
  @doc("CIBA auth_req_id の SHA-256 ハッシュ。輸送層の索引であり、決定の意味論には関与しない。")
  auth_req_id_hash: string;
  @doc("輸送層の bookkeeping — client に指示する最小ポーリング間隔 (秒)。slow_down のたびに増える。")
  interval_seconds: int64;
  @doc("輸送層の bookkeeping — 直近のポーリング時刻。slow_down の判定に使う。")
  last_polled_at?: utcDateTime;
  requested_at: utcDateTime;
  expires_at: utcDateTime;
  decided_at?: utcDateTime;
  consumed_at?: utcDateTime;
}


@doc("Sender-Constrained 証明 (RFC 7800)。")
model SenderConstraintCnf {
  @doc("DPoP 鍵の SHA-256 サムプリント (RFC 9449 §6.1)。")
  jkt?: string;
  @doc("mTLS クライアント証明書のサムプリント (x5t#S256, RFC 8705)。")
  x5t_s256?: string;
}


@doc("DPoP Proof JWT のクレーム (RFC 9449 §4.2)。typ は dpop+jwt、公開鍵は jwk ヘッダで提示する。")
model DpopProofClaims {
  @doc("proof ごとに一意な識別子。リプレイ窓の間は再使用できない。")
  jti: string;
  @doc("対象リクエストの HTTP メソッド。")
  htm: string;
  @doc("query と fragment を除いた対象リクエストの HTTP URI。")
  htu: url;
  @doc("proof の発行時刻。許容 clock skew の外は拒否する。")
  iat: utcDateTime;
  @doc("提示する access token の base64url(SHA-256) ハッシュ (RFC 9449 §4.3)。保護リソースへのアクセスでは必須、トークンエンドポイントでは対象 token が存在しないため付与しない。")
  ath?: string;
}


@doc("RefreshToken に紐づくセンダー制約。発行時のバインド先（DPoP 鍵サムプリントまたは mTLS 証明書サムプリント）を保持する。")
model RefreshTokenSenderConstraint {
  type: SenderConstraintKind;
  jkt?: string;
  x5t_s256?: string;
}


@doc("JWT Access Token クレーム (RFC 9068)。TTL 600 秒。")
model AccessTokenClaims {
  iss: url;
  @doc("client_credentials 時は client_id と一致。")
  sub: string;
  aud: Audience;
  client_id: string;
  @doc("スペース区切りスコープ列。")
  scope: string;
  @doc("Unix epoch seconds")
  exp: int64;
  iat: int64;
  nbf?: int64;
  jti: string;
  auth_time?: int64;
  acr?: string;
  amr?: string[];
  cnf?: SenderConstraintCnf;
  @doc("RFC 8693 §4.1 actor claim。委任の現在アクターを表す。subject_token に act があれば入れ子で連結される。")
  act?: unknown;
  @doc("RFC 8693 §4.4 may_act claim。許可された委任先アクターを表す。")
  may_act?: unknown;
  @doc("RFC 9396 — トークンに束縛された構造化詳細。RS の検証点。optional で後方互換。")
  authorization_details?: AuthorizationDetail[];
}


@doc("OIDC ID Token クレーム。TTL 3600 秒。")
model IdTokenClaims {
  iss: url;
  sub: string;
  aud: Audience;
  exp: int64;
  iat: int64;
  auth_time: int64;
  @doc("OIDC session id (OpenID Connect Front/Back-Channel Logout 1.0)。browser session に\n束縛された発行のときだけ含める。RP は本 claim を id_token_hint として /end_session に\n送り返し、front/back-channel logout の相関に使う。")
  sid?: string;
  nonce?: string;
  acr?: string;
  amr?: string[];
  azp?: string;
  at_hash?: string;
  c_hash?: string;
  name?: string;
  preferred_username?: string;
  email?: string;
  email_verified?: boolean;
  @doc("scope 主導の属性 claim (ClaimsForScopes) と client 単位の ClaimMapping claim_policy\nをマージした追加 claim。標準 claim とキーが衝突した場合は標準 claim を\n優先する。")
  additional_claims?: unknown;
}


@doc("文字列または1つ以上の文字列の配列（JWT RFC 7519 §4.1.3）。")
model Audience {
  @doc("JSON string または1つ以上の JSON string の配列。")
  one_or_many: unknown;
}


@doc("RFC 8414 / OIDC Discovery 1.0 メタデータ。")
model DiscoveryDocument {
  issuer: url;
  authorization_endpoint: url;
  token_endpoint: url;
  userinfo_endpoint: url;
  jwks_uri: url;
  introspection_endpoint: url;
  revocation_endpoint: url;
  pushed_authorization_request_endpoint: url;
  device_authorization_endpoint: url;
  @doc("CIBA Core §4 — backchannel authentication endpoint。CIBA を提供する場合にだけ広告する。")
  backchannel_authentication_endpoint?: url;
  @doc("CIBA Core §4 — 実装している token delivery mode。本実装は poll のみで、ping / push は広告しない。")
  backchannel_token_delivery_modes_supported?: string[] = #["poll"];
  @doc("CIBA Core §4 — user_code パラメータは非対応。承認画面は本人の認証済み session と step-up で保護されるため、二つ目の共有秘密を持たない。")
  backchannel_user_code_parameter_supported?: boolean = false;
  registration_endpoint: url;
  end_session_endpoint?: url;
  frontchannel_logout_supported: boolean = true;
  frontchannel_logout_session_supported: boolean = true;
  backchannel_logout_supported: boolean = true;
  backchannel_logout_session_supported: boolean = true;
  check_session_iframe?: url;
  scopes_supported: string[] = #["openid", "profile", "email", "phone", "address", "custom_attribute", "offline_access", "idmagic.admin", "idmagic.account", "account:read", "account:write", "account:mfa:write", "account:sessions:write", "account:consents:write", "account:password:write"];
  response_types_supported: ResponseType[];
  response_modes_supported?: ResponseMode[];
  grant_types_supported: GrantType[];
  token_endpoint_auth_methods_supported: TokenEndpointAuthMethod[];
  introspection_endpoint_auth_methods_supported: TokenEndpointAuthMethod[] = #[TokenEndpointAuthMethod.ClientSecretBasic, TokenEndpointAuthMethod.PrivateKeyJwt, TokenEndpointAuthMethod.TlsClientAuth];
  revocation_endpoint_auth_methods_supported: TokenEndpointAuthMethod[] = #[TokenEndpointAuthMethod.ClientSecretBasic, TokenEndpointAuthMethod.PrivateKeyJwt, TokenEndpointAuthMethod.TlsClientAuth, TokenEndpointAuthMethod.None];
  id_token_signing_alg_values_supported: SignatureAlgorithm[];
  code_challenge_methods_supported: CodeChallengeMethod[];
  require_pushed_authorization_requests: boolean;
  require_pkce: boolean;
  authorization_response_iss_parameter_supported?: boolean;
  dpop_signing_alg_values_supported?: SignatureAlgorithm[];
  tls_client_certificate_bound_access_tokens?: boolean;
  @doc("client_idをHTTPS URL (Client ID Metadata Document) として都度解決する経路をサポートすることの広告。")
  client_id_metadata_document_supported: boolean = true;
  subject_types_supported: string[] = #["public"];
  claims_supported?: string[] = #["sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "acr", "amr", "azp", "name", "given_name", "family_name", "preferred_username", "email", "email_verified", "updated_at"];
  acr_values_supported?: string[] = #["urn:idmagic:acr:pwd", "urn:idmagic:acr:mfa"];
  ui_locales_supported?: string[] = #["en", "ja"];
  @doc("RFC 9396 — 当テナントで受理する authorization_details の type 一覧 (Enabled な AuthorizationDetailType から導出)。")
  authorization_details_types_supported?: string[];
}


@doc("/register への動的クライアント登録リクエスト本体 (RFC 7591)。登録後の OAuth2Client は本リクエストの内容から派生する。")
model ClientRegistrationRequest {
  redirect_uris: url[];
  client_name?: string;
  client_type?: ClientType;
  grant_types: GrantType[];
  response_types?: ResponseType[];
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  scope?: string;
  jwks_uri?: url;
  jwks?: unknown;
  tls_client_auth_subject_dn?: string;
  id_token_signed_response_alg?: SignatureAlgorithm;
  fapi_profile?: FapiProfile;
  require_pkce?: boolean;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
}


@doc("client が自己ホストする HTTPS URL から fetch する Client ID Metadata Document。client_id 自体が fetch 元 URL であり document 内 client_id と厳密一致する必要がある。token_endpoint_auth_method が未指定または none 以外の document は拒否する。")
model ClientIDMetadataDocument {
  @doc("fetch 元 URL と厳密一致する必要がある。")
  client_id: url;
  client_name: string;
  client_uri?: url;
  logo_uri?: url;
  redirect_uris: url[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  @doc("未指定は none として扱う。none 以外の値を宣言する document は拒否する。")
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  @doc("未指定時は openid のみを許可する。")
  scope?: string;
}


@doc("管理者が authorization_details type を登録・更新する JSON リクエスト (RFC 9396)。type は作成時のみ。")
model AuthorizationDetailTypeRequest {
  @doc("作成時必須。更新ではパスの type を使う。")
  @maxLength(100)
  type?: string;
  @maxLength(500)
  description?: string;
  schema: AuthorizationDetailsSchema;
  @minLength(1)
  @maxLength(500)
  display_template: string;
  state?: AuthorizationDetailTypeState;
}


@doc("authorization_details type 表現。管理 UI の一覧と詳細で同じ表現を返す。")
model AuthorizationDetailTypeResponse {
  tenant_id: string;
  type: string;
  description?: string;
  schema: AuthorizationDetailsSchema;
  display_template: string;
  state: AuthorizationDetailTypeState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("管理 API の authorization_details type 一覧。")
model AuthorizationDetailTypeListResponse {
  types: AuthorizationDetailTypeResponse[];
}


@doc("管理者が所属テナントに OAuth client を作成する。生成された client_secret は作成応答で一度だけ返す。")
model AdminOAuth2ClientCreateRequest {
  redirect_uris: url[];
  @maxLength(200)
  client_name?: string;
  client_type?: ClientType;
  grant_types: GrantType[];
  response_types?: ResponseType[];
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  scope?: string;
  jwks_uri?: url;
  jwks?: unknown;
  tls_client_auth_subject_dn?: string;
  fapi_profile?: FapiProfile;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
}


@doc("管理者が client metadata を更新する。client_id、認証方式、秘密情報は変更せず、secret rotation は独立 endpoint で扱う。")
model AdminOAuth2ClientUpdateRequest {
  @maxLength(200)
  client_name?: string;
  redirect_uris?: url[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  scope?: string;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
}


@doc("管理 API の client 表現。client_secret と client_secret_hash は契約上含めない。")
model AdminOAuth2ClientResponse {
  tenant_id: string;
  client_id: string;
  client_name?: string;
  client_type: ClientType;
  redirect_uris: url[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  jwks_uri?: url;
  jwks?: unknown;
  tls_client_auth_subject_dn?: string;
  id_token_signed_response_alg: SignatureAlgorithm;
  require_pushed_authorization_requests: boolean;
  dpop_bound_access_tokens: boolean;
  fapi_profile: FapiProfile;
  created_at: utcDateTime;
}


@doc("client 作成応答。client_secret は生成された場合に限り一度だけ返す。")
model AdminOAuth2ClientCreateResponse {
  client: AdminOAuth2ClientResponse;
  client_secret?: string;
}


@doc("所属テナント内の client 一覧。")
model AdminOAuth2ClientListResponse {
  clients: AdminOAuth2ClientResponse[];
}


@doc("Application editor が表示する secret credential の非機密メタデータ。状態は revoked_at、expires_at、評価時刻の順で導出する。")
model ClientSecretCredentialMetadata {
  id: unknown;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
  status: ClientSecretCredentialStatus;
}


@doc("既存 credential を変更せず、有効期限付き client secret を追加発行する要求。expires_in_days は未指定時90、1..730日だけ受理する。")
model IssueClientSecretRequest {
  expires_in_days?: int64;
}


@doc("追加発行した secret を一度だけ返す応答。既存 secret、hash、平文の再取得経路は含めない。")
model IssueClientSecretResponse {
  client_secret: string;
  credential: ClientSecretCredentialMetadata;
  credentials: ClientSecretCredentialMetadata[];
}


@doc("個別失効後の非機密 credential metadata 一覧。secret と hash は含めない。")
model RevokeClientSecretResponse {
  credentials: ClientSecretCredentialMetadata[];
}


@doc("既存 secret の overlap 日数を指定する rotation 要求。0 は即時 revoke rotation、1..30 は猶予付き rotation、未指定は7日。")
model RotateClientSecretRequest {
  grace_days?: int64;
}


@doc("新 secret を一度だけ返す rotation 応答。以後の取得 API は存在しない。")
model RotateClientSecretResponse {
  client_secret: string;
  grace_until?: utcDateTime;
  credentials: ClientSecretCredentialMetadata[];
}


@doc("管理 API が返す所属テナント内の Consent 表現。client_id / user_id は UUID のため、表示用に解決済みの可読名 (client_name / preferred_username) を併記する。")
model AdminConsentResponse {
  user_id: string;
  @doc("user_id を解決した表示名。該当 User が見つからない場合は省略する。")
  preferred_username?: string;
  client_id: string;
  @doc("client_id を解決した表示名。OAuth2Client.client_name → Application\nカタログ名の順で解決し、いずれも無ければ client_id をそのまま返す。")
  client_name: string;
  scopes: string[];
  state: ConsentState;
  granted_at: utcDateTime;
  expires_at: utcDateTime;
  revoked_at?: utcDateTime;
}


@doc("所属テナント内の Consent 一覧。")
model AdminConsentListResponse {
  consents: AdminConsentResponse[];
}


@doc("end-user 自身の接続済みアプリ (active な Consent) 一覧 (self-service)。")
model AccountConsentListResponse {
  consents: AccountConsentResponse[];
}


@doc("application/x-www-form-urlencoded で送信される token endpoint リクエスト。")
model TokenRequest {
  grant_type: GrantType;
  code?: string;
  redirect_uri?: string;
  code_verifier?: string;
  refresh_token?: string;
  device_code?: string;
  @doc("CIBA grant (urn:openid:params:grant-type:ciba) の auth_req_id。承認が成立するまで authorization_pending / slow_down を返し、成立後は一度だけトークン化する。")
  auth_req_id?: string;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
  scope?: string;
  @doc("RFC 8693 token-exchange grant の subject_token。自己発行 access_token (urn:ietf:params:oauth:token-type:access_token)、または登録済み WorkloadTrustBundle の issuer から発行された外部 attestation JWT (urn:ietf:params:oauth:token-type:jwt) のいずれか。")
  subject_token?: string;
  @doc("subject_token の種別を示す URN。省略時は self-issued access_token として扱う。")
  subject_token_type?: string;
  @doc("RFC 8693 token-exchange grant の actor_token (任意)。自己発行 access_token に限る。")
  actor_token?: string;
  actor_token_type?: string;
  requested_token_type?: string;
  @doc("RFC 9396 — token-exchange でダウンスコープを要求する構造化詳細の JSON 文字列 (任意)。")
  authorization_details?: string;
  @doc("RFC 8707 — authorization_code redemption・client_credentials・device_code・token-exchange の resource indicator。同名パラメータの複数指定は invalid_target。refresh_token grant では明示指定せず、初回発行時に束縛された resource を rotation を跨いで自動的に保持する。")
  resource?: ResourceIndicator;
}


@doc("/token 成功時のレスポンスボディ (RFC 6749 §5.1)。発行されたトークン一式と種別を返す。")
model TokenResponse {
  access_token: string;
  @doc("\"Bearer\" or \"DPoP\"")
  token_type: string;
  expires_in?: int64;
  refresh_token?: string;
  id_token?: string;
  scope?: string;
}


@doc("/introspect リクエスト (RFC 7662 §2.1)。ResourceServer がトークンの有効性を AuthorizationServer に問い合わせる。")
model IntrospectionRequest {
  token: string;
  token_type_hint?: TokenType;
}


@doc("/introspect レスポンス (RFC 7662 §2.2)。トークンの active 状態と有効時のメタデータを返す。")
model IntrospectionResponse {
  active: boolean;
  scope?: string;
  client_id?: string;
  sub?: string;
  aud?: Audience;
  exp?: int64;
  iat?: int64;
  token_type?: string;
  jti?: string;
  cnf?: SenderConstraintCnf;
  @doc("RFC 8693 §4.1 actor claim。委任トークンの場合に現在アクターとアクター連鎖を返す。")
  act?: unknown;
}


@doc("/revoke リクエスト (RFC 7009 §2.1)。指定したトークンを失効させる。")
model RevokeRequest {
  token: string;
  token_type_hint?: TokenType;
}


@doc("任意の authorize パラメータを form で受け取る。")
model ParRequest {
  parameters: unknown;
}


@doc("/par レスポンス (RFC 9126 §2.2)。短命の request_uri を発行し、後続の /authorize から参照される。")
model ParResponse {
  request_uri: string;
  expires_in: int64;
}


@doc("/device_authorization リクエスト (RFC 8628 §3.1)。入力制約のあるデバイスからのフロー開始要求。")
model DeviceAuthorizationRequest {
  scope?: string;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
}


@doc("/device_authorization レスポンス (RFC 8628 §3.2)。device_code・user_code・verification_uri を返し、デバイスはポーリングへ移行する。")
model DeviceAuthorizationResponse {
  device_code: string;
  user_code: string;
  verification_uri: url;
  verification_uri_complete?: url;
  expires_in: int64;
  interval?: int64;
}


@doc("/bc-authorize リクエスト (CIBA Core §7.1)。client が人間の承認を求めて起票する。\nlogin_hint と id_token_hint はいずれか一方だけを指定し、どちらも解決できなければ\nunknown_user_id で fail-closed に拒否する。user_code は非対応 (discovery で広告しない)。")
model BackchannelAuthenticationRequest {
  @doc("承認対象 User を示すヒント。username または email として解決する。")
  login_hint?: string;
  @doc("本 OP が発行した有効な ID Token。署名・iss・aud・sub・exp を検証して承認対象 User を解決する。")
  id_token_hint?: string;
  @doc("承認されたときに発行するトークンの scope。必須かつ openid を含み、client の宣言 scope の部分集合でなければならない。")
  scope: string;
  @doc("RFC 9396 — 承認対象の構造化された権限要求 (JSON 文字列)。登録済み type だけを受理する。")
  authorization_details?: string;
  @doc("承認画面に表示する短い識別文 (CIBA Core §7.1)。")
  binding_message?: string;
  @doc("client が要求する有効期限 (秒)。正の整数かつ 600 秒以下。未指定時は 300 秒。")
  requested_expiry?: int64;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
}


@doc("/bc-authorize レスポンス (CIBA Core §7.3)。auth_req_id は bearer secret であり、\nclient はこれを token endpoint の ciba grant で提示してポーリングする。")
model BackchannelAuthenticationResponse {
  auth_req_id: string;
  expires_in: int64;
  @doc("client が守るべき最小ポーリング間隔 (秒)。")
  interval?: int64;
}


@doc("end-user の account portal が返す保留中の承認要求 1 件 (self-service)。自分宛に固定される\nため user_id は出さない。binding_message だけで内容を代替せず、要求元と要求内容を並べて返す。")
model AccountApprovalRequest {
  id: string;
  client_id: string;
  @doc("client_id を解決した表示名。OAuth2Client.client_name → Application カタログ名の順で\n解決し、いずれも無ければ client_id をそのまま返す。")
  client_name: string;
  @doc("client に束縛された Agent の表示名 (存在する場合)。")
  agent_name?: string;
  scopes: string[];
  authorization_details?: AuthorizationDetail[];
  binding_message?: string;
  requested_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("end-user 自身宛の保留中承認要求の一覧 (self-service)。")
model AccountApprovalRequestListResponse {
  approval_requests: AccountApprovalRequest[];
}


@doc("end-user が承認要求 1 件に下す判断。approve または deny のいずれか。")
model AccountApprovalDecisionRequest {
  @doc("approve または deny。")
  decision: string;
}


@doc("GET /authorize または PAR の認可パラメータ。全 field は単一値であり、同名parameterの複数指定はinvalid_requestとして拒否する。promptは空白区切りの一意なlogin、consent、noneのtoken集合で、noneは他tokenと併用しない。")
model AuthorizeParameters {
  response_type: ResponseType;
  client_id: string;
  redirect_uri: string;
  scope?: string;
  state?: string;
  nonce?: string;
  code_challenge: string;
  code_challenge_method: CodeChallengeMethod;
  prompt?: string;
  max_age?: int64;
  id_token_hint?: string;
  acr_values?: string;
  request_uri?: string;
  authorization_details?: string;
  @doc("RFC 8707 resource indicator。登録済み Active な McpResourceServer と一致しない場合は invalid_target で拒否する。同名parameterの複数指定も invalid_target。")
  resource?: ResourceIndicator;
}


@doc("SPA が POST /api/auth/consent に送る同意判断。")
model BrowserConsentRequest {
  @doc("allow または deny。")
  action: string;
}


@doc("SPA が POST /api/auth/device に送る device user_code の承認判断。")
model BrowserDeviceRequest {
  user_code: string;
}


@doc("SPA が現在の browser authorization transaction を描画するための no-store JSON。")
model BrowserTransactionResponse {
  @doc("描画すべき画面の識別子。login / totp / consent / device 等、factor 追加に応じて拡張する。")
  kind: string;
  csrf_token: string;
  @doc("同意画面に表示する client の可読名。OAuth2Client.client_name →\nApplication カタログ名 → client_id の順で解決する (client_id は UUID のため生表示を避ける)。")
  client_name?: string;
  scopes?: string[];
}


@doc("SPA が device user_code 画面を描画するための no-store JSON。")
model BrowserDeviceContextResponse {
  kind: string;
  user_code?: string;
  csrf_token: string;
}


@doc("RP-Initiated Logout v1 のクエリ / form パラメータ。client_id で\npost_logout_redirect_uri を検証する。id_token_hint は本 OP が発行した ID Token の\nみ受理し、署名・iss・aud (client_id と矛盾する場合は拒否)・sub・sid を検証する。\nexp は検証しない (ログアウト時点で期限切れになっている ID Token を\nhint として送るのが通常の RP 実装であるため)。")
model EndSessionParameters {
  client_id?: string;
  id_token_hint?: string;
  post_logout_redirect_uri?: url;
  state?: string;
}


@doc("FrontChannelLogout の返り値1件。EndSession 応答が埋め込む iframe 1つ分に対応する。")
model FrontChannelLogoutTarget {
  client_id: string;
  @doc("登録済み frontchannel_logout_uri に (frontchannel_logout_session_required なら) iss / sid を付与した URI。")
  iframe_uri: url;
}


@doc("OIDC Core §5.3。可変な OIDC クレームを含む。")
model UserInfoResponse {
  sub: string;
  @doc("scope 主導の属性 claim (ClaimsForScopes) と client 単位の ClaimMapping claim_policy\nをマージした追加 claim。標準 claim とキーが衝突した場合は標準 claim を優先する。")
  claims?: unknown;
}


@doc("OAuth 2.0 / OIDC のエラー応答の共通形 (RFC 6749 §5.2)。具体的なエラー種別は error フィールドの OAuthErrorCode で識別される。")
model OAuthError {

}


@doc("クライアント認証が失敗した、または未登録のクライアント (RFC 6749 §5.2 invalid_client)。client_id の存在有無を開示しないため詳細は返さない。")
model InvalidClientError {

}


@doc("認可コード・リフレッシュトークン・device_code 等の付与が無効、期限切れ、再利用、あるいは redirect_uri/PKCE 不一致 (RFC 6749 §5.2 invalid_grant)。")
model InvalidGrantError {

}


@doc("クライアントがこの grant_type を使用する権限を持たない (RFC 6749 §5.2 unauthorized_client)。")
model UnauthorizedClientError {

}


@doc("AuthorizationServer がこの grant_type をサポートしていない (RFC 6749 §5.2 unsupported_grant_type)。")
model UnsupportedGrantTypeError {

}


@doc("要求された scope が不正、未知、またはクライアントに許可されていない (RFC 6749 §5.2 invalid_scope)。")
model InvalidScopeError {

}


@doc("resource パラメータが未登録・Disabled・複数指定 (RFC 8707 §2 invalid_target)。fail-closed で拒否する。")
model InvalidTargetError {

}


@doc("アクセストークンが期限切れ、改ざん、不正、または失効済み (RFC 6750 §3.1 invalid_token)。")
model InvalidTokenError {

}


@doc("DPoP Proof JWT の検証に失敗した（署名・jti・htu/htm・nonce・clock skew 等、RFC 9449 §7.1 invalid_dpop_proof）。")
model InvalidDpopProofError {

}


@doc("提示されたトークンの scope では要求された操作に不足している (RFC 6750 §3.1 insufficient_scope)。")
model InsufficientScopeError {

}


@doc("AuthorizationDetailType のスキーマ定義が不正 (検証失敗、または未知の Semantics 値の参照)。")
model InvalidAuthorizationDetailTypeError {

}


@doc("Device Authorization Grant において device_code が有効期限を過ぎた (RFC 8628 §3.5 expired_token)。")
model ExpiredTokenError {

}


@doc("Device Authorization Grant のポーリングが速すぎる。クライアントは interval を 5 秒延長する必要がある (RFC 8628 §3.5 slow_down)。")
model SlowDownError {

}


@doc("Device Authorization Grant においてユーザーがまだ承認していない (RFC 8628 §3.5 authorization_pending)。クライアントは interval 秒待ってから再試行する。")
model AuthorizationPendingError {

}


@doc("CIBA の /bc-authorize において login_hint / id_token_hint から承認対象 User を解決できない (CIBA Core §13 unknown_user_id)。User の存在有無を error の差で開示しないため、非 active・別テナントも同じ error に畳む。")
model UnknownUserIdError {

}


@doc("CIBA の /bc-authorize において binding_message が長すぎる、または表示に適さない (CIBA Core §13 invalid_binding_message)。")
model InvalidBindingMessageError {

}


@doc("AuthorizationServer 側の予期せぬ内部エラー (RFC 6749 §4.1.2.1 server_error)。")
model ServerError {

}


@doc("route ごとの fixed-window rate limit 設定。閾値・窓は環境変数で運用者が調整する (このモデルは形だけを固定し、具体値は仕様に埋め込まない)。")
model EndpointRateLimitPolicy {
  @doc("対象 route の識別子 (例: token、authorize、par、device_authorization、password_reset、login)。")
  policy_id: string;
  max_requests: int64;
  window_seconds: int64;
  @doc("共有カウンタストア到達不能時の挙動。fail_closed 固定。")
  fail_mode: string;
}


@doc("/register で新しいクライアントが動的登録された (RFC 7591)。クライアント登録監査の起点。")
model ClientRegistered {
  occurredAt: utcDateTime;
  @doc("クライアントが属するテナント ID。")
  tenantId: string;
  clientId: string;
  clientType?: ClientType;
}


@doc("URL形式のclient_idについてClient ID Metadata Documentのfetch・検証に成功し、認可リクエストの処理を継続した。永続登録は伴わない。")
model ClientIdMetadataDocumentResolved {
  occurredAt: utcDateTime;
  @doc("認可リクエストが属するテナント ID。")
  tenantId: string;
  @doc("fetch元のURL。")
  clientId: string;
}


@doc("URL形式のclient_idについてClient ID Metadata Documentのfetch・検証が失敗し、fail-closedでリクエストを拒否した。")
model ClientIdMetadataDocumentRejected {
  occurredAt: utcDateTime;
  @doc("認可リクエストが属するテナント ID。")
  tenantId: string;
  @doc("fetch元のURL。")
  clientId: string;
  @doc("拒否理由 (fetch失敗・JSON不正・client_id不一致・redirect_uris欠落・非対応token_endpoint_auth_method等) の分類。")
  reason: string;
}


@doc("管理者が所属テナントに OAuth client を作成した。秘密情報は含めない。")
model AdminOAuth2ClientCreated {
  occurredAt: utcDateTime;
  @doc("クライアントが属するテナント ID。actor と一致する。")
  tenantId: string;
  actorUserId: string;
  clientId: string;
}


@doc("管理者が OAuth client metadata を更新した。")
model AdminOAuth2ClientUpdated {
  occurredAt: utcDateTime;
  @doc("クライアントが属するテナント ID。actor と一致する。")
  tenantId: string;
  actorUserId: string;
  clientId: string;
  changedFields: string[];
}


@doc("管理者が所属テナントから OAuth client を削除した。")
model AdminOAuth2ClientDeleted {
  occurredAt: utcDateTime;
  @doc("クライアントが属するテナント ID。actor と一致する。")
  tenantId: string;
  actorUserId: string;
  clientId: string;
}


@doc("互換 interface で管理者が secret-based client の credential を rotation または即時 revoke した。secret と hash は payload に含めない。")
model ClientSecretRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  graceUntil?: utcDateTime;
  revokedImmediately: boolean;
}


@doc("ResourceOwner がクライアントに対する scope の同意を与えた (GDPR Art.6 法的根拠の取得記録)。")
model ConsentGranted {
  occurredAt: utcDateTime;
  @doc("同意が起きたテナント ID。")
  tenantId: string;
  userId: string;
  clientId: string;
  scopes: string[];
}


@doc("ResourceOwner または所属テナントの Administrator が既存の同意を撤回した (GDPR Art.7(3) 同意撤回権)。以後の発行はブロックされる。")
model ConsentRevoked {
  occurredAt: utcDateTime;
  @doc("同意が属するテナント ID。")
  tenantId: string;
  @doc("管理 API からの撤回時の Administrator sub。ResourceOwner 自身の撤回では省略可。")
  actorUserId?: string;
  userId: string;
  clientId: string;
}


@doc("RFC 9396 — クライアントが /authorize・/par・/token で authorization_details を要求し、登録 type 適合検証を通過した。")
model AuthorizationDetailsRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  clientId: string;
  @doc("要求時点で確定していれば principal の sub。")
  userId?: string;
  @doc("要求された詳細の type 一覧。")
  detailTypes: string[];
}


@doc("RFC 9396 — ResourceOwner が提示された authorization_details に同意した。発行・交換の部分集合判定の基準となる。")
model AuthorizationDetailsConsented {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  clientId: string;
  detailTypes: string[];
}


@doc("RFC 9396 — authorization_details の構文不正・未登録 type・スキーマ不適合・同意逸脱・ダウンスコープ超過により要求を拒否した (fail-closed)。")
model AuthorizationDetailsRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  clientId: string;
  @doc("拒否理由 (invalid_authorization_details 等)。")
  reason: string;
}


@doc("/authorize 完了時に AuthorizationCode を発行した。/token への交換まで 60 秒以内に行う必要がある。")
model AuthorizationCodeIssued {
  occurredAt: utcDateTime;
  @doc("認可コードを発行したテナント ID。")
  tenantId: string;
  clientId: string;
  userId: string;
  scopes: string[];
  codeChallengeMethod?: string;
}


@doc("AuthorizationCode が /token で正しく交換された。同一コードの再使用はファミリー失効を引き起こす。")
model AuthorizationCodeRedeemed {
  occurredAt: utcDateTime;
  @doc("交換が起きたテナント ID。")
  tenantId: string;
  clientId: string;
  userId: string;
}


@doc("AccessToken (JWT) を発行した。senderConstraint がある場合は所有証明（DPoP/mTLS）バインド済み。")
model AccessTokenIssued {
  occurredAt: utcDateTime;
  @doc("トークンを発行したテナント ID。")
  tenantId: string;
  jti: string;
  clientId: string;
  userId: string;
  scopes: string[];
  senderConstraint: AccessTokenSenderConstraint = AccessTokenSenderConstraint.None;
}


@doc("RefreshToken を発行した。familyId は元の認可コード由来のファミリーを識別し、ローテーションで継承される。")
model RefreshTokenIssued {
  occurredAt: utcDateTime;
  @doc("トークンを発行したテナント ID。")
  tenantId: string;
  tokenId: string;
  familyId: string;
  parentId?: string;
  clientId: string;
  userId: string;
}


@doc("RefreshToken を /token で交換し、新しい子トークンを発行して親を Rotated 状態にした。")
model RefreshTokenRotated {
  occurredAt: utcDateTime;
  @doc("ローテーションが起きたテナント ID。")
  tenantId: string;
  oldTokenId: string;
  newTokenId: string;
  familyId: string;
}


@doc("/revoke もしくは管理操作によってトークンが失効された (RFC 7009)。監査・即時性要件あり。")
model TokenRevoked {
  occurredAt: utcDateTime;
  @doc("失効が起きたテナント ID。")
  tenantId: string;
  tokenType: TokenType;
  tokenId: string;
  reason?: string;
}


@doc("/introspect が呼ばれてトークン状態を返した (RFC 7662)。ResourceServer の利用パターン分析に使用。")
model TokenIntrospected {
  occurredAt: utcDateTime;
  @doc("introspection が起きたテナント ID。")
  tenantId: string;
  rsClientId: string;
  tokenId: string;
  active: boolean;
}


@doc("RFC 8693 — トークン交換による委任トークンを発行した。delegationDepth は発行トークンの act 入れ子の深さ。")
model TokenExchanged {
  occurredAt: utcDateTime;
  @doc("トークン交換が起きたテナント ID。")
  tenantId: string;
  @doc("発行トークンの現在アクター (act.sub)。")
  actorUserId: string;
  @doc("発行トークンの subject (sub)。subject_token の sub を維持する。")
  subjectUserId: string;
  @doc("resource パラメータ由来の発行トークン aud。")
  audience: string;
  @doc("発行トークンの act 入れ子の深さ。")
  delegationDepth: int64;
}


@doc("RFC 8693 — トークン交換要求をポリシー違反・検証失敗により拒否した (fail-closed)。")
model TokenExchangeRejected {
  occurredAt: utcDateTime;
  @doc("トークン交換が拒否されたテナント ID。")
  tenantId: string;
  @doc("拒否時に判明している現在アクター。不明な場合は空。")
  actorUserId?: string;
  @doc("拒否理由 (OAuth error code 等)。")
  reason: string;
}


@doc("RFC 9728 — /.well-known/oauth-protected-resource が登録済み McpResourceServer の metadata を配信した。")
model ProtectedResourceMetadataServed {
  occurredAt: utcDateTime;
  @doc("metadata を配信したテナント ID。")
  tenantId: string;
  @doc("配信対象の resource URI。")
  resource: string;
}


@doc("RFC 8707 — resource indicator に基づき audience を単一 McpResourceServer へ厳格限定した Access Token を発行した。Authorize/PAR/Token(authorization_code) 経路が対象。token-exchange 経由の resource 限定発行は TokenExchanged.audience で表現し、本イベントは emit しない。")
model ResourceScopedTokenIssued {
  occurredAt: utcDateTime;
  @doc("トークンを発行したテナント ID。")
  tenantId: string;
  clientId: string;
  @doc("発行トークンの aud に束縛された resource URI。")
  resource: string;
  scopes: string[];
}


@doc("RFC 8707 — resource indicator が未登録・Disabled・複数指定のため fail-closed で拒否した。Authorize/PAR/Token(authorization_code) 経路が対象。token-exchange の拒否は TokenExchangeRejected.reason で表現し、本イベントは emit しない。")
model ResourceAudienceRejected {
  occurredAt: utcDateTime;
  @doc("拒否が起きたテナント ID。")
  tenantId: string;
  clientId: string;
  @doc("拒否理由 (invalid_target 等)。")
  reason: string;
}


@doc("Rotated 状態の RefreshToken が再提示された。ファミリー全失効をトリガーする (RFC 9700 §4.14)。")
model RefreshTokenReuseDetected {
  occurredAt: utcDateTime;
  @doc("再利用が検出されたテナント ID。")
  tenantId: string;
  familyId: string;
  tokenId: string;
  clientId?: string;
}


@doc("/par に届いた authorize パラメータをサーバー側に格納し短命の request_uri を発行した (RFC 9126)。")
model PARStored {
  occurredAt: utcDateTime;
  @doc("PAR を受け付けたテナント ID。")
  tenantId: string;
  requestUri: string;
  clientId: string;
}


@doc("RFC 8628 §3.1 — device_code/user_code を発行した。user_code は監査ログに残さない。")
model DeviceAuthorizationRequested {
  occurredAt: utcDateTime;
  @doc("デバイス認可を受け付けたテナント ID。")
  tenantId: string;
  clientId: string;
  scopes: string[];
}


@doc("RFC 8628 §3.3 — ユーザーが承認した。")
model DeviceAuthorizationApproved {
  occurredAt: utcDateTime;
  @doc("承認が起きたテナント ID。")
  tenantId: string;
  clientId: string;
  userId: string;
}


@doc("RFC 8628 §3.3 — ユーザーが拒否、または user_code 誤り続けた。")
model DeviceAuthorizationDenied {
  occurredAt: utcDateTime;
  @doc("拒否が起きたテナント ID。")
  tenantId: string;
  clientId: string;
  userId?: string;
}


@doc("CIBA Core §7.3 — 人間の承認を求める承認要求を起票した。auth_req_id は bearer secret のため監査ログに残さない。")
model BackchannelAuthRequested {
  occurredAt: utcDateTime;
  @doc("承認要求を受け付けたテナント ID。")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  agentId?: string;
  userId: string;
  scopes: string[];
}


@doc("CIBA Core §7.3 — 承認対象 User が承認要求を承認した。")
model BackchannelAuthApproved {
  occurredAt: utcDateTime;
  @doc("承認が起きたテナント ID。")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  userId: string;
}


@doc("CIBA Core §7.3 — 承認対象 User が承認要求を拒否した。")
model BackchannelAuthDenied {
  occurredAt: utcDateTime;
  @doc("拒否が起きたテナント ID。")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  userId: string;
}


@doc("CIBA Core §11 — 承認要求が判断されないまま有効期限に到達した。")
model BackchannelAuthExpired {
  occurredAt: utcDateTime;
  @doc("期限切れが起きたテナント ID。")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  userId: string;
}


@doc("承認要求を承認済みトークンへ一度きり消費する lifecycle event。")
model Consume {
  occurredAt: utcDateTime;
}


@doc("認可要求を検証する lifecycle event。")
model Validate {
  occurredAt: utcDateTime;
}


@doc("lifecycle を拒否終端へ進める event。")
model Reject {
  occurredAt: utcDateTime;
}


@doc("resource owner 認証が成立した event。")
model AuthenticateUser {
  occurredAt: utcDateTime;
}


@doc("resource owner の同意を要求する event。")
model RequestConsent {
  occurredAt: utcDateTime;
}


@doc("resource owner が同意した event。")
model GrantConsent {
  occurredAt: utcDateTime;
}


@doc("authorization code を発行する lifecycle event。")
model IssueCode {
  occurredAt: utcDateTime;
}


@doc("authorization code を単発償還する lifecycle event。")
model RedeemCode {
  occurredAt: utcDateTime;
}


@doc("有効期限到達を表す lifecycle event。")
model Expire {
  occurredAt: utcDateTime;
}


@doc("device user code が入力された event。")
model EnterUserCode {
  occurredAt: utcDateTime;
}


@doc("device authorization が承認された event。")
model Approve {
  occurredAt: utcDateTime;
}


@doc("device authorization が拒否された event。")
model Deny {
  occurredAt: utcDateTime;
}


@doc("device code が token に交換された event。")
model Exchange {
  occurredAt: utcDateTime;
}


@doc("refresh token を child token へ rotate する event。")
model Rotate {
  occurredAt: utcDateTime;
}


@doc("token を失効させる lifecycle event。")
model RevokeToken {
  occurredAt: utcDateTime;
}


@doc("consent を論理撤回する lifecycle event。")
model RevokeConsent {
  occurredAt: utcDateTime;
}


@doc("PAR request_uri を単発消費する lifecycle event。")
model Use {
  occurredAt: utcDateTime;
}


@doc("LogoutNotification の配送が成功した lifecycle event。")
model Deliver {
  occurredAt: utcDateTime;
}


@doc("LogoutNotification の配送が max_attempts に到達し dead-letter へ確定した lifecycle event。")
model Exhaust {
  occurredAt: utcDateTime;
}

}
