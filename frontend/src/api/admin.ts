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
  AdminIntegrationEndpointCatalog,
  AdminSamlIDPProfile,
  ApiToken,
  ApiTokenScope,
  TenantKeyHealth,
  TenantDataKeyHealth,
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
  SamlIDPProfileMode,
  AuthorizationDetailType,
  McpResourceServer,
  TenantUserAttributeSchema,
  NotificationTemplateDetail,
  NotificationTemplateInput,
  NotificationTemplateList,
  NotificationTemplatePreview,
  NotificationTemplateTestResult,
  UserAttributeDef,
  EntraFederationProfile,
  DataExportJob,
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
  ClientSecretCredentialMetadata,
} from '../types'
import {
  authenticationAPIError,
  AuthenticationAPIError,
  adminRequest,
  request,
  requestPage,
  tenantURL,
  type APIError,
} from './core'

type AdminUserListResponse = { users: AdminUser[] }
type AdminConsentListResponse = { consents: AdminConsent[] }
type AdminAuditEventListResponse = { events: AdminAuditEvent[] }
type AdminKeyListResponse = { keys: AdminKey[] }
type TenantKeyHealthListResponse = { tenants: TenantKeyHealth[] }
type TenantDataKeyHealthListResponse = { tenants: TenantDataKeyHealth[] }
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

// 一覧 API は大規模テナントでのコスト削減のため既定 50 件・最大 200 件の keyset
// pagination になっている (ADR-158)。PICKER_LIST_LIMIT はプライマリの一覧画面ではなく
// picker/lookup 用途 (グループ追加候補、割り当て対象選択、id→name 解決など) の呼び出し元が
// 既定値 (50件) に事故で切り詰められないよう明示する上限。200 件を超えるテナントでは
// picker が一部の候補を表示できなくなるが、これは Design が許容する「capped query」の範囲
// (全件を検索可能にする UI は wi-161 の対象)。
const PICKER_LIST_LIMIT = 200

function pageQueryString(params?: {
  cursor?: string
  limit?: number
  query?: string
  status?: string
}): string {
  if (!params) return ''
  const query = new URLSearchParams()
  if (params.cursor) query.set('cursor', params.cursor)
  if (params.limit) query.set('limit', String(params.limit))
  if (params.query) query.set('query', params.query)
  if (params.status) query.set('status', params.status)
  const qs = query.toString()
  return qs ? `?${qs}` : ''
}

export type AdminUserPage = {
  users: AdminUser[]
  previousCursor: string | null
  nextCursor: string | null
}

export async function listAdminUsers(): Promise<AdminUser[]> {
  return (await request<AdminUserListResponse>(`/api/admin/v1/users?limit=${PICKER_LIST_LIMIT}`))
    .users
}

// listAdminUsersPage はユーザー一覧画面専用の addressable cursor pagination 版 (ADR-159)。
export async function listAdminUsersPage(params?: {
  cursor?: string
  limit?: number
  query?: string
  status?: string
}): Promise<AdminUserPage> {
  const page = await requestPage<AdminUserListResponse>(
    `/api/admin/v1/users${pageQueryString(params)}`,
  )
  return {
    users: page.body.users,
    previousCursor: page.previousCursor,
    nextCursor: page.nextCursor,
  }
}

export async function getAdminUser(id: string): Promise<AdminUser> {
  return request<AdminUser>(`/api/admin/v1/users/${encodeURIComponent(id)}`)
}

export async function createAdminUser(
  csrfToken: string,
  input: CreateAdminUserInput,
): Promise<AdminUser> {
  return request('/api/admin/v1/users', adminRequest(csrfToken, 'POST', input))
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
    `/api/admin/v1/users/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function setAdminUserRequiredAction(
  csrfToken: string,
  id: string,
  action: string,
): Promise<AdminUser> {
  return request(
    `/api/admin/v1/users/${encodeURIComponent(id)}/required_actions`,
    adminRequest(csrfToken, 'POST', { action }),
  )
}

export async function clearAdminUserRequiredAction(
  csrfToken: string,
  id: string,
  action: string,
): Promise<AdminUser> {
  return request(
    `/api/admin/v1/users/${encodeURIComponent(id)}/required_actions/${encodeURIComponent(action)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function updateAdminTenantQuota(
  csrfToken: string,
  id: string,
  input: TenantQuota,
): Promise<TenantQuota> {
  return request(
    `/api/admin/v1/tenants/${encodeURIComponent(id)}/quota`,
    adminRequest(csrfToken, 'PUT', input),
  )
}

export async function setAdminUserDisabled(
  csrfToken: string,
  id: string,
  disabled: boolean,
): Promise<void> {
  await request(
    `/api/admin/v1/users/${encodeURIComponent(id)}/${disabled ? 'disable' : 'enable'}`,
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
    `/api/admin/v1/users/${encodeURIComponent(id)}${query}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// restoreAdminUser は削除予約中 (pending_deletion) のユーザーを復元する。
export async function restoreAdminUser(csrfToken: string, id: string): Promise<AdminUser> {
  return request(
    `/api/admin/v1/users/${encodeURIComponent(id)}/restore`,
    adminRequest(csrfToken, 'POST'),
  )
}

// importAdminUsers は CSV を dry_run (検証のみ) または apply (作成) ジョブとして投入する。
// 202 応答はジョブ受理のみを表し、結果は getAdminUserImport の polling で取得する。
export async function importAdminUsers(
  csrfToken: string,
  input: { csv: string; mode: UserImportMode },
): Promise<UserImportJobSummary> {
  return request('/api/admin/v1/users/imports', adminRequest(csrfToken, 'POST', input))
}

export async function getAdminUserImport(jobId: string): Promise<UserImportJob> {
  return request(`/api/admin/v1/users/imports/${encodeURIComponent(jobId)}`)
}

// wi-148: 管理者向け CSV データエクスポート (per-type)。各リソース種別ごとに同じ形の
// start / list / get / file を持つため、base path でパラメタ化した ExportCollection にする。
export type StartExportInput = { columns: string[]; filter?: Record<string, string> }
type DataExportListResponse = { exports: DataExportJob[] }

export type ExportCollection = {
  start: (csrfToken: string, input: StartExportInput) => Promise<DataExportJob>
  list: () => Promise<DataExportJob[]>
  get: (exportId: string) => Promise<DataExportJob>
  cancel: (csrfToken: string, exportId: string) => Promise<DataExportJob>
  // fileURL は <a href> / window ダウンロード用の tenant 絶対パス。
  fileURL: (exportId: string) => string
}

function exportCollection(basePath: string): ExportCollection {
  return {
    start: (csrfToken, input) => request(basePath, adminRequest(csrfToken, 'POST', input)),
    list: async () => (await request<DataExportListResponse>(basePath)).exports,
    get: (exportId) => request(`${basePath}/${encodeURIComponent(exportId)}`),
    cancel: (csrfToken, exportId) =>
      request(
        `${basePath}/${encodeURIComponent(exportId)}/cancel`,
        adminRequest(csrfToken, 'POST'),
      ),
    fileURL: (exportId) => tenantURL(`${basePath}/${encodeURIComponent(exportId)}/file`),
  }
}

export const userExports = exportCollection('/api/admin/v1/users/exports')
export const groupExports = exportCollection('/api/admin/v1/groups/exports')

// メンバーエクスポートは特定グループ配下 (per-group)。group_id を束ねた ExportCollection を返す。
export function groupMemberExports(groupId: string): ExportCollection {
  return exportCollection(`/api/admin/v1/groups/${encodeURIComponent(groupId)}/members/exports`)
}

export type LifecycleWorkflowInput = {
  expected_revision?: number
  name: string
  description?: string
  trigger: WorkflowTrigger
  actions: WorkflowAction[]
}
export async function listLifecycleWorkflows(): Promise<AdminLifecycleWorkflow[]> {
  return (
    await request<{ workflows: AdminLifecycleWorkflow[] }>('/api/admin/v1/lifecycle_workflows')
  ).workflows
}
export async function getLifecycleWorkflow(id: string): Promise<AdminLifecycleWorkflow> {
  return request(`/api/admin/v1/lifecycle_workflows/${encodeURIComponent(id)}`)
}
export async function createLifecycleWorkflow(
  csrfToken: string,
  input: LifecycleWorkflowInput,
): Promise<AdminLifecycleWorkflow> {
  return request('/api/admin/v1/lifecycle_workflows', adminRequest(csrfToken, 'POST', input))
}
export async function updateLifecycleWorkflow(
  csrfToken: string,
  id: string,
  input: LifecycleWorkflowInput,
): Promise<AdminLifecycleWorkflow> {
  return request(
    `/api/admin/v1/lifecycle_workflows/${encodeURIComponent(id)}`,
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
    `/api/admin/v1/lifecycle_workflows/${encodeURIComponent(id)}/${state}`,
    adminRequest(csrfToken, 'POST', { expected_revision }),
  )
}
export async function deleteLifecycleWorkflow(
  csrfToken: string,
  id: string,
  expected_revision: number,
): Promise<void> {
  await request(
    `/api/admin/v1/lifecycle_workflows/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'DELETE', { expected_revision }),
  )
}
export async function dryRunLifecycleWorkflow(
  csrfToken: string,
  id: string,
  targetUserID: string,
): Promise<{ steps: { action_kind: string; would_change: string; reason?: string }[] }> {
  return request(
    `/api/admin/v1/lifecycle_workflows/${encodeURIComponent(id)}/dry_run`,
    adminRequest(csrfToken, 'POST', { target_user_id: targetUserID }),
  )
}
export async function listLifecycleWorkflowRuns(id: string): Promise<WorkflowRun[]> {
  return (
    await request<{ runs: WorkflowRun[] }>(
      `/api/admin/v1/lifecycle_workflows/${encodeURIComponent(id)}/runs`,
    )
  ).runs
}
export async function retryLifecycleWorkflowRun(
  csrfToken: string,
  id: string,
): Promise<WorkflowRun> {
  return request(
    `/api/admin/v1/lifecycle_workflow_runs/${encodeURIComponent(id)}/retry`,
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
    await request<{ types: AuthorizationDetailType[] }>('/api/admin/v1/authorization-detail-types')
  ).types
}

export async function createAuthorizationDetailType(
  csrfToken: string,
  input: AuthorizationDetailTypeInput,
): Promise<AuthorizationDetailType> {
  return request('/api/admin/v1/authorization-detail-types', adminRequest(csrfToken, 'POST', input))
}

export async function updateAuthorizationDetailType(
  csrfToken: string,
  detailType: string,
  input: AuthorizationDetailTypeInput,
): Promise<AuthorizationDetailType> {
  return request(
    `/api/admin/v1/authorization-detail-types/${encodeURIComponent(detailType)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function deleteAuthorizationDetailType(
  csrfToken: string,
  detailType: string,
): Promise<void> {
  await request(
    `/api/admin/v1/authorization-detail-types/${encodeURIComponent(detailType)}`,
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
    await request<{ resource_servers: McpResourceServer[] }>('/api/admin/v1/mcp-resource-servers')
  ).resource_servers
}

export async function createMcpResourceServer(
  csrfToken: string,
  input: McpResourceServerInput,
): Promise<McpResourceServer> {
  return request('/api/admin/v1/mcp-resource-servers', adminRequest(csrfToken, 'POST', input))
}

export async function updateMcpResourceServer(
  csrfToken: string,
  resourceServerID: string,
  input: McpResourceServerInput,
): Promise<McpResourceServer> {
  return request(
    `/api/admin/v1/mcp-resource-servers/${encodeURIComponent(resourceServerID)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function deleteMcpResourceServer(
  csrfToken: string,
  resourceServerID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/mcp-resource-servers/${encodeURIComponent(resourceServerID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

type WsFedRelyingPartyListResponse = { relying_parties: WsFedRelyingParty[] | null }

export async function listWsFedRelyingParties(): Promise<WsFedRelyingParty[]> {
  const response = await request<WsFedRelyingPartyListResponse>(
    '/api/admin/v1/wsfed/relying-parties',
  )
  return response.relying_parties ?? []
}

export async function deleteWsFedRelyingParty(csrfToken: string, wtrealm: string): Promise<void> {
  await request(
    `/api/admin/v1/wsfed/relying-parties?wtrealm=${encodeURIComponent(wtrealm)}`,
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
  return request('/api/admin/v1/wsfed/entra-federation', adminRequest(csrfToken, 'POST', input))
}

// Consents 一覧画面はまだ「さらに読み込む」UI に移行しておらず (T007 follow-up)、当面は
// PICKER_LIST_LIMIT で切り詰めた1ページのみを表示する。
export async function listAdminConsents(): Promise<AdminConsent[]> {
  return (
    await request<AdminConsentListResponse>(`/api/admin/v1/consents?limit=${PICKER_LIST_LIMIT}`)
  ).consents
}

export async function revokeAdminConsent(
  csrfToken: string,
  userID: string,
  clientID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/consents/${encodeURIComponent(userID)}/${encodeURIComponent(clientID)}`,
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
  // cursor (ADR-159): Link の prev/next cursor を渡して隣接ページを取得する。フィルタ変更時は
  // 呼び出し側で cursor を落として先頭ページに戻す。
  cursor?: string
  allTenants?: boolean
  filter?: string[]
}

// 監査イベント検索フォームが URL query string と同期する部分 (wi-147)。type と cursor は
// 検索フォームの入力ではないため除く (cursor はページ位置を表す URL 状態)。
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
  if (query.cursor) params.set('cursor', query.cursor)
  if (query.allTenants) params.set('all_tenants', 'true')
  for (const filter of query.filter ?? []) {
    if (filter) params.append('filter', filter)
  }
  return params
}

export type AdminAuditEventPage = {
  events: AdminAuditEvent[]
  previousCursor: string | null
  nextCursor: string | null
}

export async function listAdminAuditEvents(
  query: AdminAuditEventQuery,
): Promise<AdminAuditEventPage> {
  const params = auditEventParams(query)
  const url =
    params.size > 0
      ? `/api/admin/v1/audit_events?${params.toString()}`
      : '/api/admin/v1/audit_events'
  const page = await requestPage<AdminAuditEventListResponse>(url)
  return {
    events: page.body.events,
    previousCursor: page.previousCursor,
    nextCursor: page.nextCursor,
  }
}

// 監査イベントのエクスポート URL (認証イベント含む)。新規タブで開いてダウンロードする。
export function adminAuditEventsExportURL(query: AdminAuditEventQuery): string {
  const params = auditEventParams(query)
  return tenantURL(`/api/admin/v1/audit_events/export?${params.toString()}`)
}

// event.type / outcome を選択式にするための選択肢一覧 (wi-147)。UI 側でハードコードせず、
// Go 側の単一の正 (auditEventCategoryTypes / eventOutcome) から機械的に取得する。
export type AdminAuditEventSearchOptions = {
  event_types: string[]
  outcomes: string[]
}

export async function listAdminAuditEventSearchOptions(): Promise<AdminAuditEventSearchOptions> {
  return request<AdminAuditEventSearchOptions>('/api/admin/v1/audit_events/search_options')
}

export async function listAdminKeys(): Promise<AdminKey[]> {
  return (await request<AdminKeyListResponse>('/api/admin/v1/keys')).keys
}

export async function rotateTenantSigningKey(csrfToken: string): Promise<AdminRotateKeyResponse> {
  return request<AdminRotateKeyResponse>(
    '/api/admin/v1/keys/rotate',
    adminRequest(csrfToken, 'POST'),
  )
}

export async function disableTenantKey(csrfToken: string, kid: string): Promise<AdminKey> {
  return request<AdminKey>(
    `/api/admin/v1/keys/${encodeURIComponent(kid)}/disable`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function listTenantKeyHealth(): Promise<TenantKeyHealth[]> {
  return (await request<TenantKeyHealthListResponse>('/api/admin/v1/keys/health')).tenants
}

export async function listTenantDataKeyHealth(): Promise<TenantDataKeyHealth[]> {
  return (await request<TenantDataKeyHealthListResponse>('/api/admin/v1/data-keys/health')).tenants
}

export type UpdateAdminSettingsInput = {
  display_name?: string
  password_policy_override?: AdminSettings['password_policy_override']
  // 空文字列でシステム既定へ戻す (省略は現状維持)。
  default_locale?: string
}

export async function getAdminSettings(): Promise<AdminSettings> {
  return request<AdminSettings>('/api/admin/v1/settings')
}

export async function getAdminIntegrationEndpoints(): Promise<AdminIntegrationEndpointCatalog> {
  return request<AdminIntegrationEndpointCatalog>('/api/admin/v1/integration-endpoints')
}

export async function updateAdminSettings(
  csrfToken: string,
  input: UpdateAdminSettingsInput,
): Promise<AdminSettings> {
  return request('/api/admin/v1/settings', adminRequest(csrfToken, 'PATCH', input))
}

export type IdentityProviderConnection = {
  id: string
  tenant_id: string
  display_name: string
  protocol: 'oidc' | 'saml'
  status: 'active' | 'disabled'
  issuer: string
  client_id?: string
  client_secret_configured: boolean
  authorization_endpoint?: string
  token_endpoint?: string
  jwks_uri?: string
  saml_sso_url?: string
  saml_entity_id?: string
  saml_signing_certificates?: string[]
  claim_mapping: {
    subject: string
    username: string
    email?: string
    email_verified?: string
    name?: string
  }
  linking_policy: 'none' | 'verified_email'
  jit_provisioning: boolean
  allowed_email_domains?: string[]
  metadata_refreshed_at?: string
  created_at: string
  updated_at: string
}

// secret_reference は書き込み専用でクライアントシークレットの実値を受け取る
// (ADR-150)。未入力なら既存の値を維持する。レスポンス側の型には含まれない
// (API レスポンスには実値もciphertextも含まれないため IdentityProviderConnection
// 自体に secret_reference フィールドはない)。
export type IdentityProviderConnectionInput = Omit<
  IdentityProviderConnection,
  | 'id'
  | 'tenant_id'
  | 'status'
  | 'client_secret_configured'
  | 'metadata_refreshed_at'
  | 'created_at'
  | 'updated_at'
> & { secret_reference?: string }

export type IdentityProviderConnectionTestResult = {
  success: boolean
  failures: string[]
}

export async function listIdentityProviderConnections(): Promise<IdentityProviderConnection[]> {
  const response = await request<{ connections: IdentityProviderConnection[] }>(
    '/api/admin/v1/identity-providers',
  )
  return response.connections ?? []
}

export async function createIdentityProviderConnection(
  csrfToken: string,
  input: IdentityProviderConnectionInput,
): Promise<IdentityProviderConnection> {
  return request('/api/admin/v1/identity-providers', adminRequest(csrfToken, 'POST', input))
}

export async function updateIdentityProviderConnection(
  csrfToken: string,
  providerID: string,
  input: IdentityProviderConnectionInput,
): Promise<IdentityProviderConnection> {
  return request(
    `/api/admin/v1/identity-providers/${encodeURIComponent(providerID)}`,
    adminRequest(csrfToken, 'PUT', input),
  )
}

export async function runIdentityProviderAction(
  csrfToken: string,
  providerID: string,
  action: 'activate' | 'disable' | 'refresh',
): Promise<void> {
  await request(
    `/api/admin/v1/identity-providers/${encodeURIComponent(providerID)}/${action}`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function testIdentityProviderConnection(
  csrfToken: string,
  providerID: string,
): Promise<IdentityProviderConnectionTestResult> {
  const response = await request<{ result: IdentityProviderConnectionTestResult }>(
    `/api/admin/v1/identity-providers/${encodeURIComponent(providerID)}/test`,
    adminRequest(csrfToken, 'POST'),
  )
  return response.result
}

export async function previewIdentityProviderMapping(
  csrfToken: string,
  providerID: string,
  claims: Record<string, unknown>,
): Promise<{
  subject: string
  username: string
  email?: string
  email_verified: boolean
  name?: string
}> {
  const response = await request<{
    preview: {
      subject: string
      username: string
      email?: string
      email_verified: boolean
      name?: string
    }
  }>(
    `/api/admin/v1/identity-providers/${encodeURIComponent(providerID)}/mapping-preview`,
    adminRequest(csrfToken, 'POST', { claims }),
  )
  return response.preview
}

export async function deleteIdentityProviderConnection(
  csrfToken: string,
  providerID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/identity-providers/${encodeURIComponent(providerID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}
// 通知テンプレート (wi-288, ADR-142)。文面は組込み既定カタログとテナント上書きの
// 2 段で解決され、DELETE (reset) は上書きを消して組込み既定へ戻す。
function notificationTemplatePath(templateKey: string, locale: string): string {
  return `/api/admin/v1/tenant/notification_templates/${encodeURIComponent(templateKey)}/${encodeURIComponent(locale)}`
}

export async function listNotificationTemplates(): Promise<NotificationTemplateList> {
  return request<NotificationTemplateList>('/api/admin/v1/tenant/notification_templates')
}

export async function getNotificationTemplate(
  templateKey: string,
  locale: string,
): Promise<NotificationTemplateDetail> {
  return request<NotificationTemplateDetail>(notificationTemplatePath(templateKey, locale))
}

export async function updateNotificationTemplate(
  csrfToken: string,
  templateKey: string,
  locale: string,
  input: NotificationTemplateInput,
): Promise<NotificationTemplateDetail> {
  return request(
    notificationTemplatePath(templateKey, locale),
    adminRequest(csrfToken, 'PUT', input),
  )
}

export async function resetNotificationTemplate(
  csrfToken: string,
  templateKey: string,
  locale: string,
): Promise<NotificationTemplateDetail> {
  return request(notificationTemplatePath(templateKey, locale), adminRequest(csrfToken, 'DELETE'))
}

export async function previewNotificationTemplate(
  csrfToken: string,
  templateKey: string,
  locale: string,
  input: Partial<NotificationTemplateInput>,
): Promise<NotificationTemplatePreview> {
  return request(
    `${notificationTemplatePath(templateKey, locale)}/preview`,
    adminRequest(csrfToken, 'POST', input),
  )
}

// 宛先は指定できない。サーバが操作者本人の検証済みアドレスに固定する (ADR-142 決定 8)。
export async function sendTestNotification(
  csrfToken: string,
  templateKey: string,
  locale: string,
): Promise<NotificationTemplateTestResult> {
  return request(
    `${notificationTemplatePath(templateKey, locale)}/test`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function getTenantUserAttributeSchema(): Promise<TenantUserAttributeSchema> {
  return request<TenantUserAttributeSchema>('/api/admin/v1/tenant/user_attribute_schema')
}

export async function updateTenantUserAttributeSchema(
  csrfToken: string,
  attributes: UserAttributeDef[],
): Promise<TenantUserAttributeSchema> {
  return request(
    '/api/admin/v1/tenant/user_attribute_schema',
    adminRequest(csrfToken, 'PUT', { attributes }),
  )
}

export async function listAdminTenants(): Promise<AdminTenant[]> {
  return (await request<AdminTenantListResponse>('/api/admin/v1/tenants')).tenants
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
  return request('/api/admin/v1/tenants', adminRequest(csrfToken, 'POST', input))
}

export async function updateAdminTenant(
  csrfToken: string,
  tenantID: string,
  input: UpdateAdminTenantInput,
): Promise<AdminTenant> {
  return request(
    `/api/admin/v1/tenants/${encodeURIComponent(tenantID)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function setAdminTenantDisabled(
  csrfToken: string,
  tenantID: string,
  disabled: boolean,
): Promise<void> {
  await request(
    `/api/admin/v1/tenants/${encodeURIComponent(tenantID)}/${disabled ? 'disable' : 'enable'}`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function setAdminTenantEndpointStyle(
  csrfToken: string,
  tenantID: string,
  endpointStyle: NonNullable<AdminTenant['endpoint_style']>,
): Promise<void> {
  await request(
    `/api/admin/v1/tenants/${encodeURIComponent(tenantID)}/endpoint_style`,
    adminRequest(csrfToken, 'PUT', { endpoint_style: endpointStyle }),
  )
}

export async function listAdminGroups(): Promise<AdminGroup[]> {
  return (
    await request<{ groups: AdminGroup[] }>(`/api/admin/v1/groups?limit=${PICKER_LIST_LIMIT}`)
  ).groups
}

export type AdminGroupPage = {
  groups: AdminGroup[]
  previousCursor: string | null
  nextCursor: string | null
}

// listAdminGroupsPage はグループ一覧画面専用の addressable cursor pagination 版 (ADR-159)。
export async function listAdminGroupsPage(params?: {
  cursor?: string
  limit?: number
}): Promise<AdminGroupPage> {
  const page = await requestPage<{ groups: AdminGroup[] }>(
    `/api/admin/v1/groups${pageQueryString(params)}`,
  )
  return {
    groups: page.body.groups,
    previousCursor: page.previousCursor,
    nextCursor: page.nextCursor,
  }
}

export async function getAdminGroup(
  id: string,
): Promise<{ group: AdminGroup; members: AdminGroupMember[] }> {
  return request(`/api/admin/v1/groups/${encodeURIComponent(id)}`)
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
    `/api/admin/v1/groups/${encodeURIComponent(id)}/dynamic-rule`,
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
    `/api/admin/v1/groups/${encodeURIComponent(id)}/dynamic-rule/preview`,
    adminRequest(csrfToken, 'POST', { expression, user_ids: userIDs }),
  )
}

export async function setDynamicGroupRuleEnabled(csrfToken: string, id: string, enabled: boolean) {
  return request<NonNullable<AdminGroup['dynamic_rule']>>(
    `/api/admin/v1/groups/${encodeURIComponent(id)}/dynamic-rule/${enabled ? 'enable' : 'disable'}`,
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
  return request('/api/admin/v1/groups', adminRequest(csrfToken, 'POST', input))
}

export async function updateAdminGroup(
  csrfToken: string,
  id: string,
  input: UpdateAdminGroupInput,
): Promise<AdminGroup> {
  return request(
    `/api/admin/v1/groups/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function deleteAdminGroup(csrfToken: string, id: string): Promise<void> {
  await request(`/api/admin/v1/groups/${encodeURIComponent(id)}`, adminRequest(csrfToken, 'DELETE'))
}

export async function addAdminGroupMember(
  csrfToken: string,
  groupID: string,
  userID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/groups/${encodeURIComponent(groupID)}/members/${encodeURIComponent(userID)}`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function removeAdminGroupMember(
  csrfToken: string,
  groupID: string,
  userID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/groups/${encodeURIComponent(groupID)}/members/${encodeURIComponent(userID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function getAdminUserGroups(id: string): Promise<AdminUserGroups> {
  return request(`/api/admin/v1/users/${encodeURIComponent(id)}/groups`)
}

export async function listAdminAgents(): Promise<AdminAgent[]> {
  return (
    await request<{ agents: AdminAgent[] }>(`/api/admin/v1/agents?limit=${PICKER_LIST_LIMIT}`)
  ).agents
}

export type AdminAgentPage = {
  agents: AdminAgent[]
  previousCursor: string | null
  nextCursor: string | null
}

// listAdminAgentsPage はエージェント一覧画面専用の addressable cursor pagination 版 (ADR-159)。
export async function listAdminAgentsPage(params?: {
  cursor?: string
  limit?: number
}): Promise<AdminAgentPage> {
  const page = await requestPage<{ agents: AdminAgent[] }>(
    `/api/admin/v1/agents${pageQueryString(params)}`,
  )
  return {
    agents: page.body.agents,
    previousCursor: page.previousCursor,
    nextCursor: page.nextCursor,
  }
}

export async function getAdminAgent(id: string): Promise<AdminAgent> {
  return request<AdminAgent>(`/api/admin/v1/agents/${encodeURIComponent(id)}`)
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
  return request('/api/admin/v1/agents', adminRequest(csrfToken, 'POST', input))
}

export async function updateAdminAgent(
  csrfToken: string,
  id: string,
  input: UpdateAdminAgentInput,
): Promise<AdminAgent> {
  return request(
    `/api/admin/v1/agents/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function disableAdminAgent(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/v1/agents/${encodeURIComponent(id)}/disable`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function enableAdminAgent(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/v1/agents/${encodeURIComponent(id)}/enable`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function killAdminAgent(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/v1/agents/${encodeURIComponent(id)}/kill`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function deleteAdminAgent(csrfToken: string, id: string): Promise<void> {
  await request(`/api/admin/v1/agents/${encodeURIComponent(id)}`, adminRequest(csrfToken, 'DELETE'))
}

export async function bindAdminAgentCredential(
  csrfToken: string,
  agentID: string,
  clientID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/agents/${encodeURIComponent(agentID)}/credentials`,
    adminRequest(csrfToken, 'POST', { client_id: clientID }),
  )
}

export async function unbindAdminAgentCredential(
  csrfToken: string,
  agentID: string,
  clientID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/agents/${encodeURIComponent(agentID)}/credentials/${encodeURIComponent(clientID)}`,
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
  sub_source_attribute?: string
  rules?: WsFedClaimMappingRule[]
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
  idp_profile_id?: string
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

export async function listSamlIDPProfiles(): Promise<AdminSamlIDPProfile[]> {
  return (await request<{ profiles: AdminSamlIDPProfile[] }>('/api/admin/v1/saml/idp-profiles'))
    .profiles
}

export async function createSamlIDPProfile(
  csrfToken: string,
  input: { name: string; mode: SamlIDPProfileMode },
): Promise<AdminSamlIDPProfile> {
  return request('/api/admin/v1/saml/idp-profiles', adminRequest(csrfToken, 'POST', input))
}

export async function updateSamlIDPProfile(
  csrfToken: string,
  profileID: string,
  input: { name: string; mode: SamlIDPProfileMode },
): Promise<AdminSamlIDPProfile> {
  return request(
    `/api/admin/v1/saml/idp-profiles/${encodeURIComponent(profileID)}`,
    adminRequest(csrfToken, 'PUT', input),
  )
}

export async function deleteSamlIDPProfile(csrfToken: string, profileID: string): Promise<void> {
  await request(
    `/api/admin/v1/saml/idp-profiles/${encodeURIComponent(profileID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function listAdminApplications(): Promise<AdminApplication[]> {
  return (
    await request<{ applications: AdminApplication[] }>(
      `/api/admin/v1/applications?limit=${PICKER_LIST_LIMIT}`,
    )
  ).applications
}

export type AdminApplicationPage = {
  applications: AdminApplication[]
  previousCursor: string | null
  nextCursor: string | null
}

// listAdminApplicationsPage はアプリケーション一覧画面専用の addressable cursor pagination 版 (ADR-159)。
export async function listAdminApplicationsPage(params?: {
  cursor?: string
  limit?: number
}): Promise<AdminApplicationPage> {
  const page = await requestPage<{ applications: AdminApplication[] }>(
    `/api/admin/v1/applications${pageQueryString(params)}`,
  )
  return {
    applications: page.body.applications,
    previousCursor: page.previousCursor,
    nextCursor: page.nextCursor,
  }
}

export async function getAdminApplication(id: string): Promise<AdminApplicationDetail> {
  return request<AdminApplicationDetail>(`/api/admin/v1/applications/${encodeURIComponent(id)}`)
}

export async function createAdminApplication(
  csrfToken: string,
  input: CreateAdminApplicationInput,
): Promise<CreateAdminApplicationResult> {
  return request('/api/admin/v1/applications', adminRequest(csrfToken, 'POST', input))
}

export async function updateApplicationOidcConfig(
  csrfToken: string,
  id: string,
  input: UpdateApplicationOidcInput,
): Promise<void> {
  await request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/oidc`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function rotateApplicationClientSecret(
  csrfToken: string,
  id: string,
  graceDays: number,
): Promise<{ client_secret: string; grace_until?: string }> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/oidc/rotate-secret`,
    adminRequest(csrfToken, 'POST', { grace_days: graceDays }),
  )
}

export type IssueApplicationClientSecretResult = {
  client_secret: string
  credential: ClientSecretCredentialMetadata
  credentials: ClientSecretCredentialMetadata[]
}

export async function issueApplicationClientSecret(
  csrfToken: string,
  id: string,
  expiresInDays: number,
): Promise<IssueApplicationClientSecretResult> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/oidc/client-secrets`,
    adminRequest(csrfToken, 'POST', { expires_in_days: expiresInDays }),
  )
}

export async function revokeApplicationClientSecret(
  csrfToken: string,
  id: string,
  credentialID: string,
): Promise<{ credentials: ClientSecretCredentialMetadata[] }> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/oidc/client-secrets/${encodeURIComponent(credentialID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function updateApplicationWsFedConfig(
  csrfToken: string,
  id: string,
  input: UpdateApplicationWsFedInput,
): Promise<void> {
  await request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/wsfed`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function updateApplicationSamlConfig(
  csrfToken: string,
  id: string,
  input: UpdateApplicationSamlInput,
): Promise<void> {
  await request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/saml`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function updateAdminApplication(
  csrfToken: string,
  id: string,
  input: UpdateAdminApplicationInput,
): Promise<AdminApplication> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}`,
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
    tenantURL(`/api/admin/v1/applications/${encodeURIComponent(id)}/icon`),
    {
      method: 'POST',
      credentials: 'same-origin',
      cache: 'no-store',
      headers: { 'X-CSRF-Token': csrfToken },
      body: form,
    },
  )
  const body = (await response.json().catch(() => ({}))) as APIError & {
    application?: AdminApplication
  }
  if (!response.ok) {
    throw authenticationAPIError(body, 'Could not upload the icon.')
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
      `/api/admin/v1/applications/${encodeURIComponent(id)}/icon`,
      adminRequest(csrfToken, 'DELETE'),
    )
  ).application
}

export async function deleteAdminApplication(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// Assignments 一覧はまだ「さらに読み込む」UI に移行しておらず (T007 follow-up)、当面は
// PICKER_LIST_LIMIT で切り詰めた1ページのみを表示する。
export async function listApplicationAssignments(id: string): Promise<ApplicationAssignment[]> {
  return (
    await request<{ assignments: ApplicationAssignment[] }>(
      `/api/admin/v1/applications/${encodeURIComponent(id)}/assignments?limit=${PICKER_LIST_LIMIT}`,
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
    `/api/admin/v1/applications/${encodeURIComponent(id)}/assignments`,
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
    `/api/admin/v1/applications/${encodeURIComponent(id)}/assignments/${encodeURIComponent(subjectType)}/${encodeURIComponent(subjectID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function getAppSignInPolicy(id: string): Promise<AppSignInPolicyView> {
  return await request<AppSignInPolicyView>(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/sign-in-policy`,
  )
}

export async function updateAppSignInPolicy(
  csrfToken: string,
  id: string,
  rules: SignInRule[],
): Promise<AppSignInPolicyView> {
  return await request<AppSignInPolicyView>(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/sign-in-policy`,
    adminRequest(csrfToken, 'PUT', { rules }),
  )
}

// テナントデフォルトサインインポリシー (wi-115, ADR-081)。
export async function getTenantDefaultSignInPolicy(): Promise<TenantDefaultSignInPolicyView> {
  return request<TenantDefaultSignInPolicyView>('/api/admin/v1/default-sign-in-policy')
}

export async function updateTenantDefaultSignInPolicy(
  csrfToken: string,
  rules: SignInRule[],
): Promise<TenantDefaultSignInPolicy> {
  return (
    await request<{ policy: TenantDefaultSignInPolicy }>(
      '/api/admin/v1/default-sign-in-policy',
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
      `/api/admin/v1/users/${encodeURIComponent(userID)}/mfa-enrollment-bypass`,
      adminRequest(csrfToken, 'POST', { expires_in_seconds: 900 }),
    )
  ).bypass
}

export async function revokeMfaEnrollmentBypass(csrfToken: string, userID: string): Promise<void> {
  await request(
    `/api/admin/v1/users/${encodeURIComponent(userID)}/mfa-enrollment-bypass`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

// wi-143 / ADR-088 第2層: 管理者による認証器の緊急リセット。削除のみを行い、
// 代わりの factor は登録しない。TOTP / WebAuthn が両方無くなった場合だけ
// reenrollment_required=true になり、単発の enrollment bypass が自動発行される。
export type AuthenticatorResetTarget = 'totp' | 'webauthn' | 'recovery_code'

export type AuthenticatorResetResult = {
  mfa_enrolled: boolean
  reenrollment_required: boolean
  bypass?: MfaEnrollmentBypass
}

export async function resetUserAuthenticators(
  csrfToken: string,
  userID: string,
  targets: AuthenticatorResetTarget[],
): Promise<AuthenticatorResetResult> {
  return await request<AuthenticatorResetResult>(
    `/api/admin/v1/users/${encodeURIComponent(userID)}/authenticator-reset`,
    adminRequest(csrfToken, 'POST', { targets }),
  )
}

// Admin session management (wi-28 T007, ADR-127 決定9): view and revoke a
// target user's sessions. Unlike self-service /api/account/v1/sessions, these
// have no `current` marker and session revoke also cascades to that
// session's refresh tokens server-side (RevokeTokensBySid).
export async function listAdminUserSessions(userID: string): Promise<AdminSessionRecord[]> {
  return (
    await request<{ sessions: AdminSessionRecord[] }>(
      `/api/admin/v1/users/${encodeURIComponent(userID)}/sessions`,
    )
  ).sessions
}

export async function revokeAdminUserSession(
  csrfToken: string,
  userID: string,
  sessionID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/users/${encodeURIComponent(userID)}/sessions/${encodeURIComponent(sessionID)}/revoke`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function revokeAllAdminUserSessions(csrfToken: string, userID: string): Promise<void> {
  await request(
    `/api/admin/v1/users/${encodeURIComponent(userID)}/sessions/revoke_all`,
    adminRequest(csrfToken, 'POST'),
  )
}

// ApplicationCategory の管理 (wi-70, ADR-069)。tenant 単位で定義し Application に付与する。
export async function listApplicationCategories(): Promise<ApplicationCategory[]> {
  return (
    await request<{ categories: ApplicationCategory[] }>('/api/admin/v1/application-categories')
  ).categories
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
      '/api/admin/v1/application-categories',
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
      `/api/admin/v1/application-categories/${encodeURIComponent(categoryID)}`,
      adminRequest(csrfToken, 'PATCH', input),
    )
  ).category
}

export async function deleteApplicationCategory(
  csrfToken: string,
  categoryID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/application-categories/${encodeURIComponent(categoryID)}`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function setApplicationCategories(
  csrfToken: string,
  id: string,
  categoryIDs: string[],
): Promise<AdminApplication> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(id)}/categories`,
    adminRequest(csrfToken, 'PUT', { category_ids: categoryIDs }),
  )
}

export async function listApiTokens(): Promise<ApiToken[]> {
  return (await request<{ tokens: ApiToken[] }>('/api/admin/v1/api-tokens')).tokens
}

export async function createApiToken(
  csrfToken: string,
  input: { description: string; scopes: ApiTokenScope[]; expiry_days: number },
): Promise<{ token: string; meta: ApiToken }> {
  return request('/api/admin/v1/api-tokens', adminRequest(csrfToken, 'POST', input))
}

export async function revokeApiToken(csrfToken: string, id: string): Promise<void> {
  await request(
    `/api/admin/v1/api-tokens/${encodeURIComponent(id)}`,
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
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning`,
    adminRequest(csrfToken, 'POST', input),
  )
}

export async function getAdminApplicationProvisioning(
  applicationID: string,
): Promise<ProvisioningConnection> {
  return request(`/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning`)
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
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning`,
    adminRequest(csrfToken, 'PATCH', input),
  )
}

export async function deleteAdminApplicationProvisioning(
  csrfToken: string,
  applicationID: string,
): Promise<void> {
  await request(
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning`,
    adminRequest(csrfToken, 'DELETE'),
  )
}

export async function testAdminApplicationProvisioning(
  csrfToken: string,
  applicationID: string,
): Promise<ProvisioningTestConnectionResult> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning/test`,
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
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning/on-demand`,
    adminRequest(csrfToken, 'POST', { subject_type: subjectType, subject_id: subjectID }),
  )
}

export async function startAdminApplicationProvisioningFullResync(
  csrfToken: string,
  applicationID: string,
): Promise<{ enqueued_count: number }> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning/full-resync`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function resumeAdminApplicationProvisioning(
  csrfToken: string,
  applicationID: string,
): Promise<ProvisioningConnection> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning/resume`,
    adminRequest(csrfToken, 'POST'),
  )
}

// Deliveries 一覧はまだ「さらに読み込む」UI に移行しておらず (T007 follow-up)、当面は
// PICKER_LIST_LIMIT で切り詰めた1ページのみを表示する。
export async function listAdminApplicationProvisioningDeliveries(
  applicationID: string,
  status?: ProvisioningDeliveryStatus,
): Promise<ProvisioningDelivery[]> {
  const params = new URLSearchParams({ limit: String(PICKER_LIST_LIMIT) })
  if (status) params.set('status', status)
  return (
    await request<{ deliveries: ProvisioningDelivery[] }>(
      `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning/deliveries?${params.toString()}`,
    )
  ).deliveries
}

export async function getAdminApplicationProvisioningDelivery(
  applicationID: string,
  deliveryID: string,
): Promise<ProvisioningDelivery> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning/deliveries/${encodeURIComponent(deliveryID)}`,
  )
}

export async function retryAdminApplicationProvisioningDelivery(
  csrfToken: string,
  applicationID: string,
  deliveryID: string,
): Promise<ProvisioningDelivery> {
  return request(
    `/api/admin/v1/applications/${encodeURIComponent(applicationID)}/provisioning/deliveries/${encodeURIComponent(deliveryID)}/retry`,
    adminRequest(csrfToken, 'POST'),
  )
}

export async function listAdminTenantProvisioningConnections(): Promise<ProvisioningConnection[]> {
  return (
    await request<{ connections: ProvisioningConnection[] }>(
      '/api/admin/v1/provisioning/connections',
    )
  ).connections
}
