namespace IdMagic.Contract {

@doc("テナント単位のリソース上限値 (Hard / Soft)。システム管理者がテナントごとに上書きできる。")
model TenantQuota {
  users: int64;
  groups: int64;
  agents: int64;
  applications: int64;
  oauth2_clients: int64;
  active_sessions: int64;
  consents: int64;
  active_jobs: int64;
  audit_events_retained: int64;
  export_artifacts_bytes: int64;
}


@doc("テナント単位の現在のリソース利用量。")
model TenantUsage {
  users: int64;
  groups: int64;
  agents: int64;
  applications: int64;
  oauth2_clients: int64;
  active_sessions: int64;
  consents: int64;
  active_jobs: int64;
  audit_events_retained: int64;
  export_artifacts_bytes: int64;
}


@doc("system_admin がテナントの Quota を上書きする JSON リクエスト。")
model TenantQuotaUpdateRequest {
  users?: int64;
  groups?: int64;
  agents?: int64;
  applications?: int64;
  oauth2_clients?: int64;
  active_sessions?: int64;
  consents?: int64;
  active_jobs?: int64;
  audit_events_retained?: int64;
  export_artifacts_bytes?: int64;
}


@doc("テナント状態。Active は通常稼働、Disabled はテナント全体のプロトコル受付を停止 (`/authorize` `/token` 等が一律 invalid_request)。")
enum TenantStatus {
  Active: "Active",
  Disabled: "Disabled",
}


@doc("テナントの正規ロケーションの形。1 テナントは 1 つの正規ロケーションしか持たず、そこから issuer / cookie scope / WebAuthn RP ID が一意に決まる。Path は共有ホスト上の `{base}/realms/{realm}` で、ワイルドカード DNS も証明書も要らない既定。Subdomain はテナント固有ホスト `{realm}.{tenant_base_domain}` で、テナント固有 origin を必要とする機能 (`__Host-` cookie、テナント別 WebAuthn RP ID、テナント別ブランド origin) が成立する。tenant_base_domain が未設定の配備では Subdomain を選べない。")
enum TenantEndpointStyle {
  Path: "Path",
  Subdomain: "Subdomain",
}


@doc("テナント固有のパスワードポリシー上書き値。Authentication context の PasswordPolicy 標準値を\nテナント解決ポートで差し替えるためのスキーマ。省略フィールドは global default を継承し、\nchange_password / reset_password / 管理者によるユーザー作成の評価経路では\nテナント値が global より厳しい方向にのみ働く。")
model PasswordPolicyOverride {
  @doc("下限長 (≥ global default を推奨)")
  min_length?: int64;
  @doc("上限長")
  max_length?: int64;
  @doc("履歴照合件数 (PasswordHistoryNoReuse)")
  history_depth?: int64;
  @doc("パスワードの有効期限 (日)。0 または省略で無効 (NIST SP 800-63B-4 に沿った既定)。\n正の値を設定したテナントでは、期限を過ぎたパスワードを持つユーザーが次回ログイン成立後に\nupdate_password required action へ誘導される。認証自体は失敗しない。\nsystem ceiling は 30 日以上 3650 日以下で、範囲外は PolicyOverrideWeakerError で拒否する。")
  @minValue(0)
  max_age_days?: int64;
}

@doc("テナント (realm) 集約。Client、User、Consent、policy、branding はこの認可境界に閉じる。")
model Tenant {
  @minLength(1)
  id: string;
  @minLength(1)
  @maxLength(63)
  @pattern("^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$")
  realm: string;
  @minLength(1)
  @maxLength(200)
  display_name: string;
  status: TenantStatus = TenantStatus.Active;
  endpoint_style: TenantEndpointStyle = TenantEndpointStyle.Path;
  password_policy_override?: PasswordPolicyOverride;
  @pattern("^[a-z]{2}$")
  default_locale?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("tenant 単位の custom 属性定義集合。組み込み属性はコード (BuiltinUserAttributeDefs) が持ち、本集合は tenant 固有分のみ。tenant aggregate には埋め込まず独立 aggregate として持ち、tenant 削除時に cascade する。")
model TenantUserAttributeSchema {
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("key が一意で組み込み属性と衝突しない定義の並び。")
  attributes: UserAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("tenant 単位の Group custom 属性定義集合。Group には TenantUserAttributeSchema の builtin catalog に相当するものが無いため、本集合が実効定義そのものになる。tenant aggregate には埋め込まず独立 aggregate として持ち、tenant 削除時に cascade する。")
model TenantGroupAttributeSchema {
  @doc("所属テナント ID。")
  tenant_id: string = "default";
  @doc("key が一意な定義の並び。")
  attributes: GroupAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("Authentication context の PasswordPolicy の現行値。テナント設定 UI が「テナント上書きが\n無いときに実際に適用される値」をユーザに具体数字で示せるよう、API レスポンスに\n埋め込む。同時に override がこの値より弱い場合の reject 閾値でもある。")
model PasswordPolicyDefaults {
  @doc("Authentication context の PasswordPolicy.min_length。")
  min_length: int64;
  @doc("Authentication context の PasswordPolicy.max_length。")
  max_length: int64;
  @doc("Authentication context の PasswordPolicy.history_depth。")
  history_depth: int64;
  @doc("Authentication context の PasswordPolicy.max_age_days。省略は 0 (有効期限なし) と同じ意味で、\nこのフィールドを知らない旧クライアントの解釈と一致する。")
  max_age_days?: int64;
}


@doc("`/api/admin/settings` の返り値。テナント本体ではなく管理 UI 向けの projection で、\ntenant_id (UUID) / realm (URL slug) / display_name / password_policy_override に加え、\nUI 表示と weaker policy gating に使う Authentication context の PasswordPolicy 標準値を併せて返す。")
model AdminSettingsResponse {
  @doc("テナントの不変 UUID キー。")
  tenant_id: string;
  @doc("URL `/realms/{realm}/` に現れる slug。SCIM エンドポイント URL 等の組み立てに使う。")
  realm: string;
  display_name: string;
  password_policy_override?: PasswordPolicyOverride;
  password_policy_defaults: PasswordPolicyDefaults;
  @doc("テナントの既定 locale。未設定ならシステム既定 locale を使う。")
  default_locale?: string;
  @doc("カタログが同梱翻訳を持つ locale の一覧。管理 UI が既定 locale と通知テンプレートの選択肢を組み立てるために返す。")
  supported_locales: string[];
  quota: TenantQuota;
  usage: TenantUsage;
}


@doc("管理者が OAuth 2.0 AS / OIDC OP を RP または API client に設定するための正規エンドポイント集合。discovery / authorization server metadata を機械可読な正本とし、個別 URL は手動設定用に返す。")
model AdminOAuthEndpointSet {
  openid_configuration: string;
  oauth_authorization_server: string;
  authorization_endpoint: string;
  token_endpoint: string;
  userinfo_endpoint: string;
  jwks_uri: string;
  revocation_endpoint: string;
  introspection_endpoint: string;
  end_session_endpoint: string;
  registration_endpoint: string;
  pushed_authorization_request_endpoint: string;
  device_authorization_endpoint: string;
}


@doc("IdMagic が XML federation の新規署名に使う active X.509 証明書の公開情報。秘密鍵は含めず、rotation overlap の全証明書は metadata を正本とする。")
model AdminFederationCertificate {
  download_url: string;
  @doc("大文字16進数を colon 区切りにした SHA-256 fingerprint。")
  fingerprint_sha256: string;
  not_before: utcDateTime;
  not_after: utcDateTime;
}


@doc("SP に投入する tenant 共有 SAML IdP の正規情報。")
model AdminSamlIdentityProviderEndpointSet {
  entity_id: string;
  metadata_url: string;
  sso_url: string;
  slo_url: string;
  signing_certificate: AdminFederationCertificate;
}


@doc("WS-Federation RP / Microsoft Entra domain federation に投入する tenant の正規情報。")
model AdminWsFederationEndpointSet {
  realm: string;
  metadata_url: string;
  passive_logon_url: string;
  active_logon_url: string;
  metadata_exchange_url: string;
}


@doc("外部管理 client、SCIM client、本人用 client が API token と組み合わせて使う正規ベースURL。")
model AdminApiEndpointSet {
  management_api_base_url: string;
  scim_base_url: string;
  account_api_base_url: string;
}


@doc("`/api/admin/integration-endpoints` の read-only projection。request tenant の canonical issuer だけから導出し、path / subdomain / custom domain の知識を UI に持たせない。client secret、token、秘密鍵は含めない。")
model AdminIntegrationEndpointCatalog {
  issuer: string;
  oauth: AdminOAuthEndpointSet;
  saml: AdminSamlIdentityProviderEndpointSet;
  ws_federation: AdminWsFederationEndpointSet;
  apis: AdminApiEndpointSet;
}


@doc("`/api/admin/tenant/user_attribute_schema` の返り値。tenant 固有の custom 属性定義\n(attributes) に加え、参照用にコードが持つ組み込みカタログ (builtin) を併せて返し、\nUI が key 衝突を避けつつ全属性を一覧できるようにする。")
model TenantUserAttributeSchemaResponse {
  tenant_id: string;
  @doc("tenant が編集可能な custom 定義。")
  attributes: UserAttributeDef[];
  @doc("コード提供の読み取り専用カタログ (BuiltinUserAttributeDefs)。")
  builtin: UserAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("tenant の custom 属性定義を全置換する JSON リクエスト。組み込み属性との key 衝突・重複 key は use case 側で拒否する。")
model TenantUserAttributeSchemaUpdateRequest {
  @doc("新しい custom 定義一覧。空配列で全削除。")
  attributes: UserAttributeDef[];
}


@doc("`/api/admin/tenant/group_attribute_schema` の返り値。Group には builtin catalog が無いため、User 版と異なり builtin フィールドを持たない。")
model TenantGroupAttributeSchemaResponse {
  tenant_id: string;
  @doc("tenant が編集可能な Group custom 定義。")
  attributes: GroupAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("tenant の Group custom 属性定義を全置換する JSON リクエスト。重複 key は use case 側で拒否する。")
model TenantGroupAttributeSchemaUpdateRequest {
  @doc("新しい custom 定義一覧。空配列で全削除。")
  attributes: GroupAttributeDef[];
}


@doc("TenantBranding が持つ画像アセットの種別。")
enum TenantBrandingAssetKind {
  Logo: "Logo",
  Favicon: "Favicon",
}


@doc("hosted UI footer に表示する、順序固定の安全な外部リンク。ラベルはプレーンテキストとして描画し、URL は HTTPS のみを許可する。存在する場合は label と url の両方が非空でなければならない (片側だけの指定を拒否する)。")
model TenantFooterLink {
  @doc("利用者に表示するリンクテキスト。HTML / Markdown として解釈しない。")
  @minLength(1)
  @maxLength(80)
  label: string;
  @doc("遷移先。https scheme のみ許可する allowlist。")
  @minLength(1)
  @maxLength(2048)
  @pattern("^https://")
  url: string;
}


@doc("テナント単位の hosted UI ブランディング設定。Tenant 集約には埋め込まず、\nTenantUserAttributeSchema と同じ理由 (sparse な per-tenant config を Tenant 本体から分離する) で\n独立 entity として持つ。全フィールド optional で、行が存在しない、または全フィールドが未設定なら\n「branding 未設定」を意味し、GetTenantBranding はシステム既定にフォールバックする。\n色・リンク・テキストは限定トークンのみを受け付け、任意 CSS / HTML は構造的に受け付けない。")
model TenantBranding {
  @doc("所属テナント ID。")
  tenant_id: string;
  @doc("sign-in 画面等に出す表示名。空文字列は未設定を意味し、システム既定 (IdMagic) を表示する。")
  @maxLength(80)
  product_name?: string;
  @doc("テナント境界内に保存されたロゴ画像 object の opaque key。管理者は直接入力せず UploadTenantBrandingAsset (kind=Logo) を経由する。")
  logo_object_key?: string;
  @doc("IdP が保存済みロゴから生成する内部配信 URL。")
  logo_url?: string;
  @doc("テナント境界内に保存された favicon 画像 object の opaque key。UploadTenantBrandingAsset (kind=Favicon) を経由する。")
  favicon_object_key?: string;
  @doc("IdP が保存済み favicon から生成する内部配信 URL。")
  favicon_url?: string;
  @doc("ブランドカラー (プライマリ)。CSS custom property `--tenant-brand-primary` としてのみ注入する。`#rrggbb` 形式ならコントラスト比にかかわらず保存でき、空文字列は未設定へ正規化してシステム既定へ戻す。配色の可読性はテナントが判断する。")
  @pattern("^#[0-9a-fA-F]{6}$")
  primary_color?: string;
  @doc("ブランドカラー (アクセント)。CSS custom property `--tenant-brand-accent` としてのみ注入する。`#rrggbb` 形式ならコントラスト比にかかわらず保存でき、空文字列は未設定へ正規化してシステム既定へ戻す。配色の可読性はテナントが判断する。")
  @pattern("^#[0-9a-fA-F]{6}$")
  accent_color?: string;
  @doc("footer の先頭リンク。リンク順序は footer_link_1、footer_link_2 に固定する。")
  footer_link_1?: TenantFooterLink;
  @doc("footer の二番目のリンク。リンク順序は footer_link_1、footer_link_2 に固定する。")
  footer_link_2?: TenantFooterLink;
  @doc("footer に表示する補足テキスト (著作権表示等)。プレーンテキストとして描画側の既定エスケープでのみ表示する。")
  @maxLength(280)
  footer_text?: string;
  created_at: utcDateTime;
  @doc("更新のたびに進む。公開レスポンスの version / ETag として使う。")
  updated_at: utcDateTime;
}


@doc("GetTenantBranding / UpdateTenantBranding / asset upload・削除が返す公開安全な projection。\n未設定フィールドはシステム既定を表す (クライアント側で `IdMagic` 等のデフォルトに解決する)。")
model TenantBrandingResponse {
  product_name?: string;
  logo_url?: string;
  favicon_url?: string;
  primary_color?: string;
  accent_color?: string;
  footer_link_1?: TenantFooterLink;
  footer_link_2?: TenantFooterLink;
  footer_text?: string;
  @doc("branding が一度でも設定されていれば version / ETag として使う。未設定なら省略する。")
  updated_at?: utcDateTime;
}


@doc("管理者が branding のテキスト / 色 / リンクを更新する JSON リクエスト。画像アセットは UploadTenantBrandingAsset / DeleteTenantBrandingAsset 経由で別途扱う。指定フィールドのみ変更し、null を渡すとそのフィールドを未設定に戻す。primary_color / accent_color は空文字列でも未設定に戻す。")
model TenantBrandingUpdateRequest {
  product_name?: string;
  primary_color?: string;
  accent_color?: string;
  footer_link_1?: TenantFooterLink;
  footer_link_2?: TenantFooterLink;
  footer_text?: string;
}


@doc("画像アセットのアップロードまたは削除後の branding 表現。")
model TenantBrandingUploadResponse {
  branding: TenantBrandingResponse;
}


@doc("テナント内 admin が branding (テキスト / 色 / リンク / 画像アセット) を更新した。")
model TenantBrandingUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  changedFields: string[];
}


@doc("通知の用途を表す固定識別子。カタログはこの全 key に対し ja / en の組込み既定テンプレートを持つ。テナントは key を追加できない。AccountSecurityAlert は現在どの経路からも送信されず、カタログと編集面だけが先に存在する。")
enum NotificationTemplateKey {
  PasswordReset: "PasswordReset",
  EmailVerification: "EmailVerification",
  EmailChangeConfirmation: "EmailChangeConfirmation",
  AccountSecurityAlert: "AccountSecurityAlert",
  LifecycleWorkflowNotification: "LifecycleWorkflowNotification",
}


@doc("テナントによる通知テンプレート上書き 1 件。(tenant_id, template_key, locale) で一意。行が存在しない組み合わせは組込み既定 (BuiltinNotificationTemplate) を使う。上書きできるのは件名 / プレーンテキスト本文 / HTML 本文 / 差出人表示名だけで、任意 HTML 文書や CSS の全面差し替えは受け付けない。本文が使える差し込み変数は template_key ごとの許可集合に限られる。")
model NotificationTemplate {
  @doc("所属テナント ID。")
  tenant_id: string;
  template_key: NotificationTemplateKey;
  @doc("この上書きが適用される locale。カタログが同梱翻訳を持つ locale のみ受け付ける。")
  @minLength(1)
  @pattern("^[a-z]{2}$")
  locale: string;
  @doc("メール件名。改行を含められない単一行のテキストとして扱う。")
  @minLength(1)
  @maxLength(200)
  subject: string;
  @doc("プレーンテキスト本文。HTML として解釈しない。")
  @minLength(1)
  @maxLength(8000)
  body_text: string;
  @doc("HTML 本文。`<body>` 内に入る fragment であり、doctype / `<head>` / 文字エンコーディング / 本文コンテナのスタイルはシステムが供給する外枠が持つ。差し込み値は必ずエスケープして展開する。テキスト本文と必ず対で保存し、片方だけの上書きは作れない。")
  @minLength(1)
  @maxLength(20000)
  body_html: string;
  @doc("差出人の表示名。空文字列は未設定を意味し、システム既定の差出人表示になる。メールアドレス自体はサーバ設定であり上書きできない。")
  @maxLength(80)
  from_display_name?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("テンプレート一覧の 1 行。key × locale の組ごとに、上書き済みかどうかと現在有効な件名を返す。")
model NotificationTemplateSummary {
  template_key: NotificationTemplateKey;
  locale: string;
  @doc("true ならテナント上書きが存在する。false なら組込み既定。")
  customized: boolean;
  @doc("現在有効な件名 (上書きがあれば上書き、無ければ組込み既定)。")
  subject: string;
  @doc("上書きが存在する場合の最終更新時刻。組込み既定のみなら省略する。")
  updated_at?: utcDateTime;
}


@doc("テンプレート一覧。カタログの全 key × 全サポート locale を、上書きの有無にかかわらず返す。")
model NotificationTemplateListResponse {
  templates: NotificationTemplateSummary[];
  supported_locales: string[];
}


@doc("テンプレート 1 件の編集用表現。現在有効な文面と、組込み既定の文面、および使用できる差し込み変数の許可集合を同時に返す。管理 UI は既定との差分提示と「既定に戻す」を追加の問い合わせなしに描ける。")
model NotificationTemplateDetail {
  template_key: NotificationTemplateKey;
  locale: string;
  customized: boolean;
  subject: string;
  body_text: string;
  body_html: string;
  from_display_name?: string;
  @doc("組込み既定の件名。")
  default_subject: string;
  default_body_text: string;
  default_body_html: string;
  @doc("この template_key で使える差し込み変数名の許可集合。")
  placeholders: string[];
  updated_at?: utcDateTime;
}


@doc("テンプレート上書きの保存リクエスト。4 フィールドをまとめて置換する部分更新ではない全置換で、件名 / テキスト本文 / HTML 本文はいずれも必須。許可集合外の差し込み変数を含む場合は保存せず拒否する。")
model NotificationTemplateUpdateRequest {
  @minLength(1)
  @maxLength(200)
  subject: string;
  @minLength(1)
  @maxLength(8000)
  body_text: string;
  @minLength(1)
  @maxLength(20000)
  body_html: string;
  @maxLength(80)
  from_display_name?: string;
}


@doc("保存前の文面をサンプル値で描画するリクエスト。省略したフィールドは現在有効な文面を使う。")
model NotificationTemplatePreviewRequest {
  subject?: string;
  body_text?: string;
  body_html?: string;
  from_display_name?: string;
}


@doc("サンプル値を展開した描画結果。実際のメール送信は起こらない。")
model NotificationTemplatePreviewResponse {
  subject: string;
  body_text: string;
  body_html: string;
  from_display_name?: string;
}


@doc("テスト送信の結果。宛先は操作した管理者本人の検証済みメールアドレスに固定され、リクエストでは指定できない。")
model NotificationTemplateTestSendResponse {
  delivered: boolean;
  @doc("実際に送信した宛先 (操作者本人のアドレス)。UI が「どこへ送ったか」を明示するために返す。")
  to: string;
}


@doc("テナント内 admin が通知テンプレートの上書きを保存した。文面そのものは記録せず、対象の key と locale のみを記録する。")
model NotificationTemplateUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  templateKey: string;
  locale: string;
}


@doc("テナント内 admin が通知テンプレートの上書きを削除し組込み既定へ戻した。")
model NotificationTemplateReset {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  templateKey: string;
  locale: string;
}


@doc("管理者が新規テナントを登録する JSON リクエスト。`realm` は URL-safe slug。UUID キー (id) はサーバが採番する。")
model TenantCreateRequest {
  @minLength(1)
  @maxLength(63)
  @pattern("^[a-z0-9][a-z0-9-]{0,62}$")
  realm: string;
  @minLength(1)
  @maxLength(200)
  display_name: string;
}


@doc("管理者がテナントの表示名 / パスワードポリシー上書きを更新する JSON リクエスト。`id` と status は変更しない。status 変更は disable/enable 専用エンドポイントを経由。")
model TenantUpdateRequest {
  @minLength(1)
  @maxLength(200)
  display_name?: string;
  @doc("省略時は既存値を維持。null を渡したい場合は明示する。")
  password_policy_override?: PasswordPolicyOverride;
  @doc("テナント既定 locale。省略時は既存値を維持し、空文字列でシステム既定へ戻す。サポート外の locale は拒否する。")
  default_locale?: string;
}


@doc("テナント表現。管理 UI の一覧と詳細で同じ表現を返す。")
model TenantSummaryResponse {
  @doc("テナントの不変 UUID キー。")
  id: string;
  @doc("URL `/realms/{realm}/` に現れる slug 識別子。")
  realm: string;
  display_name: string;
  status: TenantStatus;
  password_policy_override?: PasswordPolicyOverride;
  default_locale?: string;
  quota: TenantQuota;
  usage: TenantUsage;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("管理 API のテナント一覧。")
model TenantListResponse {
  tenants: TenantSummaryResponse[];
}


@doc("system_admin が新規テナントを作成した。")
model TenantCreated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("system_admin がテナントの表示名を更新した。")
model TenantUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
  changedFields: string[];
}


@doc("system_admin がテナントのリソースクォータを上書きした。")
model TenantQuotaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("テナント内 admin が custom 属性定義 (TenantUserAttributeSchema) を更新した。値は持たず、変更後の custom 属性 key 一覧のみを記録する。")
model TenantUserAttributeSchemaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
  attributeKeys: string[];
}


@doc("テナント内 admin が Group custom 属性定義 (TenantGroupAttributeSchema) を更新した。値は持たず、変更後の custom 属性 key 一覧のみを記録する。")
model TenantGroupAttributeSchemaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
  attributeKeys: string[];
}


@doc("system_admin がテナントを無効化した。当該テナントのすべてのプロトコルルートは invalid_request で拒否される。")
model TenantDisabled {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("system_admin が無効化済みテナントを再有効化した。")
model TenantEnabled {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("テナントの password_policy_override がグローバルな下限ポリシーより弱いか、max_age_days が system ceiling の範囲外である。")
model PolicyOverrideWeakerError {

}


@doc("user_attribute_schema の定義が不正 (型不正、キー重複、組込み属性との衝突等)。")
model InvalidUserAttributeSchemaError {

}


@doc("group_attribute_schema の定義が不正 (型不正、キー重複等)。")
model InvalidGroupAttributeSchemaError {

}

}
