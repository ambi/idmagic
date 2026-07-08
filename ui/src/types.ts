export type ConsentDetailView = {
  type: string
  description?: string
  summary: string
  lines?: string[]
}

export type AdminUser = {
  id: string
  preferred_username: string
  name?: string
  given_name?: string
  family_name?: string
  email?: string
  email_verified: boolean
  mfa_enrolled: boolean
  roles: string[]
  status?: string
  attributes?: Record<string, AttributeValue>
  required_actions?: string[]
  last_login_at?: string
  password_changed_at?: string
  disabled_at?: string
  // status === 'pending_deletion' のとき、soft-delete 時刻と自動 purge 予定時刻。
  pending_deletion_at?: string
  purge_after?: string
  created_at: string
  updated_at: string
  scim_source?: string
}

export const REQUIRED_ACTIONS = [
  'update_password',
  'verify_email',
  'configure_totp',
  'update_profile',
  'terms_and_conditions',
] as const

export type RequiredActionValue = (typeof REQUIRED_ACTIONS)[number]

export const REQUIRED_ACTION_LABELS: Record<string, string> = {
  update_password: 'パスワードの変更',
  verify_email: 'メールアドレスの確認',
  configure_totp: '二要素認証の設定',
  update_profile: 'プロフィールの更新',
  terms_and_conditions: '利用規約への同意',
}

// requiredActionLabel は内部値を利用者向けの日本語表示名へ変換する。未知の値でも
// 内部表現をそのまま見せず、一般的な文言にフォールバックする。
export function requiredActionLabel(action: string): string {
  return REQUIRED_ACTION_LABELS[action] ?? 'その他の必須対応'
}

export type AdminOAuth2Client = {
  tenant_id: string
  client_id: string
  client_name?: string
  client_type: 'public' | 'confidential'
  redirect_uris: string[]
  grant_types: string[]
  response_types: string[]
  token_endpoint_auth_method:
    | 'client_secret_basic'
    | 'client_secret_post'
    | 'private_key_jwt'
    | 'tls_client_auth'
    | 'none'
  scope: string
  jwks_uri?: string
  jwks?: Record<string, unknown>
  tls_client_auth_subject_dn?: string
  id_token_signed_response_alg: string
  require_pushed_authorization_requests: boolean
  dpop_bound_access_tokens: boolean
  fapi_profile: string
  created_at: string
}

export type ApplicationKind = 'federated' | 'weblink' | 'service'
export type ApplicationStatus = 'active' | 'disabled'
export type ProtocolBindingType = 'oidc' | 'saml' | 'wsfed'

export type ProtocolBinding = {
  type: ProtocolBindingType
  client_id?: string
  wtrealm?: string
}

export type AdminApplication = {
  application_id: string
  name: string
  kind: ApplicationKind
  status: ApplicationStatus
  icon_url?: string
  icon_object_key?: string
  launch_url?: string
  bindings: ProtocolBinding[]
  category_ids: string[]
  category_names: string[]
  binding_summaries: string[]
  assigned_subject_count: number
  sign_in_policy_summary: string
  created_at: string
  updated_at: string
}

export type ApplicationCategory = {
  category_id: string
  name: string
  position: number
  created_at: string
  updated_at: string
}

export type ApplicationAssignment = {
  subject_type: 'user' | 'group'
  subject_id: string
  visibility: 'visible' | 'hidden'
  created_at: string
  updated_at: string
}

export type RequiredAuthnStrength = 'Password' | 'Mfa'

export type RequiredAuthnLevel = {
  strength: RequiredAuthnStrength
}

export type AccessCondition = {
  network_allow_cidrs?: string[]
  reauth_max_age_seconds?: number
}

export type SignInRule = {
  rule_id: string
  name: string
  enabled: boolean
  required_authn: RequiredAuthnLevel
  condition: AccessCondition
}

export type AppSignInPolicy = {
  tenant_id: string
  application_id: string
  rules: SignInRule[]
  created_at: string
  updated_at: string
}

// テナントデフォルトサインインポリシー (wi-115, ADR-081)。例外設定のない全アプリに floor として適用される。
export type TenantDefaultSignInPolicy = {
  tenant_id: string
  rules: SignInRule[]
  created_at: string
  updated_at: string
}

// アプリ詳細で「このアプリの上書き」「テナントデフォルト」「最終的に適用されるポリシー」を区別する (ADR-081)。
// weaker_than_default はアプリ個別ポリシーがデフォルトより弱いときの警告フラグ。
export type AppSignInPolicyView = {
  policy: AppSignInPolicy
  tenant_default: TenantDefaultSignInPolicy
  effective_rules: SignInRule[]
  weaker_than_default: boolean
}

// プロトコル設定はアプリ詳細で解決される。OAuth2 client / WS-Fed RP の実設定を映す。
// advanced 項目を含め、低レベル client 画面を廃してアプリ編集画面に集約する (wi-76)。
// client_type / token_endpoint_auth_method / fapi_profile は更新契約上の不変項目で表示専用。
export type ApplicationOidcConfig = {
  client_id: string
  client_type: 'public' | 'confidential'
  redirect_uris: string[]
  grant_types: string[]
  response_types: string[]
  token_endpoint_auth_method: string
  scope: string
  require_pushed_authorization_requests: boolean
  dpop_bound_access_tokens: boolean
  fapi_profile: string
}

export type ApplicationWsFedConfig = {
  wtrealm: string
  reply_urls: string[]
  audience: string
  token_type: WsFedTokenType
  name_id_format: string
  name_id_source: string
  rules: WsFedClaimMappingRule[]
}

export type ApplicationSamlConfig = {
  entity_id: string
  acs_urls: string[]
  slo_url: string
  audience: string
  name_id_format: string
  name_id_source: string
  sign_assertion: boolean
  sign_response: boolean
  want_authn_requests_signed: boolean
  authn_request_signing_certificate_pem: string
  rules: WsFedClaimMappingRule[]
}

export type AdminApplicationDetail = {
  application: AdminApplication
  oidc?: ApplicationOidcConfig | null
  wsfed?: ApplicationWsFedConfig | null
  saml?: ApplicationSamlConfig | null
  sign_in_policy?: AppSignInPolicyView | null
}

export type AuthorizationDetailFieldRule = {
  name: string
  semantics: 'set' | 'at_most' | 'enum' | 'exact'
  required?: boolean
  allowed?: string[]
}

export type AuthorizationDetailType = {
  tenant_id: string
  type: string
  description?: string
  schema: { rules: AuthorizationDetailFieldRule[] }
  display_template: string
  state: 'Enabled' | 'Disabled'
  created_at: string
  updated_at: string
}

export type WsFedClaimMappingRule = {
  claim_type: string
  source: 'user_attribute' | 'fixed' | 'nameid'
  source_key?: string
  fixed_value?: string
  required?: boolean
}

export type WsFedNameIdConfiguration = {
  format: string
  source_attribute: string
}

export type WsFedClaimMappingPolicy = {
  name_id: WsFedNameIdConfiguration
  rules?: WsFedClaimMappingRule[]
}

export type EntraFederationProfile = {
  domain: string
  issuer_uri: string
  source_anchor_attribute: string
  immutable_id_attribute: string
  passive_logon_uri?: string
  active_logon_uri?: string
  metadata_exchange_uri?: string
}

export type WsFedTokenType =
  | 'urn:oasis:names:tc:SAML:1.0:assertion'
  | 'urn:oasis:names:tc:SAML:2.0:assertion'

export type WsFedRelyingParty = {
  tenant_id: string
  wtrealm: string
  display_name?: string
  reply_urls: string[]
  audience?: string
  token_type?: WsFedTokenType
  claim_policy: WsFedClaimMappingPolicy
  entra_profile?: EntraFederationProfile
  created_at: string
  updated_at?: string
}

export type AdminConsent = {
  user_id: string
  preferred_username?: string
  client_id: string
  client_name: string
  scopes: string[]
  state: 'granted' | 'revoked' | 'expired'
  granted_at: string
  expires_at: string
  revoked_at?: string
}

export type AdminAuditEvent = {
  id: string
  tenant_id: string
  type: string
  occurred_at: string
  payload: Record<string, unknown>
}

export type AdminKey = {
  kid: string
  alg: string
  provider: string
  active: boolean
  created_at: string
  public_jwk: Record<string, unknown>
}

export type TenantKeyHealth = {
  tenant_id: string
  provider: string
  usage: string
  active_kid: string
  jwks_key_count: number
  provider_healthy: boolean
}

export type AdminGroup = {
  id: string
  tenant_id: string
  name: string
  description?: string
  roles: string[]
  member_count: number
  created_at: string
  updated_at?: string
  scim_source?: string
}

export type AdminGroupMember = {
  user_id: string
  preferred_username: string
  created_at: string
}

export type AdminUserGroups = {
  groups: AdminGroup[]
  direct_roles: string[]
  group_roles: string[]
  effective_roles: string[]
}

export type AdminAgent = {
  id: string
  tenant_id: string
  name: string
  description?: string
  kind: 'autonomous' | 'supervised'
  owner_user_id: string
  status: 'active' | 'disabled' | 'killed'
  roles: string[]
  client_ids: string[]
  created_at: string
  updated_at?: string
  disabled_at?: string
  killed_at?: string
}

export type AdminTenant = {
  id: string
  realm: string
  display_name: string
  status: 'active' | 'disabled'
  password_policy_override?: {
    min_length?: number
    max_length?: number
    history_depth?: number
  }
  created_at: string
  updated_at?: string
  disabled_at?: string
}

export type AdminRoleInterface = {
  name: string
  method: string
  path: string
}

export type AdminRolePermission = {
  name: string
  action: string
  description: string
  interfaces: AdminRoleInterface[]
}

export type AdminRole = {
  name: string
  description: string
  aliases: string[]
  permissions: AdminRolePermission[]
}

export type AdminSettings = {
  tenant_id: string
  realm: string
  display_name: string
  password_policy_override?: {
    min_length?: number
    max_length?: number
    history_depth?: number
  }
  password_policy_defaults: {
    min_length: number
    max_length: number
    history_depth: number
  }
}

export type AttributeType = 'string' | 'number' | 'boolean' | 'date' | 'string_array'

export type AttrVisibility = 'private' | 'self_readable' | 'admin_readable' | 'claim_exposed'

export type UserAttributeDef = {
  key: string
  label?: string
  type: AttributeType
  multi_valued: boolean
  required: boolean
  editable_by_user: boolean
  claim_name?: string
  oidc_scope?: string
  visibility: AttrVisibility
  pii: boolean
}

export type AttributeValue = {
  type: AttributeType
  string?: string
  number?: number
  boolean?: boolean
  date?: string
  string_array?: string[]
}

export type TenantUserAttributeSchema = {
  tenant_id: string
  attributes: UserAttributeDef[]
  builtin: UserAttributeDef[]
  created_at: string
  updated_at: string
}

export type AccountProfile = {
  sub: string
  preferred_username: string
  name?: string
  given_name?: string
  family_name?: string
  email?: string
  email_verified: boolean
  mfa_enrolled: boolean
  status: string
  attributes: Record<string, AttributeValue>
  readable_attributes: UserAttributeDef[]
  editable_attributes: UserAttributeDef[]
}

export type AccountSummary = {
  sub: string
  preferred_username: string
  name?: string
  email?: string
  email_verified: boolean
  mfa_enrolled: boolean
  status: string
  last_login_at?: string
  password_changed_at?: string
  required_actions: string[]
}

export type AccountConsent = {
  client_id: string
  client_name: string
  scopes: string[]
  state: string
  granted_at: string
  expires_at: string
}

export type MyApplication = {
  application_id: string
  name: string
  kind: ApplicationKind
  icon_url?: string
  launch_url?: string
  category_ids: string[]
}

export type PortalCategory = {
  category_id: string
  name: string
}

export type AccountMfaFactor = {
  type: string
  label?: string
  created_at: string
  last_used_at?: string
}

export type WebAuthnCredentialSummary = {
  credential_id: string
  label?: string
  transports: string[]
  created_at: string
  last_used_at?: string
}

export type RecoveryCodeStatus = {
  generated_at?: string
  total: number
  remaining: number
}

export type AccountSecurity = {
  password_changed_at?: string
  totp_enrolled: boolean
  factors: AccountMfaFactor[]
  webauthn_credentials: WebAuthnCredentialSummary[]
  recovery_codes: RecoveryCodeStatus
}

export type TotpEnrollmentStart = {
  secret: string
  otpauth_uri: string
  account_name: string
  issuer: string
}

export type AccountSignInActivity = {
  occurred_at: string
  amr: string[]
}

export type AccountSession = {
  id: string
  current: boolean
  amr: string[]
  acr: string
  started_at: string
  expires_at: string
}

export type BrowserFlowResponse = {
  next?: string
  redirect_to?: string
}

export type ScimToken = {
  id: string
  description?: string
  created_at: string
  expires_at?: string
}
