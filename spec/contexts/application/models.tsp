namespace IdMagic.Contract {

@doc("OAuth2 context が publish する client 種別の published language stub。")
enum ClientType {
  Public: "Public",
  Confidential: "Confidential",
}


@doc("OAuth2 context が publish する client secret credential 状態の published language stub。")
enum ClientSecretCredentialStatus {
  Active: "Active",
  Expired: "Expired",
  Revoked: "Revoked",
}


@doc("OAuth2 context が publish するトークン発行方式の published language stub。")
enum GrantType {
  AuthorizationCode: "AuthorizationCode",
  RefreshToken: "RefreshToken",
  ClientCredentials: "ClientCredentials",
  DeviceCode: "DeviceCode",
  TokenExchange: "TokenExchange",
  Ciba: "Ciba",
}


@doc("OAuth2 context が publish する /token クライアント認証方式の published language stub。")
enum TokenEndpointAuthMethod {
  ClientSecretBasic: "ClientSecretBasic",
  ClientSecretPost: "ClientSecretPost",
  PrivateKeyJwt: "PrivateKeyJwt",
  TlsClientAuth: "TlsClientAuth",
  None: "None",
}


@doc("OAuth2 context が publish する /authorize 応答種別の published language stub。")
enum ResponseType {
  Code: "Code",
}


@doc("OAuth2 context が publish する FAPI セキュリティプロファイルの published language stub。")
enum FapiProfile {
  None: "None",
  Fapi2SecurityProfile: "Fapi2SecurityProfile",
}


@doc("OAuth2 context が定義する Active client secret credential 最大2件の上限 error の published language stub。")
model ClientSecretLimitExceededError {

}


@doc("OAuth2 context が publish する client secret 追加発行イベントの published language stub。secret と hash は含めない。")
model ClientSecretIssued {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  credentialId: unknown;
  expiresAt: utcDateTime;
}


@doc("OAuth2 context が publish する client secret 個別失効イベントの published language stub。secret と hash は含めない。")
model ClientSecretRevoked {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  clientId: string;
  credentialId: unknown;
}


@doc("Application の種別。federated は SSO フェデレーションを束ねる業務アプリ、weblink は SSO しない外部リンク、service は M2M / client_credentials の API クライアント (Okta の API Services 相当。ポータルタイル・利用者割当を持たない)。")
enum ApplicationKind {
  federated: "federated",
  weblink: "weblink",
  service: "service",
}


@doc("Application の有効状態。")
enum ApplicationStatus {
  active: "active",
  disabled: "disabled",
}


@doc("Application が利用する単一 protocol の種別。作成後は変更できない。")
enum ApplicationProtocolType {
  oidc: "oidc",
  saml: "saml",
  wsfed: "wsfed",
}


@doc("一括作成で選ぶアプリ種別。oidc / saml / wsfed は federated Application と対応する protocol 設定を作成して catalog へ原子的に公開し、weblink は SSO なしの外部リンク、service は M2M / client_credentials の OAuth2 client (service kind) を作る。")
enum ApplicationProvisionType {
  oidc: "oidc",
  saml: "saml",
  wsfed: "wsfed",
  weblink: "weblink",
  service: "service",
}


@doc("Application に紐づく単一 protocol 設定への参照。type に対応する key だけを持ち、protocol row の application_id から Application へ通常の外部キーで関連付ける。")
model ApplicationProtocol {
  @doc("protocol 種別。")
  type: ApplicationProtocolType;
  @doc("oidc が指す OAuth2 client_id。")
  client_id?: string;
  @doc("saml が指す SAML entity_id。")
  @maxLength(2048)
  entity_id?: string;
  @doc("wsfed が指す WS-Fed RP の wtrealm。")
  @maxLength(2048)
  wtrealm?: string;
}


@doc("管理者が tenant 単位で定義するポータルの分類セクション。Application に 0..N 個\n付与され、利用者ポータルはこのカテゴリ単位でタイルをセクション表示する。割当境界は越えず、\nセクションには利用者に割当済みのアプリだけが並ぶ。")
model ApplicationCategory {
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("カテゴリ識別子。")
  id: unknown;
  @doc("セクション表示名。")
  @minLength(1)
  name: string;
  @doc("セクションの表示順。小さいほど先。同値は name 昇順。")
  position: int64;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("サインインポリシーが要求する認証強度。Password は追加要求なし (acr ⊇ urn:idmagic:acr:pwd)、Mfa は第二要素必須 (acr = urn:idmagic:acr:mfa、amr に otp / webauthn / hwk / swk のいずれかを含む)。")
enum RequiredAuthnStrength {
  Password: "Password",
  Mfa: "Mfa",
}


@doc("サインインポリシーが要求する認証強度。制約された選択肢 (Password / Mfa) を内部の acr / amr へ写像する。Mfa を満たさない場合は step-up で昇格を促す。")
model RequiredAuthnLevel {
  strength: RequiredAuthnStrength = RequiredAuthnStrength.Password;
}


@doc("MFA 必須化の事前登録期間と、強制開始後の管理者承認済みオンボーディングを制御する。enforcement_start_at より前は通常ログインを許可して self-service 登録を促し、開始後は未登録ユーザーをログイン完了させず、allow_admin_bypass=true かつ有効な単発 enrollment bypass がある場合だけ登録専用 flow へ進める。grace_period_seconds は開始後に bypass を利用できる期間であり、自由登録期間ではない。")
model MfaEnrollmentPolicy {
  enforcement_start_at?: utcDateTime;
  grace_period_seconds?: int64;
  allow_admin_bypass: boolean = false;
}


@doc("サインインポリシーの静的アクセス条件。すべて初期実装で実際に評価できる構造化条件のみを持つ。network_allow_cidrs はクライアント IP を許可 CIDR に突き合わせ、reauth_max_age_seconds は認証または step-up の recency として評価する。旧 network / device の自由入力は廃止した。")
model AccessCondition {
  @doc("許可するクライアント IP の CIDR 一覧。空なら制限なし。非空でクライアント IP がいずれの CIDR にも含まれない、または IP を取得できない場合は fail-closed で拒否する。")
  network_allow_cidrs: string[];
  reauth_max_age_seconds?: int64;
}


@doc("サインインポリシーの単一ルール。")
model SignInRule {
  rule_id: string;
  name: string;
  enabled: boolean;
  required_authn: RequiredAuthnLevel;
  condition: AccessCondition;
  mfa_enrollment?: MfaEnrollmentPolicy;
  @doc("required_authn=Mfa のとき、記憶済みの信頼済みデバイス (amr に tdev) で MFA 要件を\n満たしてよいかどうか。false は「毎回 MFA」を意味し、本物の第二要素の提示だけを認める。\n既定は true だが、テナントが trusted_device_max_age_seconds を 0 のままにしていれば\nそもそも信頼済みデバイスは発行も評価もされないので、実効的な既定は「記憶しない」である。")
  allow_trusted_device: boolean = true;
}


@doc("Application ごとのサインインポリシー。ApplicationCatalog が所有する。有効ルールを持つときテナントデフォルトを完全に上書き (置換) して評価され、持たなければデフォルトが適用される (上書きモデル)。デフォルトより弱い設定でも保存でき、その場合は UI で警告する。")
model AppSignInPolicy {
  @doc("親 Application の global UUID foreign key。tenant 境界は親 Application から導出する。")
  application_id: unknown;
  rules: SignInRule[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("テナントデフォルトのサインインポリシー。ApplicationCatalog が tenant 単位で所有し、独自ポリシーを持たない全 Application の federation 開始で適用される baseline。アプリ個別ポリシーが設定されるとそのアプリではデフォルトを上書きする。")
model TenantDefaultSignInPolicy {
  tenant_id: string = "default";
  rules: SignInRule[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("管理 API が返す Application 表現。")
model AdminApplicationResponse {
  id: unknown;
  name: string;
  kind: ApplicationKind;
  status: ApplicationStatus;
  icon_url?: string;
  icon_object_key?: string;
  launch_url?: string;
  protocol?: ApplicationProtocol;
  category_ids: unknown[];
  category_names: string[];
  protocol_summary?: string;
  assigned_subject_count: int64;
  sign_in_policy_summary: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("Application 一覧レスポンス。")
model AdminApplicationListResponse {
  applications: AdminApplicationResponse[];
}


@doc("ApplicationCategory の作成 / 更新リクエスト。position 省略時は末尾扱い。")
model AdminApplicationCategoryRequest {
  @minLength(1)
  name: string;
  position?: int64;
}


@doc("管理 API が返す ApplicationCategory 表現。")
model AdminApplicationCategoryResponse {
  id: unknown;
  name: string;
  position: int64;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("ApplicationCategory 一覧レスポンス。position 昇順 (同値は name 昇順)。")
model AdminApplicationCategoryListResponse {
  categories: AdminApplicationCategoryResponse[];
}


@doc("Application に付与するカテゴリ集合を置き換えるリクエスト。category_ids は所属テナントの 既存カテゴリのみを含む。")
model ApplicationCategoriesRequest {
  category_ids: unknown[];
}


@doc("Application 一括作成リクエスト。種別を選び、対応する protocol 設定もまとめて受け取る。backend は protocol row を catalog 外として準備した後、Application insert と protocol row.application_id の設定を同一 transaction で行う。後半が失敗した protocol row は有効な catalog 外設定であり、意味のない Application は残さない。")
model AdminApplicationCreateRequest {
  @minLength(1)
  name: string;
  @doc("oidc / saml / wsfed / weblink / service。")
  type: ApplicationProvisionType;
  @doc("ポータルのタイルから開く起動 URL。weblink では必須。")
  launch_url?: string;
  @doc("type=oidc のとき必須。生成する OAuth2 client のリダイレクト URI。")
  redirect_uris?: string[];
  @doc("生成する OAuth2 client のスコープ。")
  scope?: string;
  @doc("type=oidc の生成 client 種別。既定は confidential。")
  client_type?: ClientType;
  @doc("type=oidc の生成 client 認証方式。作成時に確定し以後不変。")
  token_endpoint_auth_method?: TokenEndpointAuthMethod;
  @doc("token_endpoint_auth_method=private_key_jwt の公開鍵 JWKS URI。")
  jwks_uri?: url;
  @doc("token_endpoint_auth_method=tls_client_auth の証明書 Subject DN。")
  @maxLength(2048)
  tls_client_auth_subject_dn?: string;
  @doc("type=wsfed のとき必須。生成する WS-Fed RP の wtrealm。")
  @maxLength(2048)
  wtrealm?: string;
  @doc("type=wsfed のとき必須。許可する wreply。")
  reply_urls?: string[];
  @doc("type=wsfed の NameID 形式。")
  name_id_format?: string;
  @doc("type=wsfed の NameID ソース属性。")
  name_id_source?: string;
  @doc("type=saml のとき必須。生成する SAML SP の entity ID。")
  @maxLength(2048)
  entity_id?: string;
  @doc("type=saml のとき必須。Assertion Consumer Service URL。")
  acs_urls?: string[];
}


@doc("一括作成の結果。type=oidc のときは生成した OAuth2 client の client_id / client_secret を 一度だけ返す (再表示不可)。")
model AdminApplicationCreateResponse {
  application: AdminApplicationResponse;
  @doc("type=oidc のとき返る生成クライアント ID。")
  client_id?: string;
  @doc("type=oidc のとき一度だけ返るシークレット。")
  client_secret?: string;
}


@doc("Application の OIDC protocol が指す OAuth2 client の実設定 (アプリ詳細表示用)。advanced 項目を含む。token_endpoint_auth_method / client_type / fapi_profile は更新契約上の不変項目で 表示専用。")
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
  @doc("client_secret_basic / client_secret_post の confidential client のみ true。private_key_jwt、mTLS、public client は false。")
  client_secret_rotatable: boolean;
  @doc("hash・平文を含まない credential metadata。editor は作成日時・有効期限・Active/Expired/Revoked 状態を表示する。")
  secret_credentials: ApplicationClientSecretCredentialMetadata[];
  @doc("ID Token / UserInfo の sub claim に使う source 属性。未設定なら\nuser の内部 id (既定挙動) を使う。WS-Fed/SAML の name_id_source と同じ位置付けの subject\n識別子上書き。")
  sub_source_attribute?: string;
  @doc("このアプリ向けの claim release 上書き。scope 主導の標準 claim\n(profile/email 等) による開示可否とは独立に、開示範囲を絞る/写像するために追加する。\nvisibility=Private の属性や reserved claim type は floor により拒否される。")
  rules: ClaimMappingRule[];
}


@doc("Application editor 向けに OAuth2 credential を写像した非機密メタデータ。OAuth2 の hash や平文を露出せず、状態は revoked_at、expires_at、評価時刻から導出する。")
model ApplicationClientSecretCredentialMetadata {
  credential_id: unknown;
  created_at: utcDateTime;
  expires_at?: utcDateTime;
  revoked_at?: utcDateTime;
  status: ClientSecretCredentialStatus;
}


@doc("Application の WS-Fed protocol が指す relying party の実設定 (アプリ詳細表示用)。per-RP の advanced 項目 (audience / token_type / claim 規則) を含む。Entra domain federation は テナント単位の設定として別に扱う。")
model ApplicationWsFedConfig {
  wtrealm: string;
  reply_urls: string[];
  audience: string;
  token_type: WsFedTokenType;
  name_id_format: string;
  name_id_source: string;
  rules: ClaimMappingRule[];
}


@doc("Application の SAML protocol が指す service provider の実設定 (アプリ詳細表示用)。")
model ApplicationSamlConfig {
  @doc("SAML SPに必ず割り当てられる同一テナント内のIdP profile ID。")
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


@doc("アプリ詳細レスポンス。Application 本体に加え、protocol relation から解決した実設定を含む。oidc / saml / wsfed のうち Application.protocol.type に対応する最大1個だけが設定される。")
model AdminApplicationDetailResponse {
  application: AdminApplicationResponse;
  oidc?: ApplicationOidcConfig;
  wsfed?: ApplicationWsFedConfig;
  saml?: ApplicationSamlConfig;
}


@doc("OIDC protocol 設定の更新リクエスト。指定フィールドのみ変更する。advanced 項目を含む。 token_endpoint_auth_method など更新契約上の不変項目は含めない。")
model ApplicationOidcConfigUpdateRequest {
  redirect_uris?: string[];
  grant_types?: GrantType[];
  response_types?: ResponseType[];
  scope?: string;
  require_pushed_authorization_requests?: boolean;
  dpop_bound_access_tokens?: boolean;
  @doc("省略時は既存設定を維持する。空文字列を送ると既定 (user の内部 id) に戻す。")
  sub_source_attribute?: string;
  @doc("省略時は既存の claim release 上書きを維持する。空配列を送ると上書きなし (標準 claim のみ) に戻す。")
  rules?: ClaimMappingRule[];
}


@doc("Application の OIDC protocol に対する secret rotation 要求。grace_days は未指定時7、0（即時 revoke）または1..30日だけ受理する。")
model RotateApplicationClientSecretRequest {
  grace_days?: int64;
}


@doc("新 secret を一度だけ返す。hash、既存 secret、平文の再取得経路は含まない。")
model RotateApplicationClientSecretResponse {
  client_secret: string;
  grace_until?: utcDateTime;
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("Application の OIDC protocol に対し、既存 credential を変更せず期限付き secret を追加発行する要求。expires_in_days は未指定時90、1..730日だけ受理する。")
model IssueApplicationClientSecretRequest {
  expires_in_days?: int64;
}


@doc("追加発行した secret を一度だけ返す。hash、既存 secret、平文の再取得経路は含まない。")
model IssueApplicationClientSecretResponse {
  client_secret: string;
  credential: ApplicationClientSecretCredentialMetadata;
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("指定 credential の個別失効後に返す非機密 metadata 一覧。")
model RevokeApplicationClientSecretResponse {
  credentials: ApplicationClientSecretCredentialMetadata[];
}


@doc("WS-Federation protocol 設定の更新リクエスト。指定フィールドのみ変更する。per-RP の advanced 項目 (audience / token_type / claim 規則) を含む。Entra domain federation は対象外。")
model ApplicationWsFedConfigUpdateRequest {
  reply_urls?: string[];
  audience?: string;
  token_type?: WsFedTokenType;
  name_id_format?: string;
  name_id_source?: string;
  rules?: ClaimMappingRule[];
}


@doc("SAML protocol 設定の部分更新リクエスト。ACS/SLO、audience、NameID、署名方針、claim 規則を更新する。")
model ApplicationSamlConfigUpdateRequest {
  @doc("SPへ割り当てる同一テナント内のIdP profile。dedicated profileが別SPで使用中なら更新を拒否する。")
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


@doc("Application 更新リクエスト。指定フィールドのみ変更する。")
model AdminApplicationUpdateRequest {
  name?: string;
  status?: ApplicationStatus;
  launch_url?: string;
}


@doc("Application icon アップロードまたは削除後の Application 表現。")
model ApplicationIconUploadResponse {
  application: AdminApplicationResponse;
}


@doc("Application 割当リクエスト。visibility 省略時は visible。")
model ApplicationAssignmentRequest {
  subject_type: AssignmentSubjectType;
  @minLength(1)
  subject_id: string;
  visibility?: AssignmentVisibility;
}


@doc("割当表現。")
model ApplicationAssignmentResponse {
  subject_type: AssignmentSubjectType;
  subject_id: string;
  visibility: AssignmentVisibility;
  created_at: utcDateTime;
}


@doc("割当一覧レスポンス。")
model ApplicationAssignmentListResponse {
  assignments: ApplicationAssignmentResponse[];
}


@doc("管理者が Application の sign-in policy を置き換えるリクエスト。有効ルールを送るとテナントデフォルトを上書きし、空で送るとデフォルト適用に戻る。")
model AppSignInPolicyRequest {
  rules: SignInRule[];
}


@doc("Application の sign-in policy 表現。アプリ詳細で「このアプリの上書き (policy)」「テナントデフォルト (tenant_default)」\n「実際に適用される effective ルール列」を区別して表示できるようにする。")
model AppSignInPolicyResponse {
  policy: AppSignInPolicy;
  tenant_default: TenantDefaultSignInPolicy;
  @doc("上書きモデルで実際に評価されるルール列。アプリが有効ルールを持てば policy.rules、持たなければ tenant_default.rules。")
  effective_rules: SignInRule[];
  @doc("アプリ独自ポリシーがデフォルトより弱い (認証強度・再認証時間・許可ネットワークのいずれかを緩める) とき true。UI 警告用で評価は妨げない。")
  weaker_than_default: boolean;
}


@doc("管理者がテナントデフォルト sign-in policy を置き換えるリクエスト。")
model TenantDefaultSignInPolicyRequest {
  rules: SignInRule[];
}


@doc("テナントデフォルト sign-in policy 表現。")
model TenantDefaultSignInPolicyResponse {
  policy: TenantDefaultSignInPolicy;
  @doc("テナント内の active user のうち TOTP / WebAuthn factor を 1 件も持たない人数。MFA 必須化前の影響確認に使う。")
  unenrolled_user_count: int64;
}


@doc("利用者ポータルに表示する割当済みアプリ 1 件。")
model MyApplication {
  id: unknown;
  name: string;
  kind: ApplicationKind;
  icon_url?: string;
  launch_url?: string;
  @doc("付与カテゴリの id 群。ポータルはこれでタイルをセクションに振り分ける。")
  category_ids: unknown[];
}


@doc("利用者ポータルのセクション見出し 1 件。tenant のカテゴリ定義を position 昇順で表す。")
model PortalApplicationCategory {
  id: unknown;
  name: string;
}


@doc("利用者ポータルの割当済みアプリ一覧。手動並び順があればそれを優先し、無ければ name 昇順。\ncategories は tenant のカテゴリ定義を position 昇順で含み、ポータルはこれをセクション見出しに使う。")
model MyApplicationListResponse {
  applications: MyApplication[];
  categories: PortalApplicationCategory[];
}


@doc("利用者ポータルでの手動並び順。user_id をキーに、Application の\n表示順を application_id の順序列として持つ。user_id は global unique な既存 User を指す。\n順序列に在って現在は未割当の application は一覧から除外し、順序列に無い割当アプリは name 昇順で末尾に付ける。application_ids は配列で保持するため DB 外部キーではなく ReorderMyApplications の use case 検証で、本人に割当済み visible な Application のみに制限する。")
model ApplicationOrdering {
  user_id: string;
  application_ids: unknown[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("利用者が自分のポータル手動並び順を保存する。application_ids は自分に割当済み visible なアプリのみを含む。")
model ReorderMyApplicationsRequest {
  application_ids: unknown[];
}


@doc("利用者が保存済みのポータル手動並び順。未保存なら application_ids は空。")
model MyApplicationOrderResponse {
  application_ids: unknown[];
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


@doc("Application の保存済みアイコンをアップロードまたは削除した。")
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


@doc("Application にユーザー / グループを割当てた。")
model ApplicationAssigned {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  subjectType: string;
  subjectId: string;
}


@doc("Application の割当を解除した。")
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


@doc("ApplicationCategory を削除した。付与済みの Application からも除かれる。")
model ApplicationCategoryDeleted {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  categoryId: string;
}


@doc("Application の sign-in policy を更新した。")
model AppSignInPolicyUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
}


@doc("Application の claim release 上書き (rules / sub または NameID の source 属性) を更新した。")
model ApplicationClaimMappingUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  applicationId: string;
  protocol: string;
}


@doc("テナントデフォルト sign-in policy を更新した (クリアを含む)。")
model TenantDefaultSignInPolicyUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
}


@doc("Application sign-in policy により federation を拒否した。")
model AppAccessDeniedByPolicy {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  protocol: string;
  subject: string;
  reason: string;
}


@doc("Application sign-in policy の認証強度を満たすため step-up が必要になった。")
model AppStepUpRequired {
  occurredAt: utcDateTime;
  tenantId: string;
  applicationId: string;
  protocol: string;
  subject: string;
}


@doc("ApplicationCategory の name が空。")
model CategoryNameRequiredError {

}


@doc("Application に付与しようとした category_id が所属テナントの ApplicationCategory に存在しない。")
model UnknownCategoryError {

}

}
