namespace IdMagic.Contract {

@doc("テナントごとのリソースの上限。ハードの上限とソフトの上限がある。システム管理者はテナントごとに上書きできる。")
model TenantQuota {
  users: int64;
  groups: int64;
  agents: int64;
  applications: int64;
  oauth2_clients: int64;
  active_sessions: int64;
  consents: int64;
  active_jobs: int64;
  @doc("SsfStream の上限。送信と受信を合わせて数える。")
  ssf_streams: int64;
  audit_events_retained: int64;
  export_artifacts_bytes: int64;
}


@doc("テナントが現在使っている各リソースの量。")
model TenantUsage {
  users: int64;
  groups: int64;
  agents: int64;
  applications: int64;
  oauth2_clients: int64;
  active_sessions: int64;
  consents: int64;
  active_jobs: int64;
  ssf_streams: int64;
  audit_events_retained: int64;
  export_artifacts_bytes: int64;
}


@doc("system_admin がテナントの Quota を上書きする JSON のリクエスト。")
model TenantQuotaUpdateRequest {
  @minValue(0)
  users?: int64;
  @minValue(0)
  groups?: int64;
  @minValue(0)
  agents?: int64;
  @minValue(0)
  applications?: int64;
  @minValue(0)
  oauth2_clients?: int64;
  @minValue(0)
  active_sessions?: int64;
  @minValue(0)
  consents?: int64;
  @minValue(0)
  active_jobs?: int64;
  @minValue(0)
  ssf_streams?: int64;
  @minValue(0)
  audit_events_retained?: int64;
  @minValue(0)
  export_artifacts_bytes?: int64;
}


@doc("テナントの状態。`active` は通常どおり動き、`disabled` はテナントがプロトコルのトラフィックを一切受け付けなくなるので、`/authorize`、`/token` などがすべて invalid_request になる。")
enum TenantStatus {
  Active: "active",
  Disabled: "disabled",
}


@doc("テナントの正規の所在の形。テナントにはちょうど一つあり、発行者、Cookie のスコープ、WebAuthn の RP ID はすべてここから決まる。`path` は共有のホスト上の `{base}/realms/{realm}` で、ワイルドカードの DNS も証明書も要らないのでデフォルトである。`subdomain` はテナント固有のホスト `{realm}.{tenant_base_domain}` で、テナント固有のオリジンが必要な機能、すなわち `__Host-` の Cookie、テナントごとの WebAuthn の RP ID、テナントごとのブランディングしたオリジンを動かす。`tenant_base_domain` を設定していないデプロイメントでは、`subdomain` を選べない。")
enum TenantEndpointStyle {
  Path: "path",
  Subdomain: "subdomain",
}


@doc("テナントごとのパスワードポリシーの上書き。Authentication の Context の PasswordPolicy の標準値を、テナントの解決のポートで置き換えるためのスキーマである。省いたフィールドは全体のデフォルトを引き継ぎ、change_password、reset_password、管理者によるユーザーの作成の評価の経路では、テナントの値は全体の値より厳しい方向にだけ動かせる。")
model PasswordPolicyOverride {
  @doc("最小の長さ。全体のデフォルト以上を推奨する。")
  min_length?: int64;
  @doc("最大の長さ。")
  max_length?: int64;
  @doc("照合する過去のパスワードの数（PasswordHistoryNoReuse）。")
  history_depth?: int64;
  @doc("パスワードが有効な日数。0 または省略は有効期限を無効にし、NIST SP 800-63B-4 に沿ってこれをデフォルトとする。正の値を設定したテナントでは、パスワードが期限を過ぎたユーザーを、次のサインインが成功した後に `update_password` の必須のアクションへ向かわせる。認証そのものは失敗しない。システムの上限は 30 から 3650 日で、その外の値は PolicyOverrideWeakerError で拒否する。")
  @minValue(0)
  max_age_days?: int64;
}

@doc("テナント（レルム）の集約。Client、User、Consent、ポリシー、ブランディングはすべてこの認可の境界の中にとどまる。")
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
  @doc("Token Exchange（RFC 8693）の `act` の連鎖の最大の深さの、テナントによる上書き。省くとシステムのデフォルトを適用する。上書きは制限を厳しくすることだけができ、システムのデフォルトを超える値は PolicyOverrideWeakerError で拒否する。この非対称な規則は password_policy_override と同じで、テナントの設定が認可の境界を広げることを防ぐ。")
  @minValue(1)
  max_delegation_depth?: int64;
  @doc("信頼済みデバイス（Authentication の Context の「このデバイスを記憶する」）が有効な秒数。0 または省略はテナント全体でこの機能を無効にし、デバイスを記憶せず、第二要素も省かない。システムの上限は 7,776,000 秒（90 日）で、それより大きい値は PolicyOverrideWeakerError で拒否する。")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
  @pattern("^[a-z]{2}$")
  default_locale?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("一つのテナントのカスタム属性の定義の集合。組み込みの属性はコード（BuiltinUserAttributeDefs）にあり、この集合はテナントが加えたものだけを保持する。テナントの集約に埋め込まず別の集約とし、テナントを削除すると連鎖して削除する。")
model TenantUserAttributeSchema {
  @doc("属するテナント。")
  tenant_id: string = "default";
  @doc("定義。キーは一意で、どの組み込みの属性とも衝突しない。")
  attributes: UserAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("一つのテナントの Group のカスタム属性の定義の集合。Group には TenantUserAttributeSchema の組み込みの一覧に当たるものがないので、この集合が有効な定義のすべてである。テナントの集約に埋め込まず別の集約とし、テナントを削除すると連鎖して削除する。")
model TenantGroupAttributeSchema {
  @doc("属するテナント。")
  tenant_id: string = "default";
  @doc("定義。キーは一意である。")
  attributes: GroupAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("Authentication の Context の PasswordPolicy の現在の値。テナントが何も上書きしないときに実際に何が適用されるかを、テナントの設定の UI が具体的な数値で示せるように、API のレスポンスに埋め込む。同時に、上書きを弱いとして拒否する閾値でもある。")
model PasswordPolicyDefaults {
  @doc("Authentication の Context の PasswordPolicy の `min_length`。")
  min_length: int64;
  @doc("Authentication の Context の PasswordPolicy の `max_length`。")
  max_length: int64;
  @doc("Authentication の Context の PasswordPolicy の `history_depth`。")
  history_depth: int64;
  @doc("Authentication の Context の PasswordPolicy の `max_age_days`。省くと 0、すなわち有効期限なしを意味し、このフィールドを知らない古いクライアントの読み方と一致する。")
  max_age_days?: int64;
}


@doc("`/api/admin/settings` が返すもの。テナントそのものではなく管理 UI 向けの射影で、tenant_id（UUID）、realm（URL のスラッグ）、表示名、パスワードポリシーの上書きに加えて、表示と弱いポリシーのゲートの両方に使う Authentication の Context の PasswordPolicy の標準値を返す。")
model AdminSettingsResponse {
  @doc("テナントの変わらない UUID のキー。")
  tenant_id: string;
  @doc("URL `/realms/{realm}/` に現れるスラッグ。SCIM のエンドポイントの URL などを組み立てるのに使う。")
  realm: string;
  display_name: string;
  password_policy_override?: PasswordPolicyOverride;
  password_policy_defaults: PasswordPolicyDefaults;
  @doc("Token Exchange の `act` の連鎖の最大の深さの、テナントによる上書き。省くとシステムのデフォルトを適用する。")
  @minValue(1)
  max_delegation_depth?: int64;
  @doc("テナントに上書きがないときに適用する委任の深さで、上書きが超えられない上限でもある。管理 UI が具体的な制限を示し、送る前に大きすぎる値を拒否できるように、レスポンスでこの値を公開する。")
  max_delegation_depth_default: int64;
  @doc("信頼済みデバイスが有効な秒数。0 または省略はこの機能を無効にする。")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
  @doc("信頼済みデバイスが有効でいられる秒数の上限。管理 UI が制限を具体的に示し、送る前により大きい値を拒否できるように返す。")
  trusted_device_max_age_seconds_ceiling: int64;
  @doc("テナントのデフォルトのロケール。設定しないと、システムのデフォルトのロケールを使う。")
  default_locale?: string;
  @doc("カタログが翻訳を提供するロケール。管理 UI がデフォルトのロケールと通知のテンプレートの選択肢を作れるように返す。")
  supported_locales: string[];
  quota: TenantQuota;
  usage: TenantUsage;
}


@doc("OAuth 2.0 の認可サーバーと OpenID Connect のプロバイダーに対して、Relying Party または API クライアントを設定するために管理者が必要とする正規のエンドポイント。ディスカバリーと認可サーバーのメタデータが機械可読な基準であり、個々の URL は手作業での設定のために返す。")
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


@doc("IdMagic が新しい XML のフェデレーションのメッセージに署名するのに使う、有効な X.509 証明書の公開部分。秘密鍵は含まず、ローテーションの重複期間内のすべての証明書については、引き続きメタデータが基準である。")
model AdminFederationCertificate {
  download_url: string;
  @doc("SHA-256 のフィンガープリント。コロンで区切った大文字の 16 進数で表す。")
  fingerprint_sha256: string;
  not_before: utcDateTime;
  not_after: utcDateTime;
}


@doc("SP に設定する、テナントの共有の SAML IdP の正規の情報。")
model AdminSamlIdentityProviderEndpointSet {
  entity_id: string;
  metadata_url: string;
  sso_url: string;
  slo_url: string;
  signing_certificate: AdminFederationCertificate;
}


@doc("WS-Federation の RP または Microsoft Entra のドメインフェデレーションに設定する、テナントの正規の情報。")
model AdminWsFederationEndpointSet {
  realm: string;
  metadata_url: string;
  passive_logon_url: string;
  active_logon_url: string;
  metadata_exchange_url: string;
}


@doc("外部の管理用のクライアント、SCIM のクライアント、セルフサービスのクライアントが API トークンとともに使う正規のベース URL。")
model AdminApiEndpointSet {
  management_api_base_url: string;
  scim_base_url: string;
  account_api_base_url: string;
}


@doc("`/api/admin/integration-endpoints` の読み取り専用の射影。すべてをリクエストのテナントの正規の発行者だけから導くので、UI はパス、サブドメイン、カスタムドメインについて何も知る必要がない。クライアントシークレット、トークン、秘密鍵は含まない。")
model AdminIntegrationEndpointCatalog {
  issuer: string;
  oauth: AdminOAuthEndpointSet;
  saml: AdminSamlIdentityProviderEndpointSet;
  ws_federation: AdminWsFederationEndpointSet;
  apis: AdminApiEndpointSet;
}


@doc("`/api/admin/tenant/user_attribute_schema` が返すもの。テナント自身のカスタム属性の定義（`attributes`）に加えて、コードが保持する組み込みの一覧（`builtin`）を参考として返すので、UI はキーの衝突を避けながらすべての属性を並べられる。")
model TenantUserAttributeSchemaResponse {
  tenant_id: string;
  @doc("テナントが編集できるカスタムの定義。")
  attributes: UserAttributeDef[];
  @doc("コードが提供する読み取り専用の一覧（BuiltinUserAttributeDefs）。")
  builtin: UserAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("テナントのカスタム属性の定義を丸ごと置き換える JSON のリクエスト。組み込みの属性と衝突するキーと重複するキーは、どちらもユースケースで拒否する。")
model TenantUserAttributeSchemaUpdateRequest {
  @doc("新しいカスタムの定義の一覧。空の配列はすべてを削除する。")
  attributes: UserAttributeDef[];
}


@doc("`/api/admin/tenant/group_attribute_schema` が返すもの。Group には組み込みの一覧がないので、User 版と違い `builtin` のフィールドを含まない。")
model TenantGroupAttributeSchemaResponse {
  tenant_id: string;
  @doc("テナントが編集できる Group のカスタムの定義。")
  attributes: GroupAttributeDef[];
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("テナントの Group のカスタム属性の定義を丸ごと置き換える JSON のリクエスト。重複するキーはユースケースで拒否する。")
model TenantGroupAttributeSchemaUpdateRequest {
  @doc("新しいカスタムの定義の一覧。空の配列はすべてを削除する。")
  attributes: GroupAttributeDef[];
}


@doc("TenantBranding が保持する画像のアセットの種類。")
enum TenantBrandingAssetKind {
  Logo: "logo",
  Favicon: "favicon",
}


@doc("ホストする UI のフッターに決まった順序で示す、安全な外部リンク。ラベルはプレーンテキストとして描画し、URL は HTTPS だけを許す。リンクがあるときは `label` と `url` のどちらも空であってはならず、片方だけを与えると拒否する。")
model TenantFooterLink {
  @doc("ユーザーに示すリンクのテキスト。HTML や Markdown として解釈することはない。")
  @minLength(1)
  @maxLength(80)
  label: string;
  @doc("リンクの行き先。許可リストにあるのは https のスキームだけである。")
  @minLength(1)
  @maxLength(2048)
  @pattern("^https://")
  url: string;
}


@doc("テナントごとのホストする UI のブランディングの設定。TenantUserAttributeSchema と同じ理由で、Tenant の集約に埋め込まず別のエンティティとして保持する。テナントごとのまばらな設定を Tenant そのものから分けるためである。どのフィールドも任意で、行がないとき、またはすべてのフィールドが未設定の行は、ブランディングが未設定であることを意味し、GetTenantBranding はシステムのデフォルトへ戻る。色、リンク、テキストは限られたトークンだけを受け付け、任意の CSS や HTML は構造上受け付けられない。")
model TenantBranding {
  @doc("属するテナント。")
  tenant_id: string;
  @doc("サインインの画面などに示す名前。空文字列は未設定を意味し、システムのデフォルト（IdMagic）を示す。")
  @maxLength(80)
  product_name?: string;
  @doc("テナントの境界の中に保存したロゴの画像のオブジェクトの不透明なキー。管理者が直接入力することはなく、`kind=Logo` の UploadTenantBrandingAsset から得る。")
  logo_object_key?: string;
  @doc("保存したロゴから IdP が組み立てる、内部の配信用の URL。")
  logo_url?: string;
  @doc("テナントの境界の中に保存したファビコンの画像のオブジェクトの不透明なキー。`kind=Favicon` の UploadTenantBrandingAsset から得る。")
  favicon_object_key?: string;
  @doc("保存したファビコンから IdP が組み立てる、内部の配信用の URL。")
  favicon_url?: string;
  @doc("主たるブランドの色。CSS のカスタムプロパティ `--tenant-brand-primary` としてだけ注入する。`#rrggbb` の形式であればコントラスト比にかかわらず保存でき、空文字列は未設定へ正規化してシステムのデフォルトに戻す。配色が読みやすいかどうかはテナントが判断する。")
  @pattern("^#[0-9a-fA-F]{6}$")
  primary_color?: string;
  @doc("アクセントのブランドの色。CSS のカスタムプロパティ `--tenant-brand-accent` としてだけ注入する。`#rrggbb` の形式であればコントラスト比にかかわらず保存でき、空文字列は未設定へ正規化してシステムのデフォルトに戻す。配色が読みやすいかどうかはテナントが判断する。")
  @pattern("^#[0-9a-fA-F]{6}$")
  accent_color?: string;
  @doc("一つ目のフッターのリンク。順序は footer_link_1、footer_link_2 の順に固定する。")
  footer_link_1?: TenantFooterLink;
  @doc("二つ目のフッターのリンク。順序は footer_link_1、footer_link_2 の順に固定する。")
  footer_link_2?: TenantFooterLink;
  @doc("フッターに示す補足のテキスト。たとえば著作権の表示である。プレーンテキストとして描画し、描画器のデフォルトのエスケープ以外の保護はない。")
  @maxLength(280)
  footer_text?: string;
  created_at: utcDateTime;
  @doc("更新のたびに進む。公開のレスポンスのバージョンと ETag に使う。")
  updated_at: utcDateTime;
}


@doc("GetTenantBranding、UpdateTenantBranding、アセットのアップロードと削除が返す、公開して安全な射影。未設定のフィールドはシステムのデフォルトを表し、クライアントが `IdMagic` などへ解決する。")
model TenantBrandingResponse {
  product_name?: string;
  logo_url?: string;
  favicon_url?: string;
  primary_color?: string;
  accent_color?: string;
  footer_link_1?: TenantFooterLink;
  footer_link_2?: TenantFooterLink;
  footer_text?: string;
  @doc("ブランディングを一度でも設定した後は、バージョンと ETag に使う。未設定の間は省く。")
  updated_at?: utcDateTime;
}


@doc("管理者がブランディングのテキスト、色、リンクを更新する JSON のリクエスト。画像のアセットは別に、UploadTenantBrandingAsset と DeleteTenantBrandingAsset で扱う。指定したフィールドだけを変え、null を渡すとそのフィールドを未設定に戻す。`primary_color` と `accent_color` は空文字列でも未設定に戻る。")
model TenantBrandingUpdateRequest {
  product_name?: string;
  primary_color?: string;
  accent_color?: string;
  footer_link_1?: TenantFooterLink;
  footer_link_2?: TenantFooterLink;
  footer_text?: string;
}


@doc("画像のアセットをアップロードまたは削除した後のブランディングの表現。")
model TenantBrandingUploadResponse {
  branding: TenantBrandingResponse;
}


@doc("テナントの管理者が、ブランディングのテキスト、色、リンク、または画像のアセットを更新した。")
model TenantBrandingUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  changedFields: string[];
}


@doc("通知の目的を表す固定の識別子。カタログは、これらのすべてのキーについて、日本語と英語の組み込みのデフォルトのテンプレートを保持し、テナントはキーを加えられない。AccountSecurityAlert はすべての種類のセキュリティ通知を一つのキーで扱い、何が起きたかは `event_description` と `device_summary` の置換で表す。ProvisioningConnectionQuarantined は User ではなくプロビジョニング接続の `notification_email` へ送るので、受信者のロケールは常にテナントのデフォルトに任せる。")
enum NotificationTemplateKey {
  PasswordReset: "password_reset",
  EmailVerification: "email_verification",
  EmailChangeConfirmation: "email_change_confirmation",
  AccountSecurityAlert: "account_security_alert",
  AgentActionApprovalRequest: "agent_action_approval_request",
  LifecycleWorkflowNotification: "lifecycle_workflow_notification",
  ProvisioningConnectionQuarantined: "provisioning_connection_quarantined",
}


@doc("テナントによる一つの通知のテンプレートの上書きで、`(tenant_id, template_key, locale)` で一意である。行のない組み合わせは、組み込みのデフォルト（BuiltinNotificationTemplate）を使う。上書きできるのは、件名、プレーンテキストの本文、HTML の本文、送信者の表示名だけで、HTML の文書全体や CSS の置き換えは受け付けない。本文で使える置換変数は、その template_key の許可する集合に限る。")
model NotificationTemplate {
  @doc("属するテナント。")
  tenant_id: string;
  template_key: NotificationTemplateKey;
  @doc("この上書きを適用するロケール。カタログが翻訳を提供するロケールだけを受け付ける。")
  @minLength(1)
  @pattern("^[a-z]{2}$")
  locale: string;
  @doc("メールの件名。一行として扱い、改行を含められない。")
  @minLength(1)
  @maxLength(200)
  subject: string;
  @doc("プレーンテキストの本文。HTML として解釈することはない。")
  @minLength(1)
  @maxLength(8000)
  body_text: string;
  @doc("HTML の本文。`<body>` の中に入る断片であり、doctype、`<head>`、文字コード、本文のコンテナーのスタイルは、すべてシステムが与える外枠から来る。置換値は展開の前に必ずエスケープする。常にテキストの本文と一緒に保存し、片方だけの上書きは作れない。")
  @minLength(1)
  @maxLength(20000)
  body_html: string;
  @doc("送信者の表示名。空文字列は未設定を意味し、システムのデフォルトの送信者の表示を使う。メールアドレスそのものはサーバーの設定で、上書きできない。")
  @maxLength(80)
  from_display_name?: string;
  created_at: utcDateTime;
  updated_at: utcDateTime;
}


@doc("テンプレートの一覧の一行。キーとロケールの組ごとに、上書きしているかどうかと現在有効な件名を返す。")
model NotificationTemplateSummary {
  template_key: NotificationTemplateKey;
  locale: string;
  @doc("テナントの上書きがあるときは true、組み込みのデフォルトが有効なときは false。")
  customized: boolean;
  @doc("現在有効な件名。上書きがあればその件名、なければ組み込みのデフォルトである。")
  subject: string;
  @doc("上書きがあるとき、それを最後に更新した時刻。組み込みのデフォルトだけが有効なときは省く。")
  updated_at?: utcDateTime;
}


@doc("テンプレートの一覧。上書きの有無にかかわらず、カタログのすべてのキーを、対応するすべてのロケールについて返す。")
model NotificationTemplateListResponse {
  templates: NotificationTemplateSummary[];
  supported_locales: string[];
}


@doc("編集のための一つのテンプレートの表現。現在有効な文面、組み込みのデフォルトの文面、許可する置換変数の集合をまとめて返すので、管理 UI は別のリクエストなしに、デフォルトとの違いを示し「デフォルトに戻す」を提供できる。")
model NotificationTemplateDetail {
  template_key: NotificationTemplateKey;
  locale: string;
  customized: boolean;
  subject: string;
  body_text: string;
  body_html: string;
  from_display_name?: string;
  @doc("組み込みのデフォルトの件名。")
  default_subject: string;
  default_body_text: string;
  default_body_html: string;
  @doc("この template_key で許可する置換変数の名前の集合。")
  placeholders: string[];
  updated_at?: utcDateTime;
}


@doc("テンプレートの上書きを保存するリクエスト。四つのフィールドは一組として置き換えるもので部分更新ではなく、件名、テキストの本文、HTML の本文はすべて必須である。許可する集合の外の置換変数があると、何も保存せずにリクエストを拒否する。")
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


@doc("保存していない文面をサンプルの値で描画するリクエスト。省いたフィールドは現在有効な文面を使う。")
model NotificationTemplatePreviewRequest {
  subject?: string;
  body_text?: string;
  body_html?: string;
  from_display_name?: string;
}


@doc("サンプルの値を展開した描画の結果。実際にメールは送らない。")
model NotificationTemplatePreviewResponse {
  subject: string;
  body_text: string;
  body_html: string;
  from_display_name?: string;
}


@doc("テスト送信の結果。宛先は操作した管理者自身の確認済みのメールアドレスに固定し、リクエストで指定することはできない。")
model NotificationTemplateTestSendResponse {
  delivered: boolean;
  @doc("メッセージを実際に送ったアドレスで、呼び出し元自身のものである。UI がどこへ送ったかをはっきり示せるように返す。")
  to: string;
}


@doc("テナントの管理者が、通知のテンプレートの上書きを保存した。文面そのものは記録せず、適用するキーとロケールだけを記録する。")
model NotificationTemplateUpdated {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  templateKey: string;
  locale: string;
}


@doc("テナントの管理者が、通知のテンプレートの上書きを削除し、組み込みのデフォルトに戻した。")
model NotificationTemplateReset {
  occurredAt: utcDateTime;
  tenantId: string;
  actorUserId: string;
  templateKey: string;
  locale: string;
}


@doc("管理者が新しいテナントを登録する JSON のリクエスト。`realm` は URL で安全なスラッグで、サーバーが UUID のキー（`id`）を採番する。")
model TenantCreateRequest {
  @minLength(1)
  @maxLength(63)
  @pattern("^[a-z0-9][a-z0-9-]{0,62}$")
  realm: string;
  @minLength(1)
  @maxLength(200)
  display_name: string;
}


@doc("管理者がテナントの表示名またはパスワードポリシーの上書きを更新する JSON のリクエスト。`id` と `status` は変えない。状態の変更は専用の無効化と有効化のエンドポイントで、`default_locale` はサーバーがそれを読むテナント自身の設定のエンドポイントで行う。")
model TenantUpdateRequest {
  @minLength(1)
  @maxLength(200)
  display_name?: string;
  @doc("省くと既存の値を保つ。消去するには明示的に null を渡す。")
  password_policy_override?: PasswordPolicyOverride;
  @doc("委任の最大の深さ。省くと現在の値を保ち、0 はシステムのデフォルトに戻す。システムのデフォルトを超える値は拒否する。")
  @minValue(0)
  max_delegation_depth?: int64;
  @doc("信頼済みデバイスが有効な秒数。省くと既存の値を保ち、0 は機能を再び無効にする。システムの上限の 7,776,000 秒（90 日）を超える値は拒否する。")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
}


@doc("テナントの管理者が自身のテナントの設定を更新する JSON のリクエスト。TenantUpdateRequest に `default_locale` を加えたものである。ロケールはテナント自身の選択なので、`context.tenant_id` に固定したエンドポイントはこれを受け取り、テナントを横断する管理用のエンドポイントは受け取らない。")
model AdminSettingsUpdateRequest {
  @minLength(1)
  @maxLength(200)
  display_name?: string;
  @doc("省くと既存の値を保つ。消去するには明示的に null を渡す。")
  password_policy_override?: PasswordPolicyOverride;
  @doc("委任の最大の深さ。省くと現在の値を保ち、0 はシステムのデフォルトに戻す。システムのデフォルトを超える値は拒否する。")
  @minValue(0)
  max_delegation_depth?: int64;
  @doc("信頼済みデバイスが有効な秒数。省くと既存の値を保ち、0 は機能を再び無効にする。システムの上限の 7,776,000 秒（90 日）を超える値は拒否する。")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
  @doc("テナントのデフォルトのロケール。省くと既存の値を保ち、空文字列はシステムのデフォルトに戻す。対応していないロケールは拒否する。")
  default_locale?: string;
}


@doc("テナントの表現。管理 UI の一覧と詳細の表示は同じ表現を返す。")
model TenantSummaryResponse {
  @doc("テナントの変わらない UUID のキー。")
  id: string;
  @doc("URL `/realms/{realm}/` に現れるスラッグの識別子。")
  realm: string;
  display_name: string;
  status: TenantStatus;
  password_policy_override?: PasswordPolicyOverride;
  @minValue(1)
  max_delegation_depth?: int64;
  @doc("信頼済みデバイスが有効な秒数。0 または省略はこの機能を無効にする。")
  @minValue(0)
  trusted_device_max_age_seconds?: int64;
  default_locale?: string;
  quota: TenantQuota;
  usage: TenantUsage;
  created_at: utcDateTime;
  updated_at?: utcDateTime;
  disabled_at?: utcDateTime;
}


@doc("管理 API のテナントの一覧。")
model TenantListResponse {
  tenants: TenantSummaryResponse[];
}


@doc("system_admin が新しいテナントを作成した。")
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


@doc("system_admin がテナントのリソースのクォータを上書きした。")
model TenantQuotaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("テナントの管理者がカスタム属性の定義（TenantUserAttributeSchema）を更新した。値は保持せず、変更後のカスタム属性のキーの一覧だけを記録する。")
model TenantUserAttributeSchemaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
  attributeKeys: string[];
}


@doc("テナントの管理者が Group のカスタム属性の定義（TenantGroupAttributeSchema）を更新した。値は保持せず、変更後のカスタム属性のキーの一覧だけを記録する。")
model TenantGroupAttributeSchemaUpdated {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
  attributeKeys: string[];
}


@doc("system_admin がテナントを無効にした。そのテナントのすべてのプロトコルの経路を invalid_request で拒否する。")
model TenantDisabled {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("system_admin が、無効にしたテナントを再び有効にした。")
model TenantEnabled {
  occurredAt: utcDateTime;
  actorUserId: string;
  tenantId: string;
}


@doc("テナントの `password_policy_override` が全体の下限のポリシーより弱いか、その `max_age_days` がシステムの上限の外にある。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:policy_override_weaker` である。")
model PolicyOverrideWeakerError is ProblemDetails;


@doc("`user_attribute_schema` の定義が不正である。誤った型、重複するキー、組み込みの属性との衝突などである。RFC 9457 の Problem Details で、`type` はグループのスキーマの拒否と同じコードの `urn:idmagic:error:invalid_attribute_schema` であり、両者はエンドポイントで区別する。")
model InvalidUserAttributeSchemaError is ProblemDetails;


@doc("`group_attribute_schema` の定義が不正である。誤った型、重複するキーなどである。RFC 9457 の Problem Details で、`type` はユーザーのスキーマの拒否と同じコードの `urn:idmagic:error:invalid_attribute_schema` であり、両者はエンドポイントで区別する。")
model InvalidGroupAttributeSchemaError is ProblemDetails;


@doc("リクエストが指定したテナントが存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:tenant_not_found` である。これは id によるテナントの管理用の読み取りであり、未知のホストに対するテナントの解決器自身の拒否は、ルートの仕様の、宣言したステータスコードの節が記述し、どの操作にも属さない。")
model TenantNotFoundError is ProblemDetails;


@doc("リクエストが指定したブランディングの画像が存在しない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:not_found` である。")
model TenantBrandingAssetNotFoundError is ProblemDetails;


@doc("フェデレーションの署名の資格情報が設定されていないので、SAML と WS-Federation のエンドポイントを記述できない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:federation_credentials_unavailable` である。")
model FederationCredentialsUnavailableError is ProblemDetails;


@doc("動的グループの規則が参照する属性は、削除も型の変更もできない。RFC 9457 の Problem Details で、`type` は `urn:idmagic:error:attribute_referenced_by_dynamic_group` である。")
model AttributeReferencedByDynamicGroupError is ProblemDetails;

}
