namespace IdMagic.Contract {

@doc("OAuth2 の Context が所有するクライアントの種類を、公表された言語として写したもの。送受信の値は RFC 6749 の 2.1 節に従う。")
enum ClientType {
  Public: "public",
  Confidential: "confidential",
}


@doc("OAuth2 の Context が公開するクライアントシークレットの資格情報の状態を、公表された言語として写したもの。")
enum ClientSecretCredentialStatus {
  Active: "Active",
  Expired: "Expired",
  Revoked: "Revoked",
}


@doc("OAuth2 の Context が所有するグラントの種類を、公表された言語として写したもの。送受信の値は、RFC 6749 の 4 節、RFC 8628 の 3.4 節（デバイスコード）、RFC 8693 の 2.1 節（トークン交換）、OpenID Connect CIBA Core の 4 節が登録する `grant_type` の値である。")
enum GrantType {
  AuthorizationCode: "authorization_code",
  RefreshToken: "refresh_token",
  ClientCredentials: "client_credentials",
  DeviceCode: "urn:ietf:params:oauth:grant-type:device_code",
  TokenExchange: "urn:ietf:params:oauth:grant-type:token-exchange",
  Ciba: "urn:openid:params:grant-type:ciba",
}


@doc("OAuth2 の Context が所有するトークンエンドポイントのクライアント認証の方式を、公表された言語として写したもの。送受信の値は、OpenID Connect Discovery 1.0 の 3 節と RFC 8705 の 2 節の `token_endpoint_auth_method` の値である。")
enum TokenEndpointAuthMethod {
  ClientSecretBasic: "client_secret_basic",
  ClientSecretPost: "client_secret_post",
  PrivateKeyJwt: "private_key_jwt",
  TlsClientAuth: "tls_client_auth",
  None: "none",
}


@doc("OAuth2 の Context が所有する認可のレスポンスの種類を、公表された言語として写したもの。送受信の値は RFC 6749 の 3.1.1 節に従い、対応するのは認可コードフローだけである。")
enum ResponseType {
  Code: "code",
}


@doc("OAuth2 の Context が所有する FAPI のセキュリティプロファイルを、公表された言語として写したもの。送受信の値は IdMagic が定める（backend/oauth2/client/domain/client.go）。")
enum FapiProfile {
  None: "none",
  Fapi2SecurityProfile: "fapi_2_security_profile",
}


@doc("OAuth2Client がすでに上限の二つの Active なクライアントシークレットの資格情報を保持しているので、これ以上発行できない。OAuth2 の Context が定める上限のエラーを、公表された言語として写したもの。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:client_secret_limit_exceeded` である。")
model ClientSecretLimitExceededError is ProblemDetails;


@doc("クライアントシークレットを追加で発行したときに OAuth2 の Context が公開するイベントを、公表された言語として写したもの。シークレットもそのハッシュも含まない。")
model ClientSecretIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  credentialId: string;
  expiresAt: utcDateTime;
}


@doc("クライアントシークレットを一つ失効させたときに OAuth2 の Context が公開するイベントを、公表された言語として写したもの。シークレットもそのハッシュも含まない。")
model ClientSecretRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  credentialId: string;
}


@doc("Application の種類。`federated` は SSO のフェデレーションで結ぶ業務アプリケーション、`weblink` は SSO のない外部リンク、`service` はマシン間の client_credentials の API クライアントで、Okta の API Services に当たり、ポータルのタイルもユーザーの割り当てもない。")
enum ApplicationKind {
  federated: "federated",
  weblink: "weblink",
  service: "service",
}


@doc("Application が有効かどうか。")
enum ApplicationStatus {
  active: "active",
  disabled: "disabled",
}


@doc("Application が使う唯一のプロトコル。作成後は変えられない。")
enum ApplicationProtocolType {
  oidc: "oidc",
  saml: "saml",
  wsfed: "wsfed",
}


@doc("一括作成で選ぶアプリケーションの種類。`oidc`、`saml`、`wsfed` は、フェデレーションの Application を対応するプロトコルの設定とともに作り、両方を原子的にカタログへ公開する。`weblink` は SSO のない外部リンクを作り、`service` は `service` の種類のマシン間の client_credentials の OAuth2 クライアントを作る。")
enum ApplicationProvisionType {
  oidc: "oidc",
  saml: "saml",
  wsfed: "wsfed",
  weblink: "weblink",
  service: "service",
}


@doc("Application が結び付く唯一のプロトコルの設定への参照。`type` に一致するキーだけを含み、プロトコルの行は、その `application_id` を通常の外部キーとして Application に関連付く。")
model ApplicationProtocol {
  @doc("プロトコルの種類。")
  type: ApplicationProtocolType;
  @doc("`oidc` が指す OAuth2 の `client_id`。")
  client_id?: string;
  @doc("`saml` が指す SAML の `entity_id`。")
  @maxLength(2048)
  entity_id?: string;
  @doc("`wsfed` が指す WS-Fed の RP の `wtrealm`。")
  @maxLength(2048)
  wtrealm?: string;
}


@doc("管理者がテナントごとに定めるポータルの節。Application は 0 個以上のカテゴリーを含み、ユーザーポータルはカテゴリーによってタイルを節にまとめる。割り当ての境界は越えず、節に入るのはそのユーザーに割り当てたアプリケーションだけである。")
model ApplicationCategory {
  @doc("属するテナント。")
  tenant_id: string = "default";
  @doc("カテゴリーの識別子。")
  id: string;
  @doc("節の表示名。")
  @minLength(1)
  name: string;
  @doc("節の表示の順序。小さいほど先で、同じなら名前の昇順にする。")
  position: int64;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("サインインのポリシーが求める認証の強度。`Password` はそれ以上を求めず（acr は urn:idmagic:acr:pwd を含む）、`Mfa` は第二要素を求める（acr は urn:idmagic:acr:mfa に等しく、amr は otp、webauthn、hwk、swk のいずれかを含む）。")
enum RequiredAuthnStrength {
  Password: "Password",
  Mfa: "Mfa",
}


@doc("サインインのポリシーが求める認証の強度。制限した選択肢である Password と Mfa を、内部の acr と amr へ対応付ける。Mfa を満たさないときは、ユーザーにステップアップを促す。")
model RequiredAuthnLevel {
  strength: RequiredAuthnStrength = RequiredAuthnStrength.Password;
}


@doc("MFA を必須にする前の事前登録の期間と、強制の開始後の管理者の承認による導入を制御する。`enforcement_start_at` より前は通常のサインインを許し、セルフサービスでの登録をユーザーに促す。その後は、登録していないユーザーはサインインを完了できず、`allow_admin_bypass` が true で有効な一回限りの登録の回避があるときだけ、登録専用のフローへ進む。`grace_period_seconds` は強制の開始後に回避を使える期間であり、自由に登録できる期間ではない。")
model MfaEnrollmentPolicy {
  enforcement_start_at?: utcDateTime;
  grace_period_seconds?: int64;
  allow_admin_bypass: boolean = false;
}


@doc("サインインのポリシーの静的なアクセスの条件。どれも最初の実装で実際に評価できる構造化された条件である。`network_allow_cidrs` はクライアントの IP を許可する CIDR と照合し、`reauth_max_age_seconds` は認証またはステップアップの新しさとして評価する。以前の自由入力のネットワークとデバイスのフィールドは廃止した。")
model AccessCondition {
  @doc("クライアントの IP が入ってよい CIDR。空は制限なしを意味する。空でなく、クライアントの IP がどれにも入らないか、IP をまったく判別できないときは、リクエストをフェイルクローズで拒否する。")
  network_allow_cidrs: string[];
  reauth_max_age_seconds?: int64;
}


@doc("サインインのポリシーの規則の一つ。")
model SignInRule {
  rule_id: string;
  name: string;
  enabled: boolean;
  required_authn: RequiredAuthnLevel;
  condition: AccessCondition;
  mfa_enrollment?: MfaEnrollmentPolicy;
  @doc("`required_authn` が `Mfa` のとき、記憶した信頼済みデバイス（amr が `tdev` を含む）で MFA の要求を満たしてよいかどうか。false は毎回 MFA を求め、本物の第二要素しか認めない。デフォルトは true だが、テナントが `trusted_device_max_age_seconds` を 0 のままにしている限り、信頼済みデバイスは発行も評価もされないので、実質的なデフォルトは何も記憶しないことである。")
  allow_trusted_device: boolean = true;
}


@doc("ApplicationCatalog が所有する、一つの Application のサインインのポリシー。規則を一つでも含むときは、そのアプリケーションについてテナントのデフォルトを丸ごと置き換え、規則がないときはデフォルトを適用する。これは上書きのモデルである。デフォルトより弱い設定も保存でき、UI がそれを警告する。")
model AppSignInPolicy {
  @doc("親の Application への、全体で一意な UUID の外部キー。テナントの境界はその親から決まる。")
  application_id: string;
  rules: SignInRule[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("テナントのデフォルトのサインインのポリシー。ApplicationCatalog がテナントごとに一つ所有し、独自のポリシーのないすべての Application でフェデレーションを開始するときに適用する基準である。アプリケーションにポリシーを設定すると、そのアプリケーションについてデフォルトを上書きする。")
model TenantDefaultSignInPolicy {
  tenant_id: string = "default";
  rules: SignInRule[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("管理 API での Application の表現。")
model AdminApplicationResponse {
  id: string;
  name: string;
  kind: ApplicationKind;
  status: ApplicationStatus;
  icon_url?: string;
  icon_object_key?: string;
  launch_url?: string;
  protocol?: ApplicationProtocol;
  category_ids: string[];
  category_names: string[];
  protocol_summary?: string;
  assigned_subject_count: int64;
  sign_in_policy_summary: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("Application の一覧のレスポンス。")
model AdminApplicationListResponse {
  applications: AdminApplicationResponse[];
}


@doc("ApplicationCategory を作成または更新するリクエスト。`position` を省くと最後に置く。")
model AdminApplicationCategoryRequest {
  @minLength(1)
  name: string;
  position?: int64;
}


@doc("管理 API での ApplicationCategory の表現。")
model AdminApplicationCategoryResponse {
  id: string;
  name: string;
  position: int64;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("ApplicationCategory の一覧のレスポンス。位置の昇順で、同じなら名前の昇順に並べる。")
model AdminApplicationCategoryListResponse {
  categories: AdminApplicationCategoryResponse[];
}


@doc("Application に割り当てるカテゴリーの集合を置き換えるリクエスト。`category_ids` に含められるのは、テナントにすでに存在するカテゴリーだけである。")
model ApplicationCategoriesRequest {
  category_ids: string[];
}


@doc("一回の呼び出しで Application を作成するリクエスト。種類を指定し、対応するプロトコルの設定を一緒に含める。バックエンドはまずカタログの外でプロトコルの行を用意し、一つのトランザクションで Application を挿入して `row.application_id` を設定する。後半が失敗したプロトコルの行はカタログの外の有効な設定なので、意味のない Application は残らない。")
model AdminApplicationCreateRequest {
  @minLength(1)
  name: string;
  @doc("oidc、saml、wsfed、weblink、service のいずれか。")
  type: ApplicationProvisionType;
  @doc("ポータルのタイルが開く起動の URL。`weblink` では必須である。")
  launch_url?: string;
  @doc("`type` が `oidc` のとき必須である。作成する OAuth2 クライアントのリダイレクト URI。")
  redirect_uris?: string[];
  @doc("作成する OAuth2 クライアントのスコープ。")
  scope?: string;
  @doc("`type` が `oidc` のときに作成するクライアントの種類。デフォルトは `confidential` である。")
  client_type?: ClientType;
  @doc("`type` が `oidc` のときに作成するクライアントの、クライアント認証の方式。作成時に確定し、後から変えられない。")
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  @doc("`token_endpoint_auth_method=private_key_jwt` のための公開鍵の JWKS の URI。")
  jwks_uri?: url;
  @doc("`token_endpoint_auth_method=tls_client_auth` のための証明書の Subject DN。")
  @maxLength(2048)
  tls_client_auth_subject_dn?: string;
  @doc("`type` が `wsfed` のとき必須である。作成する WS-Fed の RP の `wtrealm`。")
  @maxLength(2048)
  wtrealm?: string;
  @doc("`type` が `wsfed` のとき必須である。許可する `wreply` の宛先。")
  reply_urls?: string[];
  @doc("`type=wsfed` のための NameID の形式。")
  name_id_format?: string;
  @doc("`type=wsfed` のための NameID の取得元の属性。")
  name_id_source?: string;
  @doc("`type` が `saml` のとき必須である。作成する SAML の SP のエンティティ ID。")
  @maxLength(2048)
  entity_id?: string;
  @doc("`type` が `saml` のとき必須である。Assertion Consumer Service の URL。")
  acs_urls?: string[];
  @doc("`type=saml` のとき、サービスプロバイダーの Single Logout Service の URL。")
  @maxLength(2048)
  slo_url?: string;
  @doc("`type=saml` のとき、アサーションに加えて SAML のレスポンス全体にも署名するかどうか。")
  sign_response?: boolean;
  @doc("`type=saml` のとき、サービスプロバイダーの AuthnRequest に署名が必要かどうか。true のときは、`authn_request_signing_certificate_pem` が検証の鍵を与える。")
  want_authn_requests_signed?: boolean;
  @doc("`type=saml` のとき、署名した AuthnRequest を検証する PEM の証明書。")
  authn_request_signing_certificate_pem?: string;
  @doc("`type=saml` と `type=wsfed` のとき、アサーションを発行するときの発行者と署名鍵を与える ID プロバイダーのプロファイル。")
  idp_profile_id?: string;
}


@doc("一括作成の結果。`type=oidc` のときは、作成した OAuth2 クライアントの `client_id` と `client_secret` を一度だけ返す。シークレットは再び読めない。")
model AdminApplicationCreateResponse {
  application: AdminApplicationResponse;
  @doc("`type` が `oidc` のときに返す、生成したクライアントの id。")
  client_id?: string;
  @doc("`type` が `oidc` のときに一度だけ返すシークレット。")
  client_secret?: string;
}


@doc("アプリケーションの詳細画面のための、Application の OIDC のプロトコルが指す OAuth2 クライアントの実際の設定。高度なフィールドも含む。`token_endpoint_auth_method`、`client_type`、`fapi_profile` は更新の契約の下で変えられず、参考として示すだけである。")
model ApplicationOidcConfig {
  client_id: string;
  client_type: ClientType;
  redirect_uris: string[];
  grant_types: GrantType[];
  response_types: ResponseType[];
  token_endpoint_auth_method: TokenEndpointAuthMethod;
  scope: string;
  require_pushed_authorization_requests: boolean;
  dpop_bound_access_tokens: boolean;
  fapi_profile: FapiProfile;
  @doc("client_secret_basic または client_secret_post を使う confidential のクライアントのときだけ true。private_key_jwt、mTLS、public のクライアントでは false。")
  client_secret_rotatable: boolean;
  @doc("ハッシュも平文も含まない資格情報のメタデータ。エディターは、作成時刻、有効期限、資格情報が Active、Expired、Revoked のどれかを示す。")
  secret_credentials: ApplicationClientSecretCredentialMetadata[];
  @doc("ID トークンと UserInfo の `sub` のクレームに使う取得元の属性。設定しないとユーザーの内部の id を使い、これがデフォルトの振る舞いである。WS-Fed と SAML の `name_id_source` と同じ方法でサブジェクトの識別子を上書きする。")
  sub_source_attribute?: string;
  @doc("このアプリケーションのクレームの開示の上書き。スコープから決まる標準のクレーム（profile、email など）が開示するものとは独立に、開示するものを絞り込むか対応付け直す。公開範囲が `private` の属性と予約済みのクレームの種類は、下限によって拒否する。")
  rules: ClaimMappingRule[];
}


@doc("Application のエディター向けに OAuth2 の資格情報を射影する、機微でないメタデータ。OAuth2 のハッシュも平文も公開せず、状態は `revoked_at`、`expires_at`、評価の時刻から導く。")
model ApplicationClientSecretCredentialMetadata {
  credential_id: string;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
  status: ClientSecretCredentialStatus;
}


@doc("アプリケーションの詳細画面のための、Application の WS-Fed のプロトコルが指す Relying Party の実際の設定。RP ごとの高度なフィールド（オーディエンス、トークンの種類、クレームの規則）を含む。Entra のドメインフェデレーションは、テナントごとの設定として別に扱う。")
model ApplicationWsFedConfig {
  wtrealm: string;
  reply_urls: string[];
  audience: string;
  token_type: WsFedTokenType;
  name_id_format: string;
  name_id_source: string;
  rules: ClaimMappingRule[];
}


@doc("アプリケーションの詳細画面のための、Application の SAML のプロトコルが指すサービスプロバイダーの実際の設定。")
model ApplicationSamlConfig {
  @doc("すべての SAML の SP が割り当てられる、同じテナントの IdP プロファイルの id。")
  idp_profile_id: string;
  entity_id: string;
  acs_urls: string[];
  slo_url?: string;
  audience?: string;
  name_id_format: string;
  name_id_source: string;
  sign_assertion: boolean;
  sign_response: boolean;
  want_authn_requests_signed: boolean;
  authn_request_signing_certificate_pem?: string;
  rules: ClaimMappingRule[];
}


@doc("アプリケーションの詳細のレスポンス。Application そのものに加えて、プロトコルの関係を通じて解決した実際の設定を含む。`oidc`、`saml`、`wsfed` のうち設定されるのは `Application.protocol.type` に一致する一つまでである。")
model AdminApplicationDetailResponse {
  application: AdminApplicationResponse;
  oidc?: ApplicationOidcConfig;
  wsfed?: ApplicationWsFedConfig;
  saml?: ApplicationSamlConfig;
  @doc("専用のサインインのポリシーのエンドポイントが返すのと同じ表示。詳細画面は、二度目のリクエストなしにポリシーを描画できる。")
  sign_in_policy: AppSignInPolicyResponse;
}


@doc("OIDC のプロトコルの設定を更新するリクエスト。指定したフィールドだけを変える。高度なフィールドは含むが、`token_endpoint_auth_method` など更新の契約の下で変えられないフィールドは含まない。")
model ApplicationOidcConfigUpdateRequest {
  redirect_uris?: string[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  scope?: string;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
  @doc("省くと既存の設定を保つ。空文字列を送るとデフォルトのユーザーの内部の id に戻す。")
  sub_source_attribute?: string;
  @doc("省くと既存のクレームの開示の上書きを保つ。空の配列を送ると上書きなし、つまり標準のクレームだけに戻す。")
  rules?: ClaimMappingRule[];
}


@doc("Application の OIDC のプロトコルのシークレットをローテーションするリクエスト。`grace_days` は省くと 7 で、受け付けるのは 0（即座に失効）または 1 から 30 日だけである。")
model RotateApplicationClientSecretRequest {
  grace_days?: int64;
}


@doc("新しいシークレットを一度だけ返す。ハッシュも既存のシークレットも、平文を再び読む方法も含まない。")
model RotateApplicationClientSecretResponse {
  client_secret: string;
  grace_until?: utcDateTime;
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("既存の資格情報を変えずに、Application の OIDC のプロトコルに有効期限付きの追加のシークレットを発行するリクエスト。`expires_in_days` は省くと 90 で、受け付けるのは 1 から 730 日だけである。")
model IssueApplicationClientSecretRequest {
  expires_in_days?: int64;
}


@doc("追加で発行したシークレットを一度だけ返す。ハッシュも既存のシークレットも、平文を再び読む方法も含まない。")
model IssueApplicationClientSecretResponse {
  client_secret: string;
  credential: ApplicationClientSecretCredentialMetadata;
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("指定した資格情報を一つ失効させた後に返す、機微でないメタデータの一覧。")
model RevokeApplicationClientSecretResponse {
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("WS-Federation のプロトコルの設定を更新するリクエスト。指定したフィールドだけを変える。RP ごとの高度なフィールド（オーディエンス、トークンの種類、クレームの規則）を含む。Entra のドメインフェデレーションはここでは扱わない。")
model ApplicationWsFedConfigUpdateRequest {
  reply_urls?: string[];
  audience?: string;
  token_type?: WsFedTokenType;
  name_id_format?: string;
  name_id_source?: string;
  rules?: ClaimMappingRule[];
}


@doc("SAML のプロトコルの設定を部分的に更新するリクエスト。ACS と SLO の URL、オーディエンス、NameID、署名のポリシー、クレームの規則を更新する。")
model ApplicationSamlConfigUpdateRequest {
  @doc("SP に割り当てる、同じテナントの IdP プロファイル。専用のプロファイルをすでに別の SP が使っているときは、更新を拒否する。")
  idp_profile_id?: string;
  acs_urls?: string[];
  slo_url?: string;
  audience?: string;
  name_id_format?: string;
  name_id_source?: string;
  sign_assertion?: boolean;
  sign_response?: boolean;
  want_authn_requests_signed?: boolean;
  authn_request_signing_certificate_pem?: string;
  rules?: ClaimMappingRule[];
}


@doc("Application を更新するリクエスト。指定したフィールドだけを変える。")
model AdminApplicationUpdateRequest {
  name?: string;
  status?: ApplicationStatus;
  launch_url?: string;
}


@doc("アイコンをアップロードまたは削除した後の Application の表現。")
model ApplicationIconUploadResponse {
  application: AdminApplicationResponse;
}


@doc("Application を割り当てるリクエスト。`visibility` を省くと `visible` になる。")
model ApplicationAssignmentRequest {
  subject_type: AssignmentSubjectType;
  @minLength(1)
  subject_id: string;
  visibility?: AssignmentVisibility;
}


@doc("割り当ての表現。")
model ApplicationAssignmentResponse {
  subject_type: AssignmentSubjectType;
  subject_id: string;
  visibility: AssignmentVisibility;
  created_at: utcDateTime;
}


@doc("割り当ての一覧のレスポンス。")
model ApplicationAssignmentListResponse {
  assignments: ApplicationAssignmentResponse[];
}


@doc("管理者が Application のサインインのポリシーを置き換えるリクエスト。規則を一つでも送るとテナントのデフォルトを上書きし、規則を送らないとデフォルトの適用に戻す。")
model AppSignInPolicyRequest {
  rules: SignInRule[];
}


@doc("Application のサインインのポリシーの表現。アプリケーションの詳細画面が、アプリケーション自身の上書き（`policy`）、テナントのデフォルト（`tenant_default`）、実際に適用される規則の並びを区別できるようにする。")
model AppSignInPolicyResponse {
  policy: AppSignInPolicy;
  tenant_default: TenantDefaultSignInPolicy;
  @doc("上書きのモデルの下で実際に評価する規則の並び。アプリケーションが規則を含むときは `policy.rules`、含まないときは `tenant_default.rules` である。")
  effective_rules: SignInRule[];
  @doc("アプリケーション自身のポリシーがデフォルトより弱いとき、すなわち認証の強度、再認証の期間、許可するネットワークを緩めるときに true。UI の警告のためのもので、評価は止めない。")
  weaker_than_default: boolean;
}


@doc("管理者がテナントのデフォルトのサインインのポリシーを置き換えるリクエスト。")
model TenantDefaultSignInPolicyRequest {
  rules: SignInRule[];
}


@doc("テナントのデフォルトのサインインのポリシーの表現。")
model TenantDefaultSignInPolicyResponse {
  policy: TenantDefaultSignInPolicy;
  @doc("TOTP の要素も WebAuthn の要素もまったく保持していない、テナントの有効なユーザーの数。MFA を必須にする前に影響を見積もるために使う。")
  unenrolled_user_count: int64;
}


@doc("ユーザーポータルに示す、割り当てられたアプリケーションの一つ。")
model MyApplication {
  id: string;
  name: string;
  kind: ApplicationKind;
  icon_url?: string;
  launch_url?: string;
  @doc("含むカテゴリーの id。ポータルはこれらでタイルを節に振り分ける。")
  category_ids: string[];
}


@doc("ユーザーポータルの節の見出しの一つ。テナントのカテゴリーの定義を位置の昇順で表す。")
model PortalApplicationCategory {
  id: string;
  name: string;
}


@doc("ユーザーポータル向けの、割り当てられたアプリケーションの一覧。手動の並び順を保存していればそれを優先し、なければ名前の昇順にする。`categories` はテナントのカテゴリーの定義を位置の昇順で含み、ポータルはそれを節の見出しに使う。")
model MyApplicationListResponse {
  applications: MyApplication[];
  categories: PortalApplicationCategory[];
}


@doc("ユーザーポータルの手動の並び順。`user_id` をキーとし、Application の表示順を `application_id` の値の並びとして保持する。`user_id` は既存の、全体で一意な User を指す。並びの中で割り当てが外れたアプリケーションは一覧から抜け、並びにない割り当て済みのアプリケーションは名前の昇順で末尾に加える。`application_ids` はデータベースの外部キーではなく配列として保持するので、そのユーザーに割り当てられ表示される Application に限るのは、ReorderMyApplications のユースケースの検証である。")
model ApplicationOrdering {
  user_id: string;
  application_ids: string[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("ユーザーが、自身のポータルの手動の並び順を保存する。`application_ids` に含められるのは、自身に割り当てられ表示されるアプリケーションだけである。")
model ReorderMyApplicationsRequest {
  application_ids: string[];
}


@doc("ユーザーが保存したポータルの手動の並び順。何も保存していないときは `application_ids` が空である。")
model MyApplicationOrderResponse {
  application_ids: string[];
}


@doc("Application を作成した。")
model ApplicationCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
}


@doc("Application のメタデータを更新した。")
model ApplicationUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  changedFields: string[];
}


@doc("保存した Application のアイコンをアップロードまたは削除した。")
model ApplicationIconUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  action: string;
}


@doc("Application を削除した。")
model ApplicationDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
}


@doc("ユーザーまたはグループを Application に割り当てた。")
model ApplicationAssigned {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  subjectType: string;
  subjectId: string;
}


@doc("Application から割り当てを外した。")
model ApplicationUnassigned {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  subjectType: string;
  subjectId: string;
}


@doc("ApplicationCategory を作成した。")
model ApplicationCategoryCreated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  categoryId: string;
}


@doc("ApplicationCategory を更新した。")
model ApplicationCategoryUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  categoryId: string;
}


@doc("ApplicationCategory を削除した。割り当てていた Application からも外した。")
model ApplicationCategoryDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  categoryId: string;
}


@doc("Application のサインインのポリシーを更新した。")
model AppSignInPolicyUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
}


@doc("Application のクレームの開示の上書き（規則と、`sub` または NameID の取得元の属性）を更新した。")
model ApplicationClaimMappingUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  protocol: string;
}


@doc("テナントのデフォルトのサインインのポリシーを、消去も含めて更新した。")
model TenantDefaultSignInPolicyUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
}


@doc("Application のサインインのポリシーがフェデレーションを拒否した。")
model AppAccessDeniedByPolicy {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  protocol: string;
  subject: string;
  reason: string;
}


@doc("Application のサインインのポリシーの認証の強度を満たすために、ステップアップが必要になった。")
model AppStepUpRequired {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  protocol: string;
  subject: string;
}


@doc("ApplicationCategory の `name` が空である。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:category_name_required` である。")
model CategoryNameRequiredError is ProblemDetails;


@doc("Application に割り当てた `category_id` が、テナントの ApplicationCategory の記録に存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:unknown_category` である。")
model UnknownCategoryError is ProblemDetails;


@doc("リクエストの対象の Application がこのテナントに存在しない。他のテナントの Application は存在しないものとして扱い、同じように答える。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:application_not_found` である。")
model ApplicationNotFoundError is ProblemDetails;


@doc("リクエストの対象の ApplicationCategory がこのテナントに存在しない。他のテナントのカテゴリーは存在しないものとして扱い、同じように答える。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:category_not_found` である。")
model ApplicationCategoryNotFoundError is ProblemDetails;


@doc("リクエストが指定したアイコンの画像が存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:not_found` である。")
model ApplicationIconNotFoundError is ProblemDetails;


@doc("Application の結び付きが指すプロトコルの記録（OAuth 2.0 のクライアント、SAML のサービスプロバイダー、WS-Federation の Relying Party）が存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:not_found` である。")
model ApplicationProtocolTargetNotFoundError is ProblemDetails;

}
