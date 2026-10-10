namespace IdMagic.Contract {

@doc("エンドユーザーのアカウントポータルが返す、接続済みのアプリケーションの一つ（セルフサービス）。呼び出し元自身の `sub` に固定されるので、`sub` も `tenant_id` も現れない。表示のために、client_id、表示名、付与したスコープ、付与と期限の時刻を返す。")
model AccountConsentResponse {
  client_id: string;
  @doc("client_id から解決した表示名。`OAuth2Client.client_name`、Application のカタログの名前の順に解決し、どちらもなければ client_id そのものを使う。")
  client_name: string;
  scopes: string[];
  state: ConsentState;
  granted_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("EndpointRateLimitPolicy の閾値を超えた。これは届いたすべてのリクエストを数える汎用のエンドポイントのレート制限で、アカウントごとと IP ごとに失敗を数えるログインの抑制とは別である。フェイルクローズで、共有のカウンターのストアに到達できないときもリクエストを拒否する。ボディが `retry_after_seconds` を含むので、Problem Details ではなく、`support_http.WriteRateLimited` が `Content-Type: application/json` で書く。レスポンスは同じ秒数の `Retry-After` ヘッダーも含む。")
model RateLimitedError {
  @doc("常に `rate_limited` である。")
  error: string;
  @doc("再試行の前に待つ秒数。`Retry-After` のレスポンスヘッダーにも繰り返す。")
  retry_after_seconds: int32;
  @doc("コードでローカライズしない呼び出し元のための、英語の代わりのテキスト。ローカライズする UI は、代わりに `rate_limited` を自身のメッセージへ対応付ける。")
  message: string;
}


@doc("SPA の認証のステップの後に示す次の画面、またはクライアントのリダイレクト URI。")
model BrowserFlowResponse {
  next?: string;
  redirect_to?: url;
}


@doc("認可エンドポイントが redirect_uri へコードを届ける方法。送受信の値は OAuth 2.0 Multiple Response Type Encoding Practices と OAuth 2.0 Form Post Response Mode に従う。")
enum ResponseMode {
  Query: "query",
  FormPost: "form_post",
}


@doc("`token_type_hint` と失効の監査イベントが運ぶトークンの種類。送受信の値は RFC 7009 の 2.1 節に従う。")
enum TokenType {
  AccessToken: "access_token",
  RefreshToken: "refresh_token",
}


@doc("トークンを保持者にどう結び付けるか（RFC 7800）。`dpop` は DPoP の公開鍵のサムプリント、`mtls` は mTLS のクライアント証明書のサムプリントである。")
enum SenderConstraintKind {
  Dpop: "dpop",
  Mtls: "mtls",
}


@doc("発行するアクセストークンに適用する送信者の制約。`none` は bearer のトークンを意味する。")
enum AccessTokenSenderConstraint {
  None: "none",
  Dpop: "dpop",
  Mtls: "mtls",
}


@doc("PKCE の `code_challenge` のハッシュの方式（RFC 7636）。このアプリケーションは S256 だけを許し、`plain` は弱いとして拒否する。")
enum CodeChallengeMethod {
  S256: "S256",
}


@doc("AuthorizationCodeFlow の状態機械がとりうる状態。状態機械の節も参照する。")
enum AuthorizationCodeState {
  Received: "received",
  AuthenticationPending: "authentication_pending",
  Authenticated: "authenticated",
  ConsentPending: "consent_pending",
  Consented: "consented",
  CodeIssued: "code_issued",
  Exchanged: "exchanged",
  Rejected: "rejected",
  Expired: "expired",
}


@doc("DeviceCodeFlow の状態機械（RFC 8628）がとりうる状態。状態機械の節も参照する。")
enum DeviceCodeState {
  Issued: "issued",
  UserCodeEntered: "user_code_entered",
  Approved: "approved",
  Denied: "denied",
  Exchanged: "exchanged",
  Expired: "expired",
}


@doc("ApprovalRequestLifecycle の状態機械がとりうる状態。CIBA のトランスポートの語彙ではなく、「事前条件がまだ満たされていない」ことの一般的な形として保持する。`consumed` は `approved` からだけ到達できる終端の状態で、トークンへの一度だけの交換を表す。状態機械の節も参照する。")
enum ApprovalRequestState {
  Pending: "pending",
  Approved: "approved",
  Denied: "denied",
  Expired: "expired",
  Consumed: "consumed",
}


@doc("RefreshTokenLifecycle の状態機械がとりうる状態。`rotated` は子のトークンへ使われたこと、`revoked` は失効したこと、`expired` は期限が切れたことを表す。")
enum RefreshTokenState {
  Active: "Active",
  Rotated: "Rotated",
  Revoked: "Revoked",
  Expired: "Expired",
}


@doc("AuthorizationCodeRecordLifecycle の状態機械がとりうる状態。AuthorizationCodeFlow とは別の軸で、コードそのもののライフサイクルを記述する。")
enum AuthorizationCodeRecordState {
  Issued: "issued",
  Redeemed: "redeemed",
  Expired: "expired",
}


@doc("ConsentLifecycle の状態機械がとりうる状態。GDPR 第 7 条 3 項の撤回の権利により、`granted` から `revoked` への遷移がありうる。")
enum ConsentState {
  Granted: "granted",
  Revoked: "revoked",
  Expired: "expired",
}


@doc("PARRecordLifecycle の状態機械がとりうる状態。`stored` から、`/authorize` が読むと `used` に、期限が切れると `expired` になる。")
enum PARRecordState {
  Stored: "Stored",
  Used: "Used",
  Expired: "Expired",
}


@doc("登録した authorization_details のスキーマで、各フィールドが持つ絞り込みの半順序。`set` は集合の包含（リクエストが同意または元の部分集合である）、`at_most` は単調な減少（リクエストが同意または元以下である）、`enum` は許可リストとの完全一致、`exact` は不透明な値の完全一致である。判定できないものはフェイルクローズで拒否の側に倒す。")
enum AuthorizationDetailFieldSemantics {
  set: "set",
  at_most: "at_most",
  enum_: "enum",
  exact: "exact",
}


@doc("AuthorizationDetailType の運用上の状態。受け付けるのは `enabled` だけで、`disabled` は新しいリクエストを拒否し、発行済みのトークンは期限が切れるまで有効なままである。")
enum AuthorizationDetailTypeState {
  Enabled: "Enabled",
  Disabled: "Disabled",
}


@doc("OAuth 2.0 / OpenID Connect のプロトコルのエラーコード。OAuthError のボディの `error` のメンバーで運ぶ。送受信の値は、RFC 6749 の 4.1.2.1 節と 5.2 節、RFC 6750 の 3.1 節（invalid_token、insufficient_scope）、RFC 8628 の 3.5 節（authorization_pending、slow_down、expired_token）、RFC 8707 の 2 節（invalid_target）、RFC 9449 の 7.1 節（invalid_dpop_proof）、RFC 7591 の 3.2.2 節（invalid_client_metadata、invalid_redirect_uri）、RFC 9126 の 2.2 節（invalid_request_uri）、RFC 9396 の 5 節（invalid_authorization_details）、OpenID Connect Core の 3.1.2.6 節（login_required）、OpenID Connect CIBA Core の 13 節（unknown_user_id、invalid_binding_message）が登録する。")
enum OAuthErrorCode {
  InvalidRequest: "invalid_request",
  InvalidClient: "invalid_client",
  InvalidGrant: "invalid_grant",
  UnauthorizedClient: "unauthorized_client",
  UnsupportedGrantType: "unsupported_grant_type",
  InvalidScope: "invalid_scope",
  InvalidToken: "invalid_token",
  ServerError: "server_error",
  InvalidDpopProof: "invalid_dpop_proof",
  AccessDenied: "access_denied",
  ExpiredToken: "expired_token",
  SlowDown: "slow_down",
  InsufficientScope: "insufficient_scope",
  AuthorizationPending: "authorization_pending",
  InvalidTarget: "invalid_target",
  UnsupportedResponseType: "unsupported_response_type",
  LoginRequired: "login_required",
  InvalidRequestUri: "invalid_request_uri",
  InvalidClientMetadata: "invalid_client_metadata",
  InvalidRedirectUri: "invalid_redirect_uri",
  InvalidAuthorizationDetails: "invalid_authorization_details",
  UnknownUserId: "unknown_user_id",
  InvalidBindingMessage: "invalid_binding_message",
}


@doc("FAPI 2.0 の拡張を備えた RFC 7591 のクライアント。`application_id` が null のカタログの外のクライアントも許す。null でないときは同じテナントの `oidc` の Application を指し、一つの Application に結び付くクライアントは一つまでである。")
model OAuth2Client {
  @doc("属するテナント。")
  tenant_id: string = "default";
  @minLength(1)
  @maxLength(128)
  client_id: string;
  @doc("対応する Application。通常の CRUD はこれを変えず、関係を管理するのは Application の一括作成と一括削除だけである。")
  application_id?: string;
  client_secret_hash?: string;
  @maxLength(200)
  client_name?: string;
  client_type: ClientType;
  @doc("リダイレクトを使うグラント（authorization_code）では必須である。client_credentials だけを使うマシン間のクライアントにはない（RFC 6749 の 3.1.2 節）。")
  redirect_uris?: url[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  jwks_uri?: url;
  jwks?: JWKS;
  tls_client_auth_subject_dn?: string;
  id_token_signed_response_alg: SignatureAlgorithm = SignatureAlgorithm.PS256;
  require_pushed_authorization_requests: boolean = false;
  require_pkce?: boolean;
  dpop_bound_access_tokens: boolean = false;
  fapi_profile: FapiProfile = FapiProfile.None;
  first_party: boolean = false;
  @doc("OpenID Connect Back-Channel Logout 1.0 の通知の宛先。設定すると、セッションの失効のたびに、署名したログアウトトークンをこの URI へ POST する。")
  backchannel_logout_uri?: url;
  @doc("RP がログアウトトークンに `sid` のクレームを求めるかどうか。true のときは、`sid` のない対象のセッションには何も配送しない。ブラウザーのセッションのない発行には、通知する対象がないためである。")
  backchannel_logout_session_required: boolean = false;
  @doc("OpenID Connect Front-Channel Logout 1.0 の通知の宛先。`/end_session` のレスポンスは、この URI を指す iframe を含む。")
  frontchannel_logout_uri?: url;
  @doc("iframe の URI に `iss` と `sid` のクエリパラメーターを加えるかどうか。")
  frontchannel_logout_session_required: boolean = false;
  @doc("このクライアントに対するクレームの発行のポリシーで、アプリケーションごとの開示の上書きである。NameID の役割を果たす取得元の属性を `sub` の取得元に使う。ClaimExposed の属性をそもそも開示してよいかは、標準のスコープ（openid、profile、email など）によるスコープと同意で決まり、このポリシーとは独立に先に適用する。このポリシーはその範囲の中で絞り込みと対応付け直しを行い、ID トークンと UserInfo に合わせる。")
  claim_policy: ClaimMappingPolicy;
  created_at: utcDateTime;
  @doc("作成時は `created_at` と等しく、以降はクライアントのメタデータに対する管理操作のたびに更新する。")
  updated_at: utcDateTime;
}


@doc("シークレット方式の OAuth2 のクライアントの認証の資格情報。平文は発行のレスポンスでだけ返し、どのストレージ、イベント、一覧にも現れない。新しいものを発行しても既存の資格情報は変えず、個別に失効させるか期限が切れるまで一致し続ける。移行に停止時間が要らないように、有効期限のない既存の資格情報も許す。")
model ClientSecretCredential {
  id: string;
  client_id: string;
  secret_hash: string;
  created_at: utcDateTime;
  @doc("資格情報が期限切れになる時刻。新たに加える資格情報では必須である。null は、移行前の有効期限のない従来の資格情報、または互換のローテーションの現在の資格情報を表す。")
  expires_at?: utcDateTime;
  @doc("完全に失効させた時刻。`revoked_at` のある資格情報は、`expires_at` にかかわらず認証に使わない。")
  revoked_at?: utcDateTime;
}


@doc("GDPR の同意の記録。状態は ConsentLifecycle の状態機械に従う。`user_id` と `client_id` は既存の User と OAuth2Client を指す。")
model Consent {
  user_id: string;
  client_id: string;
  scopes: string[];
  state: ConsentState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  granted_at: utcDateTime;
  @doc("通常は `granted_at` に 365 日を加えたもの。")
  expires_at: utcDateTime;
  @doc("状態が `revoked` になった時刻。監査のためのものである。")
  revoked_at?: utcDateTime;
  @doc("RFC 9396：ユーザーが同意した構造化された詳細。発行と交換のときの部分集合の検査の基準である。")
  authorization_details?: AuthorizationDetail[];
}


@doc("RFC 9396 の `authorization_details` の一要素。`type` で識別する、構造化された細かい権限である。リクエスト、同意、トークンへの反映を通じて一貫して扱い、登録した AuthorizationDetailType に照らしてフェイルクローズで検証する。")
model AuthorizationDetail {
  @doc("登録した AuthorizationDetailType の識別子。たとえば `payment_initiation`。")
  @minLength(1)
  type: string;
  @doc("RFC 9396：詳細が対象とするリソースサーバーの URI。")
  locations?: string[];
  @doc("RFC 9396：対象に対して許す操作。")
  actions?: string[];
  @doc("RFC 9396：アクセスするデータの種類。")
  datatypes?: string[];
  @doc("RFC 9396：対象のリソースの識別子。")
  identifier?: string;
  @doc("RFC 9396：対象に対する権限の水準。")
  privileges?: string[];
  @doc("種類に固有の構造化されたフィールド。たとえば `instructedAmount` と `creditorAccount`。登録したスキーマがその意味を定める。")
  fields?: Record<unknown>;
}


@doc("AuthorizationDetailsSchema の一つのフィールドの規則。絞り込みの半順序と許可する値を定める。")
model AuthorizationDetailFieldRule {
  @doc("規則が扱うフィールド。`actions` のような標準のフィールド、または `fields` のマップの中のキーである。")
  @minLength(1)
  name: string;
  @doc("部分集合の検査に使う半順序。")
  semantics: AuthorizationDetailFieldSemantics;
  @doc("true のとき、詳細はこのフィールドを含まなければならない。")
  required: boolean = false;
  @doc("`semantics` が `set` または `enum` のときに許可する値の閉じた集合。空は制限なしを意味する。")
  allowed?: string[];
}


@doc("一つの種類の構造のスキーマ。受け付けるフィールドと、それぞれの絞り込みの半順序を並べる。未登録のフィールド、スキーマとの不一致、許可する集合の外の値は、いずれもフェイルクローズで拒否する。")
model AuthorizationDetailsSchema {
  @doc("フィールドの規則。")
  rules: AuthorizationDetailFieldRule[];
}


@doc("テナントが登録する `authorization_details` の種類の定義。種類の識別子、検証のスキーマ、同意の UI のための表示のテンプレートをまとめる。受け付けるのは `enabled` の種類だけである。管理は AdminAuthorizationDetailTypesManage が守る。")
model AuthorizationDetailType {
  @doc("属するテナント。")
  tenant_id: string = "default";
  @doc("`authorization_details` の種類の識別子。")
  @minLength(1)
  type: string;
  description?: string;
  schema: AuthorizationDetailsSchema;
  @doc("同意の UI で詳細を人に示すテンプレート。たとえば `up to {fields.instructedAmount} to {identifier}`。")
  display_template: string;
  state: AuthorizationDetailTypeState = AuthorizationDetailTypeState.Enabled;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("McpResourceServer の運用上の状態。リソースインジケーターの検査を通るのは `active` だけで、`disabled` は新たな発行を拒否し、発行済みのトークンは期限が切れるまで有効なままである。")
enum McpResourceServerState {
  Active: "Active",
  Disabled: "Disabled",
}


@doc("MCP のリソースサーバー（ツールまたはデータソース）の、テナントに属する登録。正規のリソースの URI と許可するスコープを管理し、Protected Resource Metadata（RFC 9728）とリソースインジケーターの検査（RFC 8707）の基準である。管理は AdminMcpResourcesManage が守る。")
model McpResourceServer {
  @doc("属するテナント。")
  tenant_id: string = "default";
  @doc("登録の一意な識別子（UUID）。テナントをまたいで全体で一意である。")
  @minLength(1)
  id: string;
  @doc("発行するトークンの `aud` に厳密に結び付く正規のリソースの URI。テナントの中で一意である。")
  resource: ResourceIndicator;
  @doc("管理 UI での表示名。")
  @minLength(1)
  @maxLength(200)
  name: string;
  @doc("このリソースにアクセスするときに許すスコープの許可リスト。`resource` とともに求めるスコープは、その部分集合でなければならない。")
  scopes: string[];
  state: McpResourceServerState = McpResourceServerState.Active;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("RFC 8707 の `resource` パラメーター。求めるオーディエンスを指す、フラグメントのない絶対 URI である。トークンごとに一つだけ許し、複数を指定すると `invalid_target` で拒否する。")
model ResourceIndicator {
  @minLength(1)
  value: url;
}


@doc("RFC 9728 の Protected Resource Metadata。レルムの IdMagic API では ApiTokenScopes から、外部のリソースでは McpResourceServer の登録から導く。")
model ProtectedResourceMetadata {
  resource: url;
  @doc("このリソースを守る認可サーバー、すなわち IdMagic の発行者。")
  authorization_servers: url[];
  scopes_supported: string[];
  bearer_methods_supported: string[] = #["header"];
  dpop_signing_alg_values_supported?: string[];
}


@doc("管理者が McpResourceServer を登録または更新する JSON のリクエスト。`resource` は作成時にだけ設定できる。")
model McpResourceServerRequest {
  @doc("作成時は必須で、更新では変えられない。変えると、発行済みのトークンの `aud` の意味が変わってしまうためである。")
  resource?: string;
  @minLength(1)
  @maxLength(200)
  name: string;
  scopes: string[];
  state?: McpResourceServerState;
}


@doc("McpResourceServer の表現。管理 UI の一覧と詳細の表示は同じ表現を返す。")
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


@doc("管理 API の McpResourceServer の一覧。")
model McpResourceServerListResponse {
  resource_servers: McpResourceServerResponse[];
}


@doc("リフレッシュトークンのストアの記録。平文の値は保存しない。状態は RefreshTokenLifecycle の状態機械に従う。テナントはトークンの行に重複して保持せず、全体で識別される User と OAuth2Client から導き、両者がテナントの中にあるという不変条件はユースケースが確かめる。")
model RefreshTokenRecord {
  id: string;
  @doc("SHA-256 のハッシュ。")
  hash: string;
  @doc("一回の認可コードの交換から派生したすべてのトークンが共有する識別子。")
  family_id: string;
  parent_id?: string;
  client_id: string;
  user_id: string;
  scopes: string[];
  state: RefreshTokenState;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  @doc("絶対の期限。ローテーションでは延びない。")
  absolute_expires_at: utcDateTime;
  sender_constraint?: RefreshTokenSenderConstraint;
  @doc("元の AuthorizationCodeRecord の `sid`（OIDC のセッションの id）。ローテーションは親の値をそのまま引き継ぐ。null は、client_credentials のようなブラウザーのセッションのない発行を意味する。値が Authentication の SessionEnded の `sessionId` と一致するすべてのファミリー（複数のクライアントにまたがることがある）をこのフィールドで解決するので、セッションの失効はそれらをまとめて RevokeToken へ移す。")
  sid?: string;
  @doc("元のリソースインジケーター（`AuthorizationCodeRecord.resource`）。ローテーションは親の値をそのまま引き継ぐので、以降もオーディエンスは同じ McpResourceServer に厳密に限られる。null は `resource` のない発行を意味する。")
  resource?: ResourceIndicator;
}


@doc("あるブラウザーのセッション（`sid`）が特定の OAuth2Client に認証の結果を発行したことの記録（RP の参加）。`sid` は Authentication の `LoginSession.id` に等しく、RP の間で共有する。セッションの状態そのものはここに写さず、セッションを失効させたときにフロントチャネルとバックチャネルのログアウトで通知する RP を解決するための索引としてだけ存在する。")
model ClientSession {
  @doc("OIDC のセッションの id。`LoginSession.id` に等しい。")
  sid: string;
  client_id: string;
  @doc("属するテナント。")
  tenant_id: string = "default";
  first_issued_at: utcDateTime;
  @doc("同じ RP への再発行（再認可）のたびに更新する。通知先の解決には影響しない。")
  last_issued_at: utcDateTime;
}


@doc("一回のバックチャネルログアウトの配送の試行の記録。この記録を作る前に、ローカルのセッションとリフレッシュトークンの失効はすでに確定しており、配送の失敗はそれを元に戻さない。配送そのものは、Jobs の境界づけられたコンテキストの永続ジョブ（`kind=backchannel_logout_delivery`）に委ねる。")
model LogoutNotification {
  id: string;
  @doc("属するテナント。")
  tenant_id: string = "default";
  @doc("通知の対象の OIDC のセッションの id。")
  sid: string;
  @doc("通知する RP。")
  client_id: string;
  @doc("発行したログアウトトークンの `jti`。RP はリプレイの検出に、IdMagic は監査での関連付けに使う。")
  logout_token_jti: string;
  @doc("配送先。配送を始めた時点の、クライアントの `backchannel_logout_uri` のスナップショットである。")
  target_uri: url;
  state: LogoutNotificationState;
  attempts: int64 = 0;
  last_error?: string;
  @doc("配送の実行を関連付ける、Jobs モジュールの `JobRef.id`。")
  job_id?: string;
  created_at: utcDateTime;
  delivered_at?: utcDateTime;
}


@doc("LogoutNotification の配送の状態。Jobs の JobStatus とは一対一に対応せず、配送そのものが成功したかどうかだけを示し、ジョブ自身の再試行とリースは Jobs が扱う。")
enum LogoutNotificationState {
  Pending: "Pending",
  Delivered: "Delivered",
  Failed: "Failed",
}


@doc("フローを追跡するサーバー内部の中間の状態。")
model AuthorizationRequest {
  id: string;
  @doc("属するテナント。")
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
  @doc("認証が成功した方式。`advance(authenticate_user)` が AuthenticationContext から伝える。")
  amr?: string[];
  @doc("AuthenticationContextPolicy に従って amr から導いた URN。")
  acr?: string;
  @doc("クライアントが `/authorize` に与えた `acr_values` パラメーター（RFC 8176）。ステップアップを決める。")
  acr_values?: string;
  @doc("`advance(authenticate_user)` が `AuthenticationContext.session_id` から伝える OIDC のセッションの id で、Authentication モジュールの `LoginSession.id` に等しい。AuthorizationCodeRecord、RefreshTokenRecord、IdTokenClaims へ引き継ぎ、セッションの失効が始めるファミリーの失効と、フロントチャネルとバックチャネルのログアウトの両方を関連付ける。")
  sid?: string;
  @doc("RFC 9396：クライアントが求めた構造化された詳細。検証済みで、登録した種類に合うものだけを保持する。")
  authorization_details?: AuthorizationDetail[];
  created_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("発行した認可コードの永続的な記録。コードそのものは短命である。状態は AuthorizationCodeRecordLifecycle に従い、これは AuthorizationCodeFlow とは別の軸で、コードそのもののライフサイクルを記述する。")
model AuthorizationCodeRecord {
  code: string;
  @doc("属するテナント。")
  tenant_id: string = "default";
  authorization_request_id: string;
  client_id: string;
  user_id: string;
  scopes: string[];
  redirect_uri: url;
  code_challenge: string;
  code_challenge_method: CodeChallengeMethod;
  nonce?: string;
  auth_time: int64;
  @doc("認可リクエストから引き継いだ認証の方式。ID トークンとアクセストークンに出力する。")
  amr?: string[];
  @doc("認可リクエストから引き継いだ ACR の URN。ID トークンとアクセストークンに出力する。")
  acr?: string;
  @doc("認可リクエストから引き継いだ OIDC のセッションの id。ID トークンの `sid` のクレームと、発行する RefreshTokenRecord の `sid` へ伝わる。")
  sid?: string;
  @doc("RFC 9396：同意した構造化された詳細。交換はこれを発行するトークンに反映する。")
  authorization_details?: AuthorizationDetail[];
  state: AuthorizationCodeRecordState;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
  @doc("状態が `redeemed` になった時刻。監査のためのものである。")
  redeemed_at?: utcDateTime;
  @doc("成功した交換が発行したリフレッシュのファミリー。リプレイを検出したときにファミリーを失効させるのに使う。")
  issued_family_id?: string;
}


@doc("Pushed Authorization Request の保存した記録。状態は PARRecordLifecycle に従う。")
model PARRecord {
  request_uri: string;
  @doc("属するテナント。")
  tenant_id: string = "default";
  client_id: string;
  parameters: Record<string>;
  state: PARRecordState;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("RFC 8628 のデバイス認可の記録。`device_code` はハッシュだけを保存する。")
model DeviceAuthorization {
  device_code_hash: string;
  @doc("属するテナント。")
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
  issued_family_id?: string;
  issued_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("人が確定するまでクライアントのリクエストを保留する承認のリクエスト。判断の状態と内容は CIBA のトランスポートの管理情報とは分けて保持するが、競合する判断と問い合わせが一つのストアの境界の中で直列化するように、同じ記録に置く。`id` は UUID で、`auth_req_id` は SHA-256 のハッシュだけを索引として保存する。")
model ApprovalRequest {
  id: string;
  @doc("属するテナント。")
  tenant_id: string = "default";
  @doc("承認のリクエストを起こしたクライアント。")
  client_id: string;
  @doc("クライアントに結び付いた Agent があれば、その Agent。承認の画面と、発行するトークンの `agent_id` のクレームに使う。")
  agent_id?: string;
  @doc("判断する User の `sub`。`login_hint` または `id_token_hint` から解決し、以降はその User だけが判断できる。")
  user_id: string;
  @doc("承認されたときに発行するトークンのスコープ。")
  scopes: string[];
  @doc("RFC 9396：承認の対象の構造化された権限のリクエスト。承認の画面は、人が読めるようにこれを描画する。")
  authorization_details?: AuthorizationDetail[];
  @doc("誤ったリクエストを取り違えて承認しないように、承認の画面に示す短い識別のテキスト（CIBA Core の 7.1 節、`binding_message`）。")
  @maxLength(64)
  binding_message?: string;
  state: ApprovalRequestState;
  @doc("CIBA の `auth_req_id` の SHA-256 のハッシュ。トランスポートの層の索引であり、判断の意味には関わらない。")
  auth_req_id_hash: string;
  @doc("トランスポートの管理情報。クライアントに守らせる問い合わせの最小の間隔（秒）である。`slow_down` のたびに増える。")
  interval_seconds: int64;
  @doc("トランスポートの管理情報。最後に問い合わせた時刻である。`slow_down` を決める。")
  last_polled_at?: utcDateTime;
  requested_at: utcDateTime;
  expires_at: utcDateTime;
  decided_at?: utcDateTime;
  consumed_at?: utcDateTime;
}


@doc("送信者の制約の証明（RFC 7800）。")
model SenderConstraintCnf {
  @doc("DPoP の鍵の SHA-256 のサムプリント（RFC 9449 の 6.1 節）。")
  jkt?: string;
  @doc("mTLS のクライアント証明書のサムプリント（`x5t#S256`、RFC 8705）。")
  x5t_s256?: string;
}


@doc("DPoP の証明の JWT のクレーム（RFC 9449 の 4.2 節）。`typ` は `dpop+jwt` で、公開鍵は `jwk` ヘッダーで提示する。")
model DpopProofClaims {
  @doc("証明ごとに一意な識別子。リプレイのウィンドウの中では再利用できない。")
  jti: string;
  @doc("対象のリクエストの HTTP のメソッド。")
  htm: string;
  @doc("対象のリクエストの、クエリとフラグメントを除いた HTTP の URI。")
  htu: url;
  @doc("証明を発行した時刻。許容する時計のずれの外のものは拒否する。")
  iat: utcDateTime;
  @doc("提示するアクセストークンの SHA-256 の base64url（RFC 9449 の 4.3 節）。保護されたリソースへのアクセスでは必須で、まだそのトークンのないトークンエンドポイントでは含まない。")
  ath?: string;
}


@doc("リフレッシュトークンに付く送信者の制約。発行時に結び付けたもの、すなわち DPoP の鍵のサムプリントまたは mTLS の証明書のサムプリントを保持する。")
model RefreshTokenSenderConstraint {
  type: SenderConstraintKind;
  jkt?: string;
  x5t_s256?: string;
}


@doc("JWT のアクセストークンのクレーム（RFC 9068）。TTL は 600 秒である。")
model AccessTokenClaims {
  iss: url;
  @doc("`client_credentials` では client_id に等しい。")
  sub: string;
  aud: Audience;
  client_id: string;
  @doc("空白で区切ったスコープ。")
  scope: string;
  @doc("Unix エポックからの秒数")
  exp: int64;
  iat: int64;
  nbf?: int64;
  jti: string;
  auth_time?: int64;
  acr?: string;
  amr?: string[];
  cnf?: SenderConstraintCnf;
  @doc("RFC 8693 の 4.1 節の `act` のクレーム。委任の現在の操作者を示す。`subject_token` が `act` を含んでいたときは入れ子になる。")
  act?: Record<unknown>;
  @doc("RFC 8693 の 4.4 節の `may_act` のクレーム。サブジェクトに代わって操作してよい操作者を示す。")
  may_act?: Record<unknown>;
  @doc("RFC 9396：トークンに結び付く構造化された詳細。リソースサーバーの検査点である。後方互換のために任意とする。")
  authorization_details?: AuthorizationDetail[];
  @doc("トークンを発行した対象の Agent の id。発行するクライアントが Agent に結び付いているときだけ含む。トークンが人でない主体のものであることを示し、イントロスペクションはこれを読んで委任のモードを導く（REQ-OAUTH2-049）。Agent の種類については何も示さない。`supervised` の Agent のトークンも同じクレームを含み、それを許した承認は代わりに監査証跡で追跡できる（REQ-OAUTH2-050）。")
  agent_id?: string;
}


@doc("OpenID Connect の ID トークンのクレーム。TTL は 3600 秒である。")
model IdTokenClaims {
  iss: url;
  sub: string;
  aud: Audience;
  exp: int64;
  iat: int64;
  auth_time: int64;
  @doc("OIDC のセッションの id（OpenID Connect Front-Channel Logout 1.0 と Back-Channel Logout 1.0）。ブラウザーのセッションに結び付く発行のときだけ含む。RP はこれを `id_token_hint` として `/end_session` へ送り返し、フロントチャネルとバックチャネルのログアウトを関連付ける。")
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
  @doc("スコープから決まる属性のクレーム（ClaimsForScopes）と、クライアントごとの ClaimMapping のクレームのポリシーを合わせた追加のクレーム。キーが標準のクレームと衝突するときは、標準のクレームを優先する。")
  additional_claims?: Record<unknown>;
}


@doc("文字列、または一つ以上の文字列の配列（JWT、RFC 7519 の 4.1.3 節）。")
model Audience {
  @doc("JSON の文字列、または一つ以上の JSON の文字列の配列。")
  one_or_many: string | string[];
}


@doc("RFC 8414 と OpenID Connect Discovery 1.0 のメタデータ。")
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
  @doc("CIBA Core の 4 節：バックチャネル認証のエンドポイント。CIBA を提供するときだけ示す。")
  backchannel_authentication_endpoint?: url;
  @doc("CIBA Core の 4 節：実装したトークンの配送のモード。この実装は poll だけを提供し、ping も push も示さない。")
  backchannel_token_delivery_modes_supported?: string[] = #["poll"];
  @doc("CIBA Core の 4 節：`user_code` のパラメーターには対応しない。承認の画面はユーザー自身の認証済みのセッションとステップアップで守るので、二つ目の共有のシークレットは含めない。")
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
  @doc("client_id を、HTTPS の URL である Client ID Metadata Document として、リクエストのたびに解決してもよいことを示す。")
  client_id_metadata_document_supported: boolean = true;
  subject_types_supported: string[] = #["public"];
  claims_supported?: string[] = #["sub", "iss", "aud", "exp", "iat", "auth_time", "nonce", "acr", "amr", "azp", "name", "given_name", "family_name", "preferred_username", "email", "email_verified", "updated_at"];
  acr_values_supported?: string[] = #["urn:idmagic:acr:pwd", "urn:idmagic:acr:mfa"];
  ui_locales_supported?: string[] = #["en", "ja"];
  @doc("RFC 9396：このテナントが受け付ける `authorization_details` の種類。有効な AuthorizationDetailType から導く。")
  authorization_details_types_supported?: string[];
}


@doc("`/register` への動的クライアント登録のリクエストのボディ（RFC 7591）。結果の OAuth2Client はここから導く。含むのは登録するクライアントが選べるメタデータだけである。サーバーは PS256 で発行しクライアントの希望を受け取らないので `id_token_signed_response_alg` は含まず、`require_pkce` はクライアントの設定ではなくサーバーのポリシーである。どちらかをここで宣言すると、登録にない選択肢を示すことになる。")
model ClientRegistrationRequest {
  redirect_uris: url[];
  client_name?: string;
  client_type?: ClientType;
  grant_types: GrantType[];
  response_types?: ResponseType[];
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  scope?: string;
  jwks_uri?: url;
  jwks?: JWKS;
  tls_client_auth_subject_dn?: string;
  fapi_profile?: FapiProfile;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
  backchannel_logout_uri?: url;
  backchannel_logout_session_required?: boolean;
  frontchannel_logout_uri?: url;
  frontchannel_logout_session_required?: boolean;
}


@doc("クライアントが自らホストする HTTPS の URL から取得する Client ID Metadata Document。client_id はその取得の URL で、文書の中の client_id と完全に一致しなければならない。`token_endpoint_auth_method` がないか、`none` 以外の文書は拒否する。")
model ClientIDMetadataDocument {
  @doc("取得元の URL と完全に一致しなければならない。")
  client_id: url;
  client_name: string;
  client_uri?: url;
  logo_uri?: url;
  redirect_uris: url[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  @doc("ないときは `none` と読む。それ以外の値を宣言した文書は拒否する。")
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  @doc("ないときは `openid` だけを許す。")
  scope?: string;
}


@doc("管理者が `authorization_details` の種類（RFC 9396）を登録または更新する JSON のリクエスト。`type` は作成時にだけ設定できる。")
model AuthorizationDetailTypeRequest {
  @doc("作成する種類の URN。更新では含まない。更新ではパスが種類を指定し、一つのリクエストに同じ名前を二度宣言しないように、UpdateAuthorizationDetailTypeRequest はこれを省く。")
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


@doc("更新のボディ。`type` を含まないことを除き、AuthorizationDetailTypeRequest と同じである。どの種類を更新するかはパスがすでに指定しており、ハンドラーはそこから読む。")
model UpdateAuthorizationDetailTypeRequest {
  @maxLength(500)
  description?: string;
  schema: AuthorizationDetailsSchema;
  @minLength(1)
  @maxLength(500)
  display_template: string;
  state?: AuthorizationDetailTypeState;
}


@doc("`authorization_details` の種類の表現。管理 UI の一覧と詳細の表示は同じ表現を返す。")
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


@doc("管理 API の `authorization_details` の種類の一覧。")
model AuthorizationDetailTypeListResponse {
  types: AuthorizationDetailTypeResponse[];
}


@doc("管理者が、自身のテナントに OAuth のクライアントを作成する。生成した `client_secret` は作成のレスポンスで一度だけ返す。")
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
  jwks?: JWKS;
  tls_client_auth_subject_dn?: string;
  fapi_profile?: FapiProfile;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
  backchannel_logout_uri?: url;
  backchannel_logout_session_required?: boolean;
  frontchannel_logout_uri?: url;
  frontchannel_logout_session_required?: boolean;
}


@doc("管理者がクライアントのメタデータを更新する。client_id、認証の方式、シークレットは変えない。シークレットのローテーションには専用のエンドポイントがある。")
model AdminOAuth2ClientUpdateRequest {
  @maxLength(200)
  client_name?: string;
  redirect_uris?: url[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  scope?: string;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
  backchannel_logout_uri?: url;
  backchannel_logout_session_required?: boolean;
  frontchannel_logout_uri?: url;
  frontchannel_logout_session_required?: boolean;
}


@doc("管理 API でのクライアントの表現。`client_secret` と `client_secret_hash` は意図して契約に含めない。")
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
  jwks?: JWKS;
  tls_client_auth_subject_dn?: string;
  id_token_signed_response_alg: SignatureAlgorithm;
  require_pushed_authorization_requests: boolean;
  dpop_bound_access_tokens: boolean;
  fapi_profile: FapiProfile;
  backchannel_logout_uri?: url;
  backchannel_logout_session_required: boolean;
  frontchannel_logout_uri?: url;
  frontchannel_logout_session_required: boolean;
  created_at: utcDateTime;
}


@doc("クライアントの作成のレスポンス。`client_secret` は、生成したときだけ一度返す。")
model AdminOAuth2ClientCreateResponse {
  client: AdminOAuth2ClientResponse;
  client_secret?: string;
}


@doc("テナントのクライアント。")
model AdminOAuth2ClientListResponse {
  clients: AdminOAuth2ClientResponse[];
}


@doc("Application のエディターが示す、シークレットの資格情報の機微でないメタデータ。状態は `revoked_at`、`expires_at`、評価の時刻の順に導く。")
model ClientSecretCredentialMetadata {
  id: string;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
  status: ClientSecretCredentialStatus;
}


@doc("既存の資格情報を変えずに、有効期限付きの追加のクライアントシークレットを発行するリクエスト。`expires_in_days` は省くと 90 で、受け付けるのは 1 から 730 日だけである。")
model IssueClientSecretRequest {
  expires_in_days?: int64;
}


@doc("追加で発行したシークレットを一度だけ返すレスポンス。既存のシークレットもハッシュも、平文を再び読む方法も含まない。")
model IssueClientSecretResponse {
  client_secret: string;
  credential: ClientSecretCredentialMetadata;
  credentials: ClientSecretCredentialMetadata[];
}


@doc("資格情報を一つ失効させた後の、機微でない資格情報のメタデータ。シークレットもハッシュも含まない。")
model RevokeClientSecretResponse {
  credentials: ClientSecretCredentialMetadata[];
}


@doc("既存のシークレットを何日重ねるかを指定するローテーションのリクエスト。0 は即座に失効させ、1 から 30 は猶予を与え、省くと 7 日である。")
model RotateClientSecretRequest {
  grace_days?: int64;
}


@doc("新しいシークレットを一度だけ返すローテーションのレスポンス。後から読む API はない。")
model RotateClientSecretResponse {
  client_secret: string;
  grace_until?: utcDateTime;
  credentials: ClientSecretCredentialMetadata[];
}


@doc("管理 API でのテナントの Consent の表現。client_id と user_id は UUID なので、解決した人が読める名前（`client_name` と `preferred_username`）を表示のために並べて返す。")
model AdminConsentResponse {
  user_id: string;
  @doc("user_id から解決した表示名。該当する User が見つからないときは省く。")
  preferred_username?: string;
  client_id: string;
  @doc("client_id から解決した表示名。`OAuth2Client.client_name`、Application のカタログの名前の順に解決し、どちらもなければ client_id そのものを使う。")
  client_name: string;
  scopes: string[];
  state: ConsentState;
  granted_at: utcDateTime;
  expires_at: utcDateTime;
  revoked_at?: utcDateTime;
}


@doc("テナントの Consent。")
model AdminConsentListResponse {
  consents: AdminConsentResponse[];
}


@doc("エンドユーザー自身の接続済みのアプリケーション、すなわち有効な Consent（セルフサービス）。")
model AccountConsentListResponse {
  consents: AccountConsentResponse[];
}


@doc("`application/x-www-form-urlencoded` で送るトークンエンドポイントのリクエスト。")
model TokenRequest {
  grant_type: GrantType;
  code?: string;
  redirect_uri?: string;
  code_verifier?: string;
  refresh_token?: string;
  device_code?: string;
  @doc("CIBA のグラント（`urn:openid:params:grant-type:ciba`）の `auth_req_id`。承認が確定するまでは `authorization_pending` または `slow_down` を返し、確定した後はちょうど一度だけトークンと交換する。")
  auth_req_id?: string;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
  scope?: string;
  @doc("RFC 8693 のトークン交換のグラントの `subject_token`。自身が発行したアクセストークン（`urn:ietf:params:oauth:token-type:access_token`）か、登録した WorkloadTrustBundle の発行者が発行した外部のアテステーションの JWT（`urn:ietf:params:oauth:token-type:jwt`）のどちらかである。")
  subject_token?: string;
  @doc("`subject_token` の種類を示す URN。省くと自身が発行したアクセストークンとして扱う。")
  subject_token_type?: string;
  @doc("RFC 8693 のトークン交換のグラントの任意の `actor_token`。自身が発行したアクセストークンでなければならない。")
  actor_token?: string;
  actor_token_type?: string;
  requested_token_type?: string;
  @doc("RFC 9396：トークン交換で絞り込みを求める、構造化された詳細の JSON の文字列。任意である。")
  authorization_details?: string;
  @doc("RFC 8707：authorization_code の引き換え、client_credentials、device_code、トークン交換のためのリソースインジケーター。パラメーターを複数回指定すると `invalid_target` になる。refresh_token のグラントは明示的に指定せず、最初の発行で結び付けたリソースをローテーションを通じて自動で引き継ぐ。")
  resource?: ResourceIndicator;
}


@doc("成功した `/token` のレスポンスのボディ（RFC 6749 の 5.1 節）。発行したトークンとその種類を返す。")
model TokenResponse {
  access_token: string;
  @doc("\"Bearer\" または \"DPoP\"")
  token_type: string;
  expires_in?: int64;
  refresh_token?: string;
  id_token?: string;
  scope?: string;
}


@doc("リソースサーバーが、トークンが有効かどうかを認可サーバーに問い合わせる `/introspect` のリクエスト（RFC 7662 の 2.1 節）。")
model IntrospectionRequest {
  token: string;
  token_type_hint?: TokenType;
}


@doc("トークンのサブジェクトが現在どう動いているか。この導出した値は `act` の連鎖と主体の種類から決まり、独立した永続的な状態はない。`OnBehalfOf` は別の主体がサブジェクトのために動いていること（`act` がサブジェクトと異なる操作者を含む）を意味する。`Autonomous` は委任がなく、サブジェクトが人でない主体（Agent またはクライアント）であることを意味する。`Direct` は委任がなく、サブジェクトが人のユーザーであることを意味する。")
enum DelegationMode {
  Direct: "direct",
  Autonomous: "autonomous",
  OnBehalfOf: "on_behalf_of",
}


@doc("`/introspect` のレスポンス（RFC 7662 の 2.2 節）。トークンが有効かどうかと、有効なときはそのメタデータを返す。")
model IntrospectionResponse {
  active: boolean;
  scope?: string;
  client_id?: string;
  sub?: string;
  aud?: Audience;
  exp?: int64;
  iat?: int64;
  @doc("RFC 7662 の 2.2 節が求める、イントロスペクションしたアクセストークンの提示の方法（RFC 6749 の 5.1 節の種類）。`DPoP` は RFC 9449 の 5 節の送信者の制約付きの提示で、証明書に結び付いたトークンは RFC 8705 の 3 節に従って `Bearer` のままである。有効でないトークンと、固有の提示の種類のないリフレッシュトークンでは含まない。この語彙は、リクエストがどのトークンについてのものかを示す `TokenType` とは別である。")
  token_type?: "Bearer" | "DPoP";
  jti?: string;
  cnf?: SenderConstraintCnf;
  @doc("RFC 8693 の 4.1 節の `act` のクレーム。委任したトークンでは、現在の操作者と操作者の連鎖を返す。")
  act?: Record<unknown>;
  @doc("有効なトークンが、自律的な実行、代理の実行、直接の実行のどれを表すか。リソースサーバーが `act` と主体の種類の規則を実装し直さずに済むように、認可サーバーが導出した値を返す。値は、監査イベントと同じ導出の関数が生成する。")
  delegation_mode?: DelegationMode;
}


@doc("指定したトークンを失効させる `/revoke` のリクエスト（RFC 7009 の 2.1 節）。")
model RevokeRequest {
  token: string;
  token_type_hint?: TokenType;
}


@doc("authorize のパラメーターをフォームとして受け取る。")
model ParRequest {
  parameters: Record<string>;
}


@doc("`/par` のレスポンス（RFC 9126 の 2.2 節）。後の `/authorize` が参照する短命の `request_uri` を発行する。")
model ParResponse {
  request_uri: string;
  expires_in: int64;
}


@doc("入力の制約されたデバイスからフローを開始する `/device_authorization` のリクエスト（RFC 8628 の 3.1 節）。")
model DeviceAuthorizationRequest {
  scope?: string;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
}


@doc("`/device_authorization` のレスポンス（RFC 8628 の 3.2 節）。`device_code`、`user_code`、`verification_uri` を返し、デバイスは問い合わせへ進む。")
model DeviceAuthorizationResponse {
  device_code: string;
  user_code: string;
  verification_uri: url;
  verification_uri_complete?: url;
  expires_in: int64;
  interval?: int64;
}


@doc("クライアントが人の承認のリクエストを起こす `/bc-authorize` のリクエスト（CIBA Core の 7.1 節）。`login_hint` と `id_token_hint` のちょうど一つを与え、どちらも解決しないときはフェイルクローズで `unknown_user_id` で拒否する。`user_code` には対応せず、ディスカバリーでも示さない。")
model BackchannelAuthenticationRequest {
  @doc("問い合わせる User を指すヒント。ユーザー名またはメールアドレスとして解決する。")
  login_hint?: string;
  @doc("この OP が発行した有効な ID トークン。問い合わせる User を解決するために、その署名、`iss`、`aud`、`sub`、`exp` を検証する。")
  id_token_hint?: string;
  @doc("承認されたときに発行するトークンのスコープ。必須で、`openid` を含み、クライアントが宣言したスコープの部分集合でなければならない。")
  scope: string;
  @doc("RFC 9396：承認の対象の構造化された権限のリクエスト。JSON の文字列で表す。登録した種類だけを受け付ける。")
  authorization_details?: string;
  @doc("承認の画面に示す短い識別のテキスト（CIBA Core の 7.1 節）。")
  @maxLength(64)
  binding_message?: string;
  @doc("クライアントが求める有効期間の秒数。600 以下の正の整数で、省くと 300 である。")
  requested_expiry?: int64;
  client_id?: string;
  client_secret?: string;
  client_assertion?: string;
  client_assertion_type?: string;
}


@doc("`/bc-authorize` のレスポンス（CIBA Core の 7.3 節）。`auth_req_id` は bearer のシークレットで、クライアントは問い合わせの間、CIBA のグラントでトークンエンドポイントへ提示する。")
model BackchannelAuthenticationResponse {
  auth_req_id: string;
  expires_in: int64;
  @doc("クライアントが守らなければならない問い合わせの最小の間隔（秒）。")
  interval?: int64;
}


@doc("エンドユーザーのアカウントポータルが返す、保留中の承認のリクエストの一つ（セルフサービス）。呼び出し元宛てのリクエストに固定されるので、`user_id` は現れない。`binding_message` を内容の代わりにはせず、誰が何を求めたかを並べて返す。")
model AccountApprovalRequest {
  id: string;
  client_id: string;
  @doc("client_id から解決した表示名。`OAuth2Client.client_name`、Application のカタログの名前の順に解決し、どちらもなければ client_id そのものを使う。")
  client_name: string;
  @doc("クライアントに結び付いた Agent があれば、その表示名。")
  agent_name?: string;
  scopes: string[];
  authorization_details?: AuthorizationDetail[];
  binding_message?: string;
  requested_at: utcDateTime;
  expires_at: utcDateTime;
}


@doc("エンドユーザー自身宛ての保留中の承認のリクエスト（セルフサービス）。")
model AccountApprovalRequestListResponse {
  approval_requests: AccountApprovalRequest[];
}


@doc("一つの承認のリクエストに対するエンドユーザーの判断。承認または拒否である。")
model AccountApprovalDecisionRequest {
  @doc("`approve` または `deny` のどちらか。")
  decision: string;
}


@doc("`GET /authorize` または PAR の認可のパラメーター。どのフィールドも単一の値で、パラメーターを複数回指定すると `invalid_request` として拒否する。`prompt` は一意なトークン `login`、`consent`、`none` を空白で区切った集合で、`none` をほかと組み合わせることはない。")
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
  @doc("RFC 8707 のリソースインジケーター。登録済みで有効な McpResourceServer のどれにも一致しないときは `invalid_target` で拒否し、パラメーターを複数回指定したときも同じである。")
  resource?: ResourceIndicator;
}


@doc("SPA が `/api/auth/consent` へ POST する同意の判断。")
model BrowserConsentRequest {
  @doc("`allow` または `deny` のどちらか。")
  action: string;
}


@doc("SPA が `/api/auth/device` へ POST する、デバイスの `user_code` に対する判断。")
model BrowserDeviceRequest {
  user_code: string;
  @doc("ユーザーがどちらで答えたか。`approve` または `deny` である。エンドポイントはその答えを記録するためにあるので、これのないボディは何も決めない。")
  action: string;
}


@doc("SPA が現在のブラウザーの認可のトランザクションを描画するのに使う、no-store の JSON。")
model BrowserTransactionResponse {
  @doc("描画する画面。login、totp、consent、device で、要素が加わればさらに増える。")
  kind: string;
  csrf_token: string;
  @doc("同意の画面に示すクライアントの読める名前。`OAuth2Client.client_name`、Application のカタログの名前、client_id の順に解決する。client_id は UUID なので、そのままの形では示さないようにする。")
  client_name?: string;
  scopes?: string[];
}


@doc("SPA がデバイスの `user_code` の画面を描画するのに使う、no-store の JSON。")
model BrowserDeviceContextResponse {
  kind: string;
  user_code?: string;
  csrf_token: string;
}


@doc("RP-Initiated Logout v1 のクエリまたはフォームのパラメーター。`post_logout_redirect_uri` は client_id に照らして検証する。`id_token_hint` はこの OP が発行した ID トークンのときだけ受け付け、署名、`iss`、`aud`（client_id と矛盾するときは拒否する）、`sub`、`sid` をすべて検証する。ログアウトの時点ですでに期限切れの ID トークンを送るのは通常の RP の実装の振る舞いなので、`exp` は検証しない。")
model EndSessionParameters {
  client_id?: string;
  id_token_hint?: string;
  post_logout_redirect_uri?: url;
  state?: string;
}


@doc("FrontChannelLogout の結果の一つの項目。EndSession のレスポンスが埋め込む一つの iframe に対応する。")
model FrontChannelLogoutTarget {
  client_id: string;
  @doc("登録した `frontchannel_logout_uri`。`frontchannel_logout_session_required` を設定しているときは `iss` と `sid` を加える。")
  iframe_uri: url;
}


@doc("OpenID Connect Core の 5.3 節。変わりうる OIDC のクレームを含む。")
model UserInfoResponse {
  sub: string;
  @doc("スコープから決まる属性のクレーム（ClaimsForScopes）と、クライアントごとの ClaimMapping のクレームのポリシーを合わせた追加のクレーム。キーが標準のクレームと衝突するときは、標準のクレームを優先する。")
  claims?: Record<unknown>;
}


@doc("リクエストに必須のパラメーターがない、パラメーターが重複している、または不正な値を含む（RFC 6749 の 5.2 節、invalid_request）。OAuth 2.0 と OpenID Connect のエンドポイントは、この条件を OAuthError のボディで報告する。汎用の API は同じ条件を、RFC 9457 の Problem Details のボディである InvalidRequestError として報告する。")
model OAuthInvalidRequestError is OAuthError;


@doc("リソースオーナーまたは認可サーバーがリクエストを拒否した（RFC 6749 の 5.2 節、access_denied）。RFC 6749 の 5.2 節はトークンエンドポイントのエラーのステータスを 400 と定め、401 は invalid_client にだけ許すので、ユーザーが CIBA やデバイスフローで示した拒否は、`/token` には 403 ではなく 400 として届く。汎用の API は同じ条件を RFC 9457 の Problem Details のボディである AccessDeniedError として報告し、`/authorize` がリダイレクトせずにブラウザーへ書く唯一の拒否も同じである。")
model OAuthAccessDeniedError is OAuthError;


@doc("OAuth 2.0 / OpenID Connect の共有のエラーのレスポンスのボディ（RFC 6749 の 5.2 節）。`support_http.OAuthErrorBody` が `Content-Type: application/json` で書く。エラーの形を標準が定めるすべてのエンドポイントは、既製のクライアントが読めるように、RFC 9457 の Problem Details ではなくこのボディを返す。`is OAuthError` として宣言した名前付きの `*Error` モデルはこのボディを共有し、`error` の値で区別する。")
model OAuthError {
  @doc("プロトコルのエラーコード。どの条件が起きたかを識別する。")
  error: OAuthErrorCode;
  @doc("発生した事象の、人が読む説明。常に英語で、ローカライズしない。サーバーが安全に開示できるものがないときは値を空にし、メンバー自体はどちらの場合も含む。")
  error_description: string;
}


@doc("クライアントの認証に失敗したか、クライアントが登録されていない（RFC 6749 の 5.2 節、invalid_client）。詳細は返さないので、レスポンスは `client_id` が存在するかを明かさない。")
model InvalidClientError is OAuthError;


@doc("認可コード、リフレッシュトークン、デバイスコード、またはほかのグラントが、不正、期限切れ、再送されたもの、または `redirect_uri` や PKCE の検証子と一致しない（RFC 6749 の 5.2 節、invalid_grant）。")
model InvalidGrantError is OAuthError;


@doc("クライアントはこの `grant_type` を使うことを許されていない（RFC 6749 の 5.2 節、unauthorized_client）。")
model UnauthorizedClientError is OAuthError;


@doc("認可サーバーはこの `grant_type` に対応しない（RFC 6749 の 5.2 節、unsupported_grant_type）。")
model UnsupportedGrantTypeError is OAuthError;


@doc("求めたスコープの形式が誤っている、未知である、またはクライアントに許されていない（RFC 6749 の 5.2 節、invalid_scope）。")
model InvalidScopeError is OAuthError;


@doc("`resource` パラメーターが、未登録、無効、または複数回指定されている（RFC 8707 の 2 節、invalid_target）。フェイルクローズで拒否する。")
model InvalidTargetError is OAuthError;


@doc("アクセストークンが、期限切れ、改ざん、形式の誤り、または失効済みである（RFC 6750 の 3.1 節、invalid_token）。")
model InvalidTokenError is OAuthError;


@doc("DPoP の証明の JWT の検証が、署名、`jti`、`htu`/`htm`、nonce、時計のずれ、またはそれに類する検査で失敗した（RFC 9449 の 7.1 節、invalid_dpop_proof）。")
model InvalidDpopProofError is OAuthError;


@doc("提示したトークンのスコープが、求めた OAuth 2.0 または OpenID Connect の操作を満たさない（RFC 6750 の 3.1 節、insufficient_scope）。汎用の API は同じ条件を、RFC 9457 の Problem Details のボディである InsufficientScopeError として報告する。")
model OAuthInsufficientScopeError is OAuthError;


@doc("AuthorizationDetailType のスキーマの定義が不正である。検証に失敗したか、未知の Semantics の値を参照している。トークンエンドポイントではなく管理 API が返す。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:invalid_type` である。")
model InvalidAuthorizationDetailTypeError is ProblemDetails;


@doc("Device Authorization Grant で、`device_code` が期限を過ぎた（RFC 8628 の 3.5 節、expired_token）。")
model ExpiredTokenError is OAuthError;


@doc("クライアントが Device Authorization Grant を速すぎる間隔で問い合わせており、間隔に 5 秒を加えなければならない（RFC 8628 の 3.5 節、slow_down）。")
model SlowDownError is OAuthError;


@doc("Device Authorization Grant で、ユーザーがまだ承認していない（RFC 8628 の 3.5 節、authorization_pending）。クライアントは `interval` 秒待って再試行する。")
model AuthorizationPendingError is OAuthError;


@doc("CIBA の /bc-authorize のエンドポイントで、`login_hint` も `id_token_hint` も、承認を求める User へ解決しない（OpenID Connect CIBA Core の 13 節、unknown_user_id）。有効でないユーザーと他のテナントのユーザーも同じエラーにまとめ、違いから User の存在が分からないようにする。")
model UnknownUserIdError is OAuthError;


@doc("CIBA の /bc-authorize のエンドポイントで、`binding_message` が長すぎるか、表示に適さない（OpenID Connect CIBA Core の 13 節、invalid_binding_message）。")
model InvalidBindingMessageError is OAuthError;


@doc("認可サーバーの予期しない内部の失敗（RFC 6749 の 4.1.2.1 節、server_error）。")
model ServerError is OAuthError;


@doc("一つの経路の固定ウィンドウのレート制限の設定。運用者は閾値とウィンドウを環境変数で調整し、このモデルは形だけを定め、仕様に具体的な値を埋め込まない。")
model EndpointRateLimitPolicy {
  @doc("対象の経路の識別子。`token`、`authorize`、`par`、`device_authorization`、`password_reset`、`login` など。")
  policy_id: string;
  max_requests: int64;
  window_seconds: int64;
  @doc("共有のカウンターのストアに到達できないときの動作。`fail_closed` に固定する。")
  fail_mode: string;
}


@doc("`/register`（RFC 7591）で新しいクライアントを動的に登録した。クライアントの登録の監査証跡の起点である。")
model ClientRegistered {
  occurredAt: utcDateTime;
  @doc("クライアントが属するテナント。")
  tenantId: string;
  clientId: string;
  clientType?: ClientType;
}


@doc("URL の形の client_id について、Client ID Metadata Document を取得して検証し、認可リクエストが先へ進んだ。永続的には何も登録しない。")
model ClientIdMetadataDocumentResolved {
  occurredAt: utcDateTime;
  @doc("認可リクエストが属するテナント。")
  tenantId: string;
  @doc("取得元の URL。")
  clientId: string;
}


@doc("URL の形の client_id について、Client ID Metadata Document の取得または検証に失敗し、リクエストをフェイルクローズで拒否した。")
model ClientIdMetadataDocumentRejected {
  occurredAt: utcDateTime;
  @doc("認可リクエストが属するテナント。")
  tenantId: string;
  @doc("取得元の URL。")
  clientId: string;
  @doc("拒否の分類。取得の失敗、形式の誤った JSON、client_id の不一致、`redirect_uris` の欠如、対応しない `token_endpoint_auth_method` などである。")
  reason: string;
}


@doc("管理者が、自身のテナントに OAuth のクライアントを作成した。シークレットは含まない。")
model AdminOAuth2ClientCreated {
  occurredAt: utcDateTime;
  @doc("クライアントが属するテナント。操作した主体と一致する。")
  tenantId: string;
  actorUserId: string;
  clientId: string;
}


@doc("管理者が OAuth のクライアントのメタデータを更新した。")
model AdminOAuth2ClientUpdated {
  occurredAt: utcDateTime;
  @doc("クライアントが属するテナント。操作した主体と一致する。")
  tenantId: string;
  actorUserId: string;
  clientId: string;
  changedFields: string[];
}


@doc("管理者が、自身のテナントから OAuth のクライアントを削除した。")
model AdminOAuth2ClientDeleted {
  occurredAt: utcDateTime;
  @doc("クライアントが属するテナント。操作した主体と一致する。")
  tenantId: string;
  actorUserId: string;
  clientId: string;
}


@doc("管理者が互換のインターフェースで、シークレット方式のクライアントの資格情報をローテーションしたか、即座に失効させた。ペイロードにはシークレットもハッシュも含まない。")
model ClientSecretRotated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  graceUntil?: utcDateTime;
  revokedImmediately: boolean;
}


@doc("ResourceOwner が、クライアントに対するスコープに同意した。GDPR 第 6 条の法的根拠を得たことの記録である。")
model ConsentGranted {
  occurredAt: utcDateTime;
  @doc("同意が行われたテナント。")
  tenantId: string;
  userId: string;
  clientId: string;
  scopes: string[];
}


@doc("ResourceOwner またはテナントの Administrator が、GDPR 第 7 条 3 項の撤回の権利を行使して、既存の同意を撤回した。以降の発行を止める。")
model ConsentRevoked {
  occurredAt: utcDateTime;
  @doc("同意が属するテナント。")
  tenantId: string;
  @doc("撤回が管理 API を通じて行われたときの、Administrator の `sub`。ResourceOwner が自ら撤回したときは省いてよい。")
  actorUserId?: string;
  userId: string;
  clientId: string;
}


@doc("RFC 9396：クライアントが `/authorize`、`/par`、`/token` で `authorization_details` を求め、リクエストが登録した種類に照らした検証を通った。")
model AuthorizationDetailsRequested {
  occurredAt: utcDateTime;
  tenantId: string;
  clientId: string;
  @doc("リクエストの時点で主体がすでに確定していたときの、その主体の `sub`。")
  userId?: string;
  @doc("求めた詳細の種類。")
  detailTypes: string[];
}


@doc("RFC 9396：ResourceOwner が、示された `authorization_details` に同意した。発行と交換のときの部分集合の検査の基準になる。")
model AuthorizationDetailsConsented {
  occurredAt: utcDateTime;
  tenantId: string;
  userId: string;
  clientId: string;
  detailTypes: string[];
}


@doc("RFC 9396：`authorization_details` の形式が誤っている、未登録の種類を指定している、スキーマに合わない、同意を超えている、または絞り込みを超えているので、リクエストをフェイルクローズで拒否した。")
model AuthorizationDetailsRejected {
  occurredAt: utcDateTime;
  tenantId: string;
  clientId: string;
  @doc("拒否した理由。たとえば `invalid_authorization_details`。")
  reason: string;
}


@doc("`/authorize` の完了とともに AuthorizationCode を発行した。60 秒以内に `/token` で交換しなければならない。")
model AuthorizationCodeIssued {
  occurredAt: utcDateTime;
  @doc("認可コードを発行したテナント。")
  tenantId: string;
  clientId: string;
  userId: string;
  scopes: string[];
  codeChallengeMethod?: string;
}


@doc("AuthorizationCode を `/token` で正しく交換した。同じコードを再び使うと、ファミリーを失効させる。")
model AuthorizationCodeRedeemed {
  occurredAt: utcDateTime;
  @doc("交換が行われたテナント。")
  tenantId: string;
  clientId: string;
  userId: string;
}


@doc("アクセストークン（JWT）を発行した。`senderConstraint` があるときは、DPoP または mTLS による所持の証明に結び付く。")
model AccessTokenIssued {
  occurredAt: utcDateTime;
  @doc("トークンを発行したテナント。")
  tenantId: string;
  jti: string;
  clientId: string;
  userId: string;
  scopes: string[];
  senderConstraint: AccessTokenSenderConstraint = AccessTokenSenderConstraint.None;
}


@doc("リフレッシュトークンを発行した。`familyId` は元の認可コードから派生したファミリーを識別し、ローテーションを通じて引き継ぐ。")
model RefreshTokenIssued {
  occurredAt: utcDateTime;
  @doc("トークンを発行したテナント。")
  tenantId: string;
  tokenId: string;
  familyId: string;
  parentId?: string;
  clientId: string;
  userId: string;
}


@doc("リフレッシュトークンを `/token` で交換した。新しい子のトークンを発行し、親は `rotated` へ移った。")
model RefreshTokenRotated {
  occurredAt: utcDateTime;
  @doc("ローテーションが行われたテナント。")
  tenantId: string;
  oldTokenId: string;
  newTokenId: string;
  familyId: string;
}


@doc("`/revoke` または管理操作によって、トークンを失効させた（RFC 7009）。監査と即時性の要件をともなう。")
model TokenRevoked {
  occurredAt: utcDateTime;
  @doc("失効が行われたテナント。")
  tenantId: string;
  tokenType: TokenType;
  tokenId: string;
  reason?: string;
}


@doc("`/introspect` を呼び出し、トークンの状態を返した（RFC 7662）。リソースサーバーの振る舞いの分析に使う。")
model TokenIntrospected {
  occurredAt: utcDateTime;
  @doc("イントロスペクションが行われたテナント。")
  tenantId: string;
  rsClientId: string;
  tokenId: string;
  active: boolean;
}


@doc("RFC 8693：トークン交換で、委任したトークンを発行した。`delegationDepth` は、発行したトークンの `act` のクレームの入れ子の深さである。")
model TokenExchanged {
  occurredAt: utcDateTime;
  @doc("トークン交換が行われたテナント。")
  tenantId: string;
  @doc("発行したトークンの現在の操作者（`act.sub`）。")
  actorUserId: string;
  @doc("発行したトークンのサブジェクト（`sub`）。`subject_token` の `sub` を保つ。")
  subjectUserId: string;
  @doc("発行したトークンの `aud`。`resource` パラメーターから得る。")
  audience: string;
  @doc("発行したトークンの `act` のクレームの入れ子の深さ。")
  delegationDepth: int64;
  @doc("この判断に適用した委任の最大の深さ。テナントに上書きがないときはシステムのデフォルトを適用する。適用した上限を記録することで、判断を監査イベントから再現できる。")
  maxDelegationDepth: int64;
  @doc("発行したトークンの委任のモード。イントロスペクションと同じ導出の関数が生成する。")
  delegationMode: DelegationMode;
  @doc("交換するクライアントが Agent に結び付いているときの、その Agent。操作者の種類を推測ではなくイベントから読めるようにし、監査の `actor.type` と `agent.id` の軸が必要とする（REQ-AUDIT-005）。")
  agentId?: string;
  @doc("発行したトークンの `act` の連鎖のすべての主体。現在の操作者を先頭に、サブジェクトを最後に、それぞれ一度ずつ並べる。深さだけでは委任がどこまで及んだかは分かっても誰を経由したかは分からず、途中の操作者を見つけられない。このリストが、監査の `delegation.actor` の軸が検索するものである（REQ-AUDIT-006）。これらは Agent の id ではなくトークンのサブジェクトである。交換の時点で解決できるのは現在の操作者の Agent だけで、それだけを示すと、連鎖のどこに Agent がいたかを取り違える。Agent の単位の問いには、代わりに `agentId` が答える。ResolvableUserEventPayloadPolicy に従い、識別子だけを含め、ユーザー名は含めない。")
  actorChain: string[];
}


@doc("RFC 8693：ポリシー違反または検査の失敗により、トークン交換のリクエストをフェイルクローズで拒否した。")
model TokenExchangeRejected {
  occurredAt: utcDateTime;
  @doc("トークン交換を拒否したテナント。")
  tenantId: string;
  @doc("拒否の時点で分かっていたときの、現在の操作者。分かっていなかったときは空である。")
  actorUserId?: string;
  @doc("拒否した理由。たとえば OAuth のエラーコード。")
  reason: string;
}


@doc("RFC 9728：`/.well-known/oauth-protected-resource` が、登録済みの McpResourceServer のメタデータを返した。")
model ProtectedResourceMetadataServed {
  occurredAt: utcDateTime;
  @doc("メタデータを返した対象のテナント。")
  tenantId: string;
  @doc("メタデータを返した対象のリソースの URI。")
  resource: string;
}


@doc("RFC 8707：リソースインジケーターに従い、オーディエンスを一つの McpResourceServer に厳密に限ったアクセストークンを発行した。authorize、PAR、token（authorization_code）の経路を対象とする。トークン交換でリソースに限った発行は `TokenExchanged.audience` で表し、このイベントは発行しない。")
model ResourceScopedTokenIssued {
  occurredAt: utcDateTime;
  @doc("トークンを発行したテナント。")
  tenantId: string;
  clientId: string;
  @doc("発行したトークンの `aud` に結び付くリソースの URI。")
  resource: string;
  scopes: string[];
}


@doc("RFC 8707：リソースインジケーターが未登録、無効、または複数回指定されていたので、リクエストをフェイルクローズで拒否した。authorize、PAR、token（authorization_code）の経路を対象とする。トークン交換での拒否は `TokenExchangeRejected.reason` で表し、このイベントは発行しない。")
model ResourceAudienceRejected {
  occurredAt: utcDateTime;
  @doc("拒否が行われたテナント。")
  tenantId: string;
  clientId: string;
  @doc("拒否した理由。たとえば `invalid_target`。")
  reason: string;
}


@doc("すでに `rotated` の状態のリフレッシュトークンが再び提示された。ファミリー全体の失効を引き起こす（RFC 9700 の 4.14 節）。")
model RefreshTokenReuseDetected {
  occurredAt: utcDateTime;
  @doc("再利用を検出したテナント。")
  tenantId: string;
  familyId: string;
  tokenId: string;
  clientId?: string;
}


@doc("`/par` に届いた authorize のパラメーターをサーバー側に保存し、短命の `request_uri` を発行した（RFC 9126）。")
model PARStored {
  occurredAt: utcDateTime;
  @doc("PAR を受け付けたテナント。")
  tenantId: string;
  requestUri: string;
  clientId: string;
}


@doc("RFC 8628 の 3.1 節：`device_code` と `user_code` を発行した。`user_code` は監査ログに届かない。")
model DeviceAuthorizationRequested {
  occurredAt: utcDateTime;
  @doc("デバイス認可を受け付けたテナント。")
  tenantId: string;
  clientId: string;
  scopes: string[];
}


@doc("RFC 8628 の 3.3 節：ユーザーが承認した。")
model DeviceAuthorizationApproved {
  occurredAt: utcDateTime;
  @doc("承認が行われたテナント。")
  tenantId: string;
  clientId: string;
  userId: string;
}


@doc("RFC 8628 の 3.3 節：ユーザーが拒否したか、誤った `user_code` を入力し続けた。")
model DeviceAuthorizationDenied {
  occurredAt: utcDateTime;
  @doc("拒否が行われたテナント。")
  tenantId: string;
  clientId: string;
  userId?: string;
}


@doc("Agent の種類が、この経路が記録しない人の承認を求めるので、発行をフェイルクローズで拒否した（REQ-OAUTH2-050）。`client_credentials`、トークン交換、ワークロード ID フェデレーションの交換を対象とする。承認をともなう経路は CIBA で、その結果は代わりに BackchannelAuthApproved で表す。読んだ種類を含めることで、Agent を再び解決せずに、監査イベントだけから判断を再現できる。")
model AgentApprovalRequired {
  occurredAt: utcDateTime;
  @doc("拒否が行われたテナント。")
  tenantId: string;
  agentId: string;
  @doc("Agent が結び付き、認証に使ったクライアント。")
  clientId: string;
  @doc("判断に使った種類。AgentKind の外の値も読んだまま記録する。それこそがフェイルクローズの規則が対象とする場合だからである。")
  kind: string;
  @doc("拒否したリクエストが求めたグラントの種類。たとえば `client_credentials`。")
  grantType: string;
}


@doc("CIBA Core の 7.3 節：人が確定する承認のリクエストを起こした。`auth_req_id` は bearer のシークレットで、監査ログに届かない。")
model BackchannelAuthRequested {
  occurredAt: utcDateTime;
  @doc("承認のリクエストを受け付けたテナント。")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  agentId?: string;
  userId: string;
  scopes: string[];
}


@doc("CIBA Core の 7.3 節：問い合わせを受けた User がリクエストを承認した。")
model BackchannelAuthApproved {
  occurredAt: utcDateTime;
  @doc("承認が行われたテナント。")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  @doc("求めたクライアントが Agent に結び付いているときの、承認の対象の Agent。`supervised` の Agent の発行を、それを許した人へたどれるようにする（REQ-OAUTH2-050）。")
  agentId?: string;
  userId: string;
}


@doc("CIBA Core の 7.3 節：問い合わせを受けた User がリクエストを拒否した。")
model BackchannelAuthDenied {
  occurredAt: utcDateTime;
  @doc("拒否が行われたテナント。")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  userId: string;
}


@doc("CIBA Core の 11 節：承認のリクエストが判断されないまま期限に達した。")
model BackchannelAuthExpired {
  occurredAt: utcDateTime;
  @doc("期限切れが起きたテナント。")
  tenantId: string;
  approvalRequestId: string;
  clientId: string;
  userId: string;
}


@doc("承認のリクエストを、承認済みのトークンへ一度だけ使うライフサイクルのイベント。")
model Consume {
  occurredAt: utcDateTime;
}


@doc("認可リクエストを検証するライフサイクルのイベント。")
model Validate {
  occurredAt: utcDateTime;
}


@doc("ライフサイクルを拒否の終端の状態へ移すイベント。")
model Reject {
  occurredAt: utcDateTime;
}


@doc("リソースオーナーの認証が成功したイベント。")
model AuthenticateUser {
  occurredAt: utcDateTime;
}


@doc("リソースオーナーに同意を求めるイベント。")
model RequestConsent {
  occurredAt: utcDateTime;
}


@doc("リソースオーナーが同意したイベント。")
model GrantConsent {
  occurredAt: utcDateTime;
}


@doc("認可コードを発行するライフサイクルのイベント。")
model IssueCode {
  occurredAt: utcDateTime;
}


@doc("認可コードを一度だけ引き換えるライフサイクルのイベント。")
model RedeemCode {
  occurredAt: utcDateTime;
}


@doc("期限に達したことを表すライフサイクルのイベント。")
model Expire {
  occurredAt: utcDateTime;
}


@doc("デバイスのユーザーコードが入力されたイベント。")
model EnterUserCode {
  occurredAt: utcDateTime;
}


@doc("デバイス認可が承認されたイベント。")
model Approve {
  occurredAt: utcDateTime;
}


@doc("デバイス認可が拒否されたイベント。")
model Deny {
  occurredAt: utcDateTime;
}


@doc("デバイスコードをトークンと交換したイベント。")
model Exchange {
  occurredAt: utcDateTime;
}


@doc("リフレッシュトークンを子のトークンへローテーションするイベント。")
model Rotate {
  occurredAt: utcDateTime;
}


@doc("トークンを失効させるライフサイクルのイベント。")
model RevokeToken {
  occurredAt: utcDateTime;
}


@doc("同意を論理的に撤回するライフサイクルのイベント。")
model RevokeConsent {
  occurredAt: utcDateTime;
}


@doc("PAR の `request_uri` を一度だけ使うライフサイクルのイベント。")
model Use {
  occurredAt: utcDateTime;
}


@doc("LogoutNotification の配送に成功したライフサイクルのイベント。")
model Deliver {
  occurredAt: utcDateTime;
}


@doc("LogoutNotification が `max_attempts` に達し、デッドレターとして確定したライフサイクルのイベント。")
model Exhaust {
  occurredAt: utcDateTime;
}


@doc("ListAdminRolePolicies が報告する、一つの管理用のロールとそれが含む権限。一覧はコンパイルした契約から導くので、サーバーが実際に公開する操作と常に一致する。")
model AdminRolePolicy {
  @doc("ロールの名前。User の `roles` に現れる値である。")
  name: string;
  @doc("ロールの用途。常に英語で、ローカライズしない。")
  description: string;
  @doc("同じロールへ解決する別名。")
  aliases: string[];
  permissions: AdminRolePermission[];
}


@doc("ロールが含む一つの権限と、その権限が開く HTTP のインターフェース。")
model AdminRolePermission {
  @doc("権限の名前。")
  name: string;
  @doc("権限が許す操作。`read` や `write` など。")
  action: string;
  @doc("権限が許すことの説明。常に英語で、ローカライズしない。")
  description: string;
  interfaces: AdminRoleInterface[];
}


@doc("権限が開く一つの HTTP のインターフェース。コンパイルした契約と同じ方法で識別する。")
model AdminRoleInterface {
  @doc("`@TypeSpec.OpenAPI.operationId` で宣言した操作の id。")
  name: string;
  @doc("HTTP のメソッド。")
  method: string;
  @doc("経路のテンプレート。")
  path: string;
}


@doc("Dynamic Client Registration のクライアント情報のレスポンス（RFC 7591 の 3.2.1 節）。登録したメタデータをサーバーが保存したとおりに繰り返し、confidential のクライアントでは登録時にちょうど一度だけ `client_secret` を含む。")
model ClientRegistrationResponse {
  client_id: string;
  @doc("confidential のクライアントで、このレスポンスでだけ返す。後から読み直すことはできない。")
  client_secret?: string;
  client_type: ClientType;
  redirect_uris: url[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  @doc("クライアントがプッシュ型の認可リクエスト（RFC 9126）を使わなければならないかどうか。")
  require_pushed_authorization_requests: boolean;
  @doc("このクライアントに発行するアクセストークンを DPoP に結び付けるかどうか（RFC 9449）。")
  dpop_bound_access_tokens: boolean;
  fapi_profile: FapiProfile;
  @doc("登録がインラインの JWK Set を与えたときだけ繰り返す。")
  jwks?: JWKS;
  @doc("登録が JWKS の URL を与えたときだけ繰り返す。")
  jwks_uri?: url;
  @doc("`tls_client_auth`（RFC 8705 の 2.1 節）のときだけ繰り返す。")
  tls_client_auth_subject_dn?: string;
  backchannel_logout_uri?: url;
  backchannel_logout_session_required: boolean;
  frontchannel_logout_uri?: url;
  frontchannel_logout_session_required: boolean;
}


@doc("このテナントに、指定した `client_id` の OAuth 2.0 のクライアントが存在しない。他のテナントのクライアントは存在しないものとして扱い、同じように答えるので、呼び出し元が見られないテナントのクライアントの存在を id だけで確かめることはできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:client_not_found` である。")
model OAuth2ClientNotFoundError is ProblemDetails;


@doc("リクエストが指定した `authorization_details` の種類は、このテナントに登録されていない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:type_not_found` である。")
model AuthorizationDetailTypeNotFoundError is ProblemDetails;


@doc("同じ名前の `authorization_details` の種類が、このテナントにすでに登録されている。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:type_exists` である。")
model AuthorizationDetailTypeExistsError is ProblemDetails;


@doc("リクエストが指定した MCP のリソースサーバーが、このテナントに存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:resource_server_not_found` である。")
model McpResourceServerNotFoundError is ProblemDetails;


@doc("同じ `resource` が、このテナントの MCP のリソースサーバーとしてすでに登録されている。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:resource_exists` である。")
model McpResourceServerExistsError is ProblemDetails;

}
