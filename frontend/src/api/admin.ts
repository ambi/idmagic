import type {
  AdminAgent,
  AdminApplication,
  AdminApplicationDetail,
  AdminAuditEvent,
  ApplicationCategory,
  AdminConsent,
  AdminGroup,
  AdminGroupMember,
  AdminKey,
  AdminSessionRecord,
  AdminSettings,
  ApiToken,
  ApiTokenScope,
  TenantKeyHealth,
  AdminTenant,
  TenantQuota,
  AdminUser,
  AdminUserGroups,
  ApplicationAssignment,
  ApplicationStatus,
  AppSignInPolicyView,
  TenantDefaultSignInPolicy,
  TenantDefaultSignInPolicyView,
  SignInRule,
  AuthorizationDetailType,
  McpResourceServer,
  TenantUserAttributeSchema,
  UserAttributeDef,
  EntraFederationProfile,
  UserImportJob,
  UserImportJobSummary,
  UserImportMode,
  WsFedClaimMappingRule,
  WsFedRelyingParty,
  WsFedTokenType,
  AdminLifecycleWorkflow,
  WorkflowAction,
  WorkflowTrigger,
  WorkflowRun,
  AttributeMappingRule,
  DeprovisionPolicy,
  GroupPushConfig,
  MatchingRule,
  ProvisioningAuthMethod,
  ProvisioningConnection,
  ProvisioningConnectionStatus,
  ProvisioningDelivery,
  ProvisioningDeliveryStatus,
  ProvisioningFeatureFlags,
  ProvisioningScope,
  ProvisioningSourceType,
  ProvisioningTestConnectionResult,
} from '../types'
import { AuthenticationAPIError, adminRequest, request, tenantURL } from './core'

type AdminUserListResponse = { users: AdminUser[] }
type AdminConsentListResponse = { consents: AdminConsent[] }
type AdminAuditEventListResponse = { events: AdminAuditEvent[] }
type AdminKeyListResponse = { keys: AdminKey[] }
type TenantKeyHealthListResponse = { tenants: TenantKeyHealth[] }
export type AdminRotateKeyResponse = { next: AdminKey; previous?: AdminKey }
type AdminTenantListResponse = { tenants: AdminTenant[] }

export type CreateAdminUserInput = {
  preferred_username: string
  password: string
  name?: string
  email?: string
  email_verified: boolean
  roles: string[]
}

export async function listAdminUsers(): Promise<AdminUser[]> {
  return (await request<AdminUserListResponse>('/api/admin/users')).users
}

export async function getAdminUser(id: string): Promise<AdminUser> {
  return request<AdminUser>(`/api/admin/users/${encodeURIComponent(id)}`)
}

export async function createAdminUser(
  csrfToken: string,
  input: CreateAdminUserInput,
): Promise<AdminUser> {
  return request('/api/admin/users', adminRequest(csrfToken, 'POST', input))
}

export type UpdateAdminUserInput = {
  preferred_username?: string
  name?: string
  given_name?: string
  family_name?: string
  email?: string
  email_verified?: boolean
  roles?: string[]
  attributes?: AdminUser['attributes']
}

export async function updateAdminUser(
  csrfToken: string,
  id: string,
  input: UpdateAdminUserInput,
): Promise<AdminUser> {
  return request(
    `/api/admin/users/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function setAdminUserRequiredAction(
  csrfToken: string,
  id: string,
  action: string,
): Promise<AdminUser> {
  return request(
    `/api/admin/users/${encodeURIComponent(id)}/required_actions`,
    adminRequest(csrfToken, 'POST', { action }),
  )
}

export async function clearAdminUserRequiredAction(
  csrfToken: string,
  id: string,
  action: string,
): Promise<AdminUser> {
  return request(
    `/api/admin/users/${encodeURIComponent(id)}/required_actions/${encodeURIComponent(action)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function updateAdminTenantQuota(
  csrfToken: string,
  id: string,
  input: TenantQuota,
): Promise<TenantQuota> {
  return request(
    `/api/admin/tenants/${encodeURIComponent(id)}/quota`,
    adminRequest(csrfToken, 'PUT', input),
  )
}

export async function setAdminUserDisabled(
  csrfToken: string,
  id: string,
  disabled: boolean,
): Promise<void> {
  await request(
    `/api/admin/users/${encodeURIComponent(id)}/${disabled ? 'disable' : 'enable'}`,
    adminRequest(csrfToken, 'POST'),
  )
}

// deleteAdminUser は既定で soft-delete (削除予約) する。purge=true のとき
// ?purge=true を付けて完全削除 (匿名化) に切り替える。
export async function deleteAdminUser(
  csrfToken: string,
  id: string,
  options?: { purge?: boolean },
): Promise<void> {
  const query = options?.purge ? '?purge=true' : ''
  await request(
    `/api/admin/users/${encodeURIComponent(id)}${query}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// restoreAdminUser は削除予約中 (pending_deletion) のユーザーを復元する。
export async function restoreAdminUser(csrfToken: string, id: string): Promise<AdminUser> {
  return request(
    `/api/admin/users/${encodeURIComponent(id)}/restore`,
    adminRequest(csrfToken, 'POST'),
  )
}

// importAdminUsers は CSV を dry_run (検証のみ) または apply (作成) ジョブとして投入する。
// 202 応答はジョブ受理のみを表し、結果は getAdminUserImport の polling で取得する。
export async function importAdminUsers(
  csrfToken: string,
  input: { csv: string; mode: UserImportMode },
): Promise<UserImportJobSummary> {
  return request('/api/admin/users/imports', adminRequest(csrfToken, 'POST', input))
}

export async function getAdminUserImport(jobId: string): Promise<UserImportJob> {
  return request(`/api/admin/users/imports/${encodeURIComponent(jobId)}`)
}

export type LifecycleWorkflowInput = {
  expected_revision?: number
  name: string
  description?: string
  trigger: WorkflowTrigger
  actions: WorkflowAction[]
}
export async function listLifecycleWorkflows(): Promise<AdminLifecycleWorkflow[]> {
  return (await request<{ workflows: AdminLifecycleWorkflow[] }>('/api/admin/lifecycle_workflows'))
    .workflows
}
export async function getLifecycleWorkflow(id: string): Promise<AdminLifecycleWorkflow> {
  return request(`/api/admin/lifecycle_workflows/${encodeURIComponent(id)}`)
}
export async function createLifecycleWorkflow(
  csrfToken: string,
  input: LifecycleWorkflowInput,
): Promise<AdminLifecycleWorkflow> {
  return request('/api/admin/lifecycle_workflows', adminRequest(csrfToken, 'POST', input))
}
export async function updateLifecycleWorkflow(
  csrfToken: string,
  id: string,
  input: LifecycleWorkflowInput,
): Promise<AdminLifecycleWorkflow> {
  return request(
    `/api/admin/lifecycle_workflows/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'PUT', input),
  )
}
export async function setLifecycleWorkflowState(
  csrfToken: string,
  id: string,
  state: 'enable' | 'disable',
  expected_revision: number,
): Promise<AdminLifecycleWorkflow> {
  return request(
    `/api/admin/lifecycle_workflows/${encodeURIComponent(id)}/${state}`,
    adminRequest(csrfToken, 'POST', { expected_revision }),
  )
}
export async function deleteLifecycleWorkflow(
  csrfToken: string,
  id: string,
  expected_revision: number,
): Promise<void> {
  await request(
    `/api/admin/lifecycle_workflows/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'DELETE', { expected_revision }),
  )
}
export async function dryRunLifecycleWorkflow(
  csrfToken: string,
  id: string,
  targetUserID: string,
): Promise<{ steps: { action_kind: string; would_change: string; reason?: string }[] }> {
  return request(
    `/api/admin/lifecycle_workflows/${encodeURIComponent(id)}/dry_run`,
    adminRequest(csrfToken, 'POST', { target_user_id: targetUserID }),
  )
}
export async function listLifecycleWorkflowRuns(id: string): Promise<WorkflowRun[]> {
  return (
    await request<{ runs: WorkflowRun[] }>(
      `/api/admin/lifecycle_workflows/${encodeURIComponent(id)}/runs`,
    )
  ).runs
}
export async function retryLifecycleWorkflowRun(
  csrfToken: string,
  id: string,
): Promise<WorkflowRun> {
  return request(
    `/api/admin/lifecycle_workflow_runs/${encodeURIComponent(id)}/retry`,
    adminRequest(csrfToken, 'POST'),
  )
}

// authorization_details type (RFC 9396 / ADR-050) の管理 API クライアント。
export type AuthorizationDetailTypeInput = {
  type?: string
  description?: string
  display_template: string
  state?: AuthorizationDetailType['state']
  schema: AuthorizationDetailType['schema']
}

export async function listAuthorizationDetailTypes(): Promise<AuthorizationDetailType[]> {
  return (
    await request<{ types: AuthorizationDetailType[] }>('/api/admin/authorization-detail-types')
  ).types
}

export async function createAuthorizationDetailType(
  csrfToken: string,
  input: AuthorizationDetailTypeInput,
): Promise<AuthorizationDetailType> {
  return request('/api/admin/authorization-detail-types', adminRequest(csrfToken, 'POST', input))
}

export async function updateAuthorizationDetailType(
  csrfToken: string,
  detailType: string,
  input: AuthorizationDetailTypeInput,
): Promise<AuthorizationDetailType> {
  return request(
    `/api/admin/authorization-detail-types/${encodeURIComponent(detailType)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function deleteAuthorizationDetailType(
  csrfToken: string,
  detailType: string,
): Promise<void> {
  await request(
    `/api/admin/authorization-detail-types/${encodeURIComponent(detailType)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// MCP resource server (RFC 9728 / RFC 8707) の管理 API クライアント。
export type McpResourceServerInput = {
  resource?: string
  name: string
  scopes: string[]
  state?: McpResourceServer['state']
}

export async function listMcpResourceServers(): Promise<McpResourceServer[]> {
  return (
    await request<{ resource_servers: McpResourceServer[] }>('/api/admin/mcp-resource-servers')
  ).resource_servers
}

export async function createMcpResourceServer(
  csrfToken: string,
  input: McpResourceServerInput,
): Promise<McpResourceServer> {
  return request('/api/admin/mcp-resource-servers', adminRequest(csrfToken, 'POST', input))
}

export async function updateMcpResourceServer(
  csrfToken: string,
  resourceServerID: string,
  input: McpResourceServerInput,
): Promise<McpResourceServer> {
  return request(
    `/api/admin/mcp-resource-servers/${encodeURIComponent(resourceServerID)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function deleteMcpResourceServer(
  csrfToken: string,
  resourceServerID: string,
): Promise<void> {
  await request(
    `/api/admin/mcp-resource-servers/${encodeURIComponent(resourceServerID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

type WsFedRelyingPartyListResponse = { relying_parties: WsFedRelyingParty[] | null }

export async function listWsFedRelyingParties(): Promise<WsFedRelyingParty[]> {
  const response = await request<WsFedRelyingPartyListResponse>('/api/admin/wsfed/relying-parties')
  return response.relying_parties ?? []
}

export async function deleteWsFedRelyingParty(csrfToken: string, wtrealm: string): Promise<void> {
  await request(
    `/api/admin/wsfed/relying-parties?wtrealm=${encodeURIComponent(wtrealm)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export type ConfigureEntraFederationInput = {
  domain: string
  issuer_uri?: string
  source_anchor_attribute: string
  reply_url?: string
}

export type ConfigureEntraFederationResponse = {
  profile: EntraFederationProfile
  relying_party: WsFedRelyingParty
  powershell: Record<string, string>
  known_limitations: string[]
}

export async function configureEntraFederation(
  csrfToken: string,
  input: ConfigureEntraFederationInput,
): Promise<ConfigureEntraFederationResponse> {
  return request('/api/admin/wsfed/entra-federation', adminRequest(csrfToken, 'POST', input))
}

export async function listAdminConsents(): Promise<AdminConsent[]> {
  return (await request<AdminConsentListResponse>('/api/admin/consents')).consents
}

export async function revokeAdminConsent(
  csrfToken: string,
  userID: string,
  clientID: string,
): Promise<void> {
  await request(
    `/api/admin/consents/${encodeURIComponent(userID)}/${encodeURIComponent(clientID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// イベントカテゴリ (wi-44 統合)。認証サブ分類 + 管理操作カテゴリ。
export type AdminAuditEventCategory =
  | 'authentication'
  | 'success'
  | 'fail'
  | 'aggregated'
  | 'user'
  | 'group'
  | 'client'
  | 'consent'
  | 'token'
  | 'tenant'
  | 'key'

export type AdminAuditEventQuery = {
  // type 完全一致 (機械向け低レベルフィルタ)。UI には出さない。
  type?: string
  category?: AdminAuditEventCategory
  sub?: string
  // username (wi-147): 実アカウントが常に確定するイベントの検索用。サーバ側で user_id に
  // 解決してから絞り込む (該当なしは 0 件)。
  username?: string
  after?: string
  before?: string
  limit?: number
  allTenants?: boolean
  filter?: string[]
}

// 監査イベント検索フォームが URL query string と同期する部分 (wi-147)。type は
// 機械向け低レベルフィルタで UI からは設定しないため除く。
export type AdminAuditEventsSearchParams = Omit<AdminAuditEventQuery, 'type'>

function auditEventParams(query: AdminAuditEventQuery): URLSearchParams {
  const params = new URLSearchParams()
  if (query.type) params.set('type', query.type)
  if (query.category) params.set('category', query.category)
  if (query.sub) params.set('user_id', query.sub)
  if (query.username) params.set('username', query.username)
  if (query.after) params.set('after', query.after)
  if (query.before) params.set('before', query.before)
  if (query.limit !== undefined) params.set('limit', String(query.limit))
  if (query.allTenants) params.set('all_tenants', 'true')
  for (const filter of query.filter ?? []) {
    if (filter) params.append('filter', filter)
  }
  return params
}

export async function listAdminAuditEvents(
  query: AdminAuditEventQuery,
): Promise<AdminAuditEvent[]> {
  const params = auditEventParams(query)
  const url =
    params.size > 0 ? `/api/admin/audit_events?${params.toString()}` : '/api/admin/audit_events'
  return (await request<AdminAuditEventListResponse>(url)).events
}

// 監査イベントのエクスポート URL (認証イベント含む)。新規タブで開いてダウンロードする。
export function adminAuditEventsExportURL(query: AdminAuditEventQuery): string {
  const params = auditEventParams(query)
  return tenantURL(`/api/admin/audit_events/export?${params.toString()}`)
}

// event.type / outcome を選択式にするための選択肢一覧 (wi-147)。UI 側でハードコードせず、
// Go 側の単一の正 (auditEventCategoryTypes / eventOutcome) から機械的に取得する。
export type AdminAuditEventSearchOptions = {
  event_types: string[]
  outcomes: string[]
}

export async function listAdminAuditEventSearchOptions(): Promise<AdminAuditEventSearchOptions> {
  return request<AdminAuditEventSearchOptions>('/api/admin/audit_events/search_options')
}

export async function listAdminKeys(): Promise<AdminKey[]> {
  return (await request<AdminKeyListResponse>('/api/admin/keys')).keys
}

export async function rotateTenantSigningKey(csrfToken: string): Promise<AdminRotateKeyResponse> {
  return request<AdminRotateKeyResponse>('/api/admin/keys/rotate', adminRequest(csrfToken, 'POST'))
}

export async function disableTenantKey(csrfToken: string, kid: string): Promise<AdminKey> {
  return request<AdminKey>(
    `/api/admin/keys/${encodeURIComponent(kid)}/disable`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function listTenantKeyHealth(): Promise<TenantKeyHealth[]> {
  return (await request<TenantKeyHealthListResponse>('/api/admin/keys/health')).tenants
}

export type UpdateAdminSettingsInput = {
  display_name?: string
  password_policy_override?: AdminSettings['password_policy_override']
}

export async function getAdminSettings(): Promise<AdminSettings> {
  return request<AdminSettings>('/api/admin/settings')
}

export async function updateAdminSettings(
  csrfToken: string,
  input: UpdateAdminSettingsInput,
): Promise<AdminSettings> {
  return request('/api/admin/settings', adminRequest(csrfToken, 'PATCH', input))
}
export async function getTenantUserAttributeSchema(): Promise<TenantUserAttributeSchema> {
  return request<TenantUserAttributeSchema>('/api/admin/tenant/user_attribute_schema')
}

export async function updateTenantUserAttributeSchema(
  csrfToken: string,
  attributes: UserAttributeDef[],
): Promise<TenantUserAttributeSchema> {
  return request(
    '/api/admin/tenant/user_attribute_schema',
    adminRequest(csrfToken, 'PUT', { attributes }),
  )
}

export async function listAdminTenants(): Promise<AdminTenant[]> {
  return (await request<AdminTenantListResponse>('/api/admin/tenants')).tenants
}

export type CreateAdminTenantInput = {
  realm: string
  display_name: string
}

export type UpdateAdminTenantInput = {
  display_name?: string
  password_policy_override?: AdminTenant['password_policy_override']
}

export async function createAdminTenant(
  csrfToken: string,
  input: CreateAdminTenantInput,
): Promise<AdminTenant> {
  return request('/api/admin/tenants', adminRequest(csrfToken, 'POST', input))
}

export async function updateAdminTenant(
  csrfToken: string,
  tenantID: string,
  input: UpdateAdminTenantInput,
): Promise<AdminTenant> {
  return request(
    `/api/admin/tenants/${encodeURIComponent(tenantID)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function setAdminTenantDisabled(
  csrfToken: string,
  tenantID: string,
  disabled: boolean,
): Promise<void> {
  await request(
    `/api/admin/tenants/${encodeURIComponent(tenantID)}/${disabled ? 'disable' : 'enable'}`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function listAdminGroups(): Promise<AdminGroup[]> {
  return (await request<{ groups: AdminGroup[] }>('/api/admin/groups')).groups
}

export async function getAdminGroup(
  id: string,
): Promise<{ group: AdminGroup; members: AdminGroupMember[] }> {
  return request(`/api/admin/groups/${encodeURIComponent(id)}`)
}

export type CreateAdminGroupInput = {
  name: string
  description?: string
  roles?: string[]
  membership_type?: AdminGroup['membership_type']
  dynamic_rule?: { expression: string }
}

export async function updateDynamicGroupRule(csrfToken: string, id: string, expression: string) {
  return request<NonNullable<AdminGroup['dynamic_rule']>>(
    `/api/admin/groups/${encodeURIComponent(id)}/dynamic-rule`,
    adminRequest(csrfToken, 'PUT', { expression }),
  )
}

export async function previewDynamicGroupRule(
  csrfToken: string,
  id: string,
  expression: string,
  userIDs: string[],
) {
  return request<{ results: import('../types').DynamicGroupPreview[] }>(
    `/api/admin/groups/${encodeURIComponent(id)}/dynamic-rule/preview`,
    adminRequest(csrfToken, 'POST', { expression, user_ids: userIDs }),
  )
}

export async function setDynamicGroupRuleEnabled(csrfToken: string, id: string, enabled: boolean) {
  return request<NonNullable<AdminGroup['dynamic_rule']>>(
    `/api/admin/groups/${encodeURIComponent(id)}/dynamic-rule/${enabled ? 'enable' : 'disable'}`,
    adminRequest(csrfToken, 'POST'),
  )
}

export type UpdateAdminGroupInput = {
  name?: string
  description?: string
  roles?: string[]
}

export async function createAdminGroup(
  csrfToken: string,
  input: CreateAdminGroupInput,
): Promise<AdminGroup> {
  return request('/api/admin/groups', adminRequest(csrfToken, 'POST', input))
}

export async function updateAdminGroup(
  csrfToken: string,
  id: string,
  input: UpdateAdminGroupInput,
): Promise<AdminGroup> {
  return request(
    `/api/admin/groups/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function deleteAdminGroup(csrfToken: string, id: string): Promise<void> {
  await request(`/api/admin/groups/${encodeURIComponent(id)}`, adminRequest(csrfToken, 'DELETE'))
}

export async function addAdminGroupMember(
  csrfToken: string,
  groupID: string,
  userID: string,
): Promise<void> {
  await request(
    `/api/admin/groups/${encodeURIComponent(groupID)}/members/${encodeURIComponent(userID)}`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function removeAdminGroupMember(
  csrfToken: string,
  groupID: string,
  userID: string,
): Promise<void> {
  await request(
    `/api/admin/groups/${encodeURIComponent(groupID)}/members/${encodeURIComponent(userID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function getAdminUserGroups(id: string): Promise<AdminUserGroups> {
  return request(`/api/admin/users/${encodeURIComponent(id)}/groups`)
}

export async function listAdminAgents(): Promise<AdminAgent[]> {
  return (await request<{ agents: AdminAgent[] }>('/api/admin/agents')).agents
}

export async function getAdminAgent(id: string): Promise<AdminAgent> {
  return request<AdminAgent>(`/api/admin/agents/${encodeURIComponent(id)}`)
}

export type RegisterAdminAgentInput = {
  name: string
  description?: string
  kind?: AdminAgent['kind']
  owner_user_id?: string
  roles?: string[]
}

export type UpdateAdminAgentInput = {
  name?: string
  description?: string
  kind?: AdminAgent['kind']
  owner_user_id?: string
  roles?: string[]
}

export async function registerAdminAgent(
  csrfToken: string,
  input: RegisterAdminAgentInput,
): Promise<AdminAgent> {
  return request('/api/admin/agents', adminRequest(csrfToken, 'POST', input))
}

export async function updateAdminAgent(
  csrfToken: string,
  id: string,
  input: UpdateAdminAgentInput,
): Promise<AdminAgent> {
  return request(
    `/api/admin/agents/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function disableAdminAgent(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/agents/${encodeURIComponent(id)}/disable`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function enableAdminAgent(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/agents/${encodeURIComponent(id)}/enable`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function killAdminAgent(csrfToken: string, id: string): Promise<void> {
  await request(`/api/admin/agents/${encodeURIComponent(id)}/kill`, adminRequest(csrfToken, 'POST'))
}

export async function deleteAdminAgent(csrfToken: string, id: string): Promise<void> {
  await request(`/api/admin/agents/${encodeURIComponent(id)}`, adminRequest(csrfToken, 'DELETE'))
}

export async function bindAdminAgentCredential(
  csrfToken: string,
  agentID: string,
  clientID: string,
): Promise<void> {
  await request(
    `/api/admin/agents/${encodeURIComponent(agentID)}/credentials`,
    adminRequest(csrfToken, 'POST', { client_id: clientID }),
  )
}

export async function unbindAdminAgentCredential(
  csrfToken: string,
  agentID: string,
  clientID: string,
): Promise<void> {
  await request(
    `/api/admin/agents/${encodeURIComponent(agentID)}/credentials/${encodeURIComponent(clientID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// Application カタログ (wi-69)。種別を選びプロトコル設定もまとめて入力する一括作成 API。
// backend が OAuth2 client / WS-Fed RP を作成し、Application と binding を一括で作る。
export type CreateAdminApplicationInput = {
  name: string
  type: 'oidc' | 'wsfed' | 'saml' | 'weblink' | 'service'
  launch_url?: string
  // OIDC
  redirect_uris?: string[]
  // OIDC / service の生成 client 設定。認証方式は作成時に確定し以後不変。
  scope?: string
  client_type?: 'public' | 'confidential'
  token_endpoint_auth_method?: string
  jwks_uri?: string
  tls_client_auth_subject_dn?: string
  // WS-Federation
  wtrealm?: string
  reply_urls?: string[]
  name_id_format?: string
  name_id_source?: string
  // SAML 2.0
  entity_id?: string
  acs_urls?: string[]
  slo_url?: string
  sign_response?: boolean
  want_authn_requests_signed?: boolean
  authn_request_signing_certificate_pem?: string
}

// OIDC を一括作成すると client_secret が一度だけ返る (再表示不可)。
export type CreateAdminApplicationResult = {
  application: AdminApplication
  client_id?: string
  client_secret?: string
}

export type UpdateAdminApplicationInput = {
  name?: string
  status?: ApplicationStatus
  launch_url?: string
}

export type UpdateApplicationOidcInput = {
  redirect_uris?: string[]
  grant_types?: string[]
  response_types?: string[]
  scope?: string
  require_pushed_authorization_requests?: boolean
  dpop_bound_access_tokens?: boolean
}

export type UpdateApplicationWsFedInput = {
  reply_urls?: string[]
  audience?: string
  token_type?: WsFedTokenType
  name_id_format?: string
  name_id_source?: string
  rules?: WsFedClaimMappingRule[]
}

export type UpdateApplicationSamlInput = {
  acs_urls?: string[]
  slo_url?: string
  audience?: string
  name_id_format?: string
  name_id_source?: string
  sign_assertion?: boolean
  sign_response?: boolean
  want_authn_requests_signed?: boolean
  authn_request_signing_certificate_pem?: string
  rules?: WsFedClaimMappingRule[]
}

export async function listAdminApplications(): Promise<AdminApplication[]> {
  return (await request<{ applications: AdminApplication[] }>('/api/admin/applications'))
    .applications
}

export async function getAdminApplication(id: string): Promise<AdminApplicationDetail> {
  return request<AdminApplicationDetail>(`/api/admin/applications/${encodeURIComponent(id)}`)
}

export async function createAdminApplication(
  csrfToken: string,
  input: CreateAdminApplicationInput,
): Promise<CreateAdminApplicationResult> {
  return request('/api/admin/applications', adminRequest(csrfToken, 'POST', input))
}

export async function updateApplicationOidcConfig(
  csrfToken: string,
  id: string,
  input: UpdateApplicationOidcInput,
): Promise<void> {
  await request(
    `/api/admin/applications/${encodeURIComponent(id)}/oidc`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function rotateApplicationClientSecret(
  csrfToken: string,
  id: string,
  graceDays: number,
): Promise<{ client_secret: string; grace_until?: string }> {
  return request(
    `/api/admin/applications/${encodeURIComponent(id)}/oidc/rotate-secret`,
    adminRequest(csrfToken, 'POST', { grace_days: graceDays }),
  )
}

export async function updateApplicationWsFedConfig(
  csrfToken: string,
  id: string,
  input: UpdateApplicationWsFedInput,
): Promise<void> {
  await request(
    `/api/admin/applications/${encodeURIComponent(id)}/wsfed`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function updateApplicationSamlConfig(
  csrfToken: string,
  id: string,
  input: UpdateApplicationSamlInput,
): Promise<void> {
  await request(
    `/api/admin/applications/${encodeURIComponent(id)}/saml`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function updateAdminApplication(
  csrfToken: string,
  id: string,
  input: UpdateAdminApplicationInput,
): Promise<AdminApplication> {
  return request(
    `/api/admin/applications/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function uploadApplicationIcon(
  csrfToken: string,
  id: string,
  file: File,
): Promise<AdminApplication> {
  const form = new FormData()
  form.set('file', file)
  const response = await fetch(
    tenantURL(`/api/admin/applications/${encodeURIComponent(id)}/icon`),
    {
      method: 'POST',
      credentials: 'same-origin',
      cache: 'no-store',
      headers: { 'X-CSRF-Token': csrfToken },
      body: form,
    },
  )
  const body = (await response.json().catch(() => ({}))) as {
    application?: AdminApplication
    error?: string
    message?: string
    error_description?: string
  }
  if (!response.ok) {
    throw new AuthenticationAPIError(
      body.message ?? body.error_description ?? 'Could not upload the icon.',
      body.error,
    )
  }
  if (!body.application) {
    throw new AuthenticationAPIError('Could not upload the icon.')
  }
  return body.application
}

export async function deleteApplicationIcon(
  csrfToken: string,
  id: string,
): Promise<AdminApplication> {
  return (
    await request<{ application: AdminApplication }>(
      `/api/admin/applications/${encodeURIComponent(id)}/icon`,
      adminRequest(csrfToken, 'DELETE'),
    )
  ).application
}

export async function deleteAdminApplication(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/applications/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function listApplicationAssignments(id: string): Promise<ApplicationAssignment[]> {
  return (
    await request<{ assignments: ApplicationAssignment[] }>(
      `/api/admin/applications/${encodeURIComponent(id)}/assignments`,
    )
  ).assignments
}

export type AssignApplicationInput = {
  subject_type: 'user' | 'group'
  subject_id: string
  visibility?: 'visible' | 'hidden'
}

export async function assignApplication(
  csrfToken: string,
  id: string,
  input: AssignApplicationInput,
): Promise<ApplicationAssignment> {
  return request(
    `/api/admin/applications/${encodeURIComponent(id)}/assignments`,
    adminRequest(csrfToken, 'POST', input),
  )
}

export async function unassignApplication(
  csrfToken: string,
  id: string,
  subjectType: 'user' | 'group',
  subjectID: string,
): Promise<void> {
  await request(
    `/api/admin/applications/${encodeURIComponent(id)}/assignments/${encodeURIComponent(subjectType)}/${encodeURIComponent(subjectID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function getAppSignInPolicy(id: string): Promise<AppSignInPolicyView> {
  return await request<AppSignInPolicyView>(
    `/api/admin/applications/${encodeURIComponent(id)}/sign-in-policy`,
  )
}

export async function updateAppSignInPolicy(
  csrfToken: string,
  id: string,
  rules: SignInRule[],
): Promise<AppSignInPolicyView> {
  return await request<AppSignInPolicyView>(
    `/api/admin/applications/${encodeURIComponent(id)}/sign-in-policy`,
    adminRequest(csrfToken, 'PUT', { rules }),
  )
}

// テナントデフォルトサインインポリシー (wi-115, ADR-081)。
export async function getTenantDefaultSignInPolicy(): Promise<TenantDefaultSignInPolicyView> {
  return request<TenantDefaultSignInPolicyView>('/api/admin/default-sign-in-policy')
}

export async function updateTenantDefaultSignInPolicy(
  csrfToken: string,
  rules: SignInRule[],
): Promise<TenantDefaultSignInPolicy> {
  return (
    await request<{ policy: TenantDefaultSignInPolicy }>(
      '/api/admin/default-sign-in-policy',
      adminRequest(csrfToken, 'PUT', { rules }),
    )
  ).policy
}

export type MfaEnrollmentBypass = {
  id: string
  tenant_id: string
  user_id: string
  issued_at: string
  expires_at: string
}

export async function issueMfaEnrollmentBypass(
  csrfToken: string,
  userID: string,
): Promise<MfaEnrollmentBypass> {
  return (
    await request<{ bypass: MfaEnrollmentBypass }>(
      `/api/admin/users/${encodeURIComponent(userID)}/mfa-enrollment-bypass`,
      adminRequest(csrfToken, 'POST', { expires_in_seconds: 900 }),
    )
  ).bypass
}

export async function revokeMfaEnrollmentBypass(csrfToken: string, userID: string): Promise<void> {
  await request(
    `/api/admin/users/${encodeURIComponent(userID)}/mfa-enrollment-bypass`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// Admin session management (wi-28 T007, ADR-127 決定9): view and revoke a
// target user's sessions. Unlike self-service /api/account/sessions, these
// have no `current` marker and session revoke also cascades to that
// session's refresh tokens server-side (RevokeTokensBySid).
export async function listAdminUserSessions(userID: string): Promise<AdminSessionRecord[]> {
  return (
    await request<{ sessions: AdminSessionRecord[] }>(
      `/api/admin/users/${encodeURIComponent(userID)}/sessions`,
    )
  ).sessions
}

export async function revokeAdminUserSession(
  csrfToken: string,
  userID: string,
  sessionID: string,
): Promise<void> {
  await request(
    `/api/admin/users/${encodeURIComponent(userID)}/sessions/${encodeURIComponent(sessionID)}/revoke`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function revokeAllAdminUserSessions(csrfToken: string, userID: string): Promise<void> {
  await request(
    `/api/admin/users/${encodeURIComponent(userID)}/sessions/revoke_all`,
    adminRequest(csrfToken, 'POST'),
  )
}

// ApplicationCategory の管理 (wi-70, ADR-069)。tenant 単位で定義し Application に付与する。
export async function listApplicationCategories(): Promise<ApplicationCategory[]> {
  return (await request<{ categories: ApplicationCategory[] }>('/api/admin/application-categories'))
    .categories
}

export type ApplicationCategoryInput = {
  name: string
  position?: number
}

export async function createApplicationCategory(
  csrfToken: string,
  input: ApplicationCategoryInput,
): Promise<ApplicationCategory> {
  return (
    await request<{ category: ApplicationCategory }>(
      '/api/admin/application-categories',
      adminRequest(csrfToken, 'POST', input),
    )
  ).category
}

export async function updateApplicationCategory(
  csrfToken: string,
  categoryID: string,
  input: ApplicationCategoryInput,
): Promise<ApplicationCategory> {
  return (
    await request<{ category: ApplicationCategory }>(
      `/api/admin/application-categories/${encodeURIComponent(categoryID)}`,
      adminRequest(csrfToken, 'PATCH', input),
    )
  ).category
}

export async function deleteApplicationCategory(
  csrfToken: string,
  categoryID: string,
): Promise<void> {
  await request(
    `/api/admin/application-categories/${encodeURIComponent(categoryID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function setApplicationCategories(
  csrfToken: string,
  id: string,
  categoryIDs: string[],
): Promise<AdminApplication> {
  return request(
    `/api/admin/applications/${encodeURIComponent(id)}/categories`,
    adminRequest(csrfToken, 'PUT', { category_ids: categoryIDs }),
  )
}

export async function listApiTokens(): Promise<ApiToken[]> {
  return (await request<{ tokens: ApiToken[] }>('/api/admin/api-tokens')).tokens
}

export async function createApiToken(
  csrfToken: string,
  input: { description: string; scopes: ApiTokenScope[]; expiry_days: number },
): Promise<{ token: string; meta: ApiToken }> {
  return request('/api/admin/api-tokens', adminRequest(csrfToken, 'POST', input))
}

export async function revokeApiToken(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/api-tokens/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// Provisioning (outbound SCIM, wi-45): Application 詳細ページの「プロビジョニング」サブルート
// と、テナント全体の読み取り専用集約ビューが使う管理 API クライアント。
export type ProvisioningCredentialInput = {
  auth_method: ProvisioningAuthMethod
  bearer_token?: string
  oauth2_token_url?: string
  oauth2_client_id?: string
  oauth2_client_secret?: string
  oauth2_scope?: string
}

export type RegisterAdminApplicationProvisioningInput = {
  base_url: string
  credential: ProvisioningCredentialInput
}

export async function registerAdminApplicationProvisioning(
  csrfToken: string,
  applicationID: string,
  input: RegisterAdminApplicationProvisioningInput,
): Promise<ProvisioningConnection> {
  return request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning`,
    adminRequest(csrfToken, 'POST', input),
  )
}

export async function getAdminApplicationProvisioning(
  applicationID: string,
): Promise<ProvisioningConnection> {
  return request(`/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning`)
}

export type UpdateAdminApplicationProvisioningInput = {
  base_url?: string
  status?: ProvisioningConnectionStatus
  credential?: ProvisioningCredentialInput
  feature_flags?: ProvisioningFeatureFlags
  scope?: ProvisioningScope
  group_push?: GroupPushConfig | null
  attribute_mappings?: AttributeMappingRule[]
  matching?: MatchingRule
  deprovision_policy?: DeprovisionPolicy
  rate_limit_per_minute?: number
  max_attempts?: number
  notification_email?: string | null
  quarantine_after_consecutive_failures?: number
}

export async function updateAdminApplicationProvisioning(
  csrfToken: string,
  applicationID: string,
  input: UpdateAdminApplicationProvisioningInput,
): Promise<ProvisioningConnection> {
  return request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function deleteAdminApplicationProvisioning(
  csrfToken: string,
  applicationID: string,
): Promise<void> {
  await request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function testAdminApplicationProvisioning(
  csrfToken: string,
  applicationID: string,
): Promise<ProvisioningTestConnectionResult> {
  return request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning/test`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function provisionOnDemand(
  csrfToken: string,
  applicationID: string,
  subjectType: ProvisioningSourceType,
  subjectID: string,
): Promise<ProvisioningDelivery> {
  return request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning/on-demand`,
    adminRequest(csrfToken, 'POST', { subject_type: subjectType, subject_id: subjectID }),
  )
}

export async function startAdminApplicationProvisioningFullResync(
  csrfToken: string,
  applicationID: string,
): Promise<{ enqueued_count: number }> {
  return request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning/full-resync`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function resumeAdminApplicationProvisioning(
  csrfToken: string,
  applicationID: string,
): Promise<ProvisioningConnection> {
  return request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning/resume`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function listAdminApplicationProvisioningDeliveries(
  applicationID: string,
  status?: ProvisioningDeliveryStatus,
): Promise<ProvisioningDelivery[]> {
  const query = status ? `?status=${encodeURIComponent(status)}` : ''
  return (
    await request<{ deliveries: ProvisioningDelivery[] }>(
      `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning/deliveries${query}`,
    )
  ).deliveries
}

export async function getAdminApplicationProvisioningDelivery(
  applicationID: string,
  deliveryID: string,
): Promise<ProvisioningDelivery> {
  return request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning/deliveries/${encodeURIComponent(deliveryID)}`,
  )
}

export async function retryAdminApplicationProvisioningDelivery(
  csrfToken: string,
  applicationID: string,
  deliveryID: string,
): Promise<ProvisioningDelivery> {
  return request(
    `/api/admin/applications/${encodeURIComponent(applicationID)}/provisioning/deliveries/${encodeURIComponent(deliveryID)}/retry`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function listAdminTenantProvisioningConnections(): Promise<ProvisioningConnection[]> {
  return (
    await request<{ connections: ProvisioningConnection[] }>('/api/admin/provisioning/connections')
  ).connections
}
