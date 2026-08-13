import type {
  AccountConsent,
  AccountProfile,
  AccountSecurity,
  AccountSession,
  AccountSignInActivity,
  AccountSummary,
  MyApplication,
  PortalCategory,
  TotpEnrollmentStart,
} from '../types'
import { adminRequest, AuthenticationAPIError, request, responseAPIError, tenantURL } from './core'
import { createPasskey, getPasskeyAssertion } from './webauthn'

export type UpdateAccountProfileInput = {
  name?: string
  given_name?: string
  family_name?: string
  attributes?: AccountProfile['attributes']
}

export async function getAccountProfile(): Promise<AccountProfile> {
  return request<AccountProfile>('/api/account/v1/profile')
}

export async function updateAccountProfile(
  csrfToken: string,
  input: UpdateAccountProfileInput,
): Promise<AccountProfile> {
  return request('/api/account/v1/profile', adminRequest(csrfToken, 'PATCH', input))
}

export async function getAccountSummary(): Promise<AccountSummary> {
  return request<AccountSummary>('/api/account/v1/summary')
}

export async function requestEmailChange(csrfToken: string, newEmail: string): Promise<void> {
  const response = await fetch(tenantURL('/api/account/v1/email/change_request'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ new_email: newEmail }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export async function exportAccountData(): Promise<unknown> {
  return request<unknown>('/api/account/v1/data_export')
}

export async function listAccountConsents(): Promise<AccountConsent[]> {
  return (await request<{ consents: AccountConsent[] }>('/api/account/v1/consents')).consents
}

export type AccountApprovalRequest = {
  id: string
  client_id: string
  client_name: string
  agent_name?: string
  scopes: string[]
  authorization_details?: Array<Record<string, unknown> & { type: string }>
  binding_message?: string
  requested_at: string
  expires_at: string
}

export async function listMyApprovalRequests(): Promise<AccountApprovalRequest[]> {
  const body = await request<{ approval_requests: AccountApprovalRequest[] }>(
    '/api/account/v1/approval-requests',
  )
  return body.approval_requests ?? []
}

export async function decideMyApprovalRequest(
  csrfToken: string,
  id: string,
  decision: 'approve' | 'deny',
): Promise<void> {
  const response = await fetch(
    tenantURL(`/api/account/v1/approval-requests/${encodeURIComponent(id)}/decision`),
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
      body: JSON.stringify({ decision }),
      credentials: 'same-origin',
      cache: 'no-store',
    },
  )
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export async function revokeAccountConsent(csrfToken: string, clientId: string): Promise<void> {
  const response = await fetch(
    tenantURL(`/api/account/v1/consents/${encodeURIComponent(clientId)}/revoke`),
    {
      method: 'POST',
      headers: { 'X-CSRF-Token': csrfToken },
      credentials: 'same-origin',
      cache: 'no-store',
    },
  )
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export async function getAccountSecurity(): Promise<AccountSecurity> {
  return request<AccountSecurity>('/api/account/v1/security')
}

export type LinkedIdentity = {
  provider_id: string
  local_user_id: string
  linked_at: string
  last_login_at?: string
}

export async function listLinkedIdentities(): Promise<LinkedIdentity[]> {
  const response = await request<{ identities: LinkedIdentity[] }>(
    '/api/account/v1/linked-identities',
  )
  return response.identities ?? []
}

export async function unlinkIdentity(csrfToken: string, providerId: string): Promise<void> {
  const response = await fetch(
    tenantURL(`/api/account/v1/linked-identities/${encodeURIComponent(providerId)}`),
    {
      method: 'DELETE',
      headers: { 'X-CSRF-Token': csrfToken },
      credentials: 'same-origin',
      cache: 'no-store',
    },
  )
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export async function getSignInActivity(): Promise<AccountSignInActivity[]> {
  return (await request<{ activities: AccountSignInActivity[] }>('/api/account/v1/signin_activity'))
    .activities
}

export async function listAccountSessions(): Promise<AccountSession[]> {
  return (await request<{ sessions: AccountSession[] }>('/api/account/v1/sessions')).sessions
}

export async function revokeAccountSession(csrfToken: string, id: string): Promise<void> {
  const response = await fetch(
    tenantURL(`/api/account/v1/sessions/${encodeURIComponent(id)}/revoke`),
    {
      method: 'POST',
      headers: { 'X-CSRF-Token': csrfToken },
      credentials: 'same-origin',
      cache: 'no-store',
    },
  )
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export async function revokeOtherAccountSessions(csrfToken: string): Promise<void> {
  const response = await fetch(tenantURL('/api/account/v1/sessions/revoke_others'), {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  throw await responseAPIError(response)
}

// step-up 再認証 (wi-43)。高 sensitivity 操作が 403 step_up_required を返したら、
// start で利用可能な factor を取得し、complete で password / TOTP を提示して再認証する。
export type StepUpMethod = 'password' | 'totp' | 'webauthn' | 'recovery_code'

export function isStepUpRequired(cause: unknown): boolean {
  return cause instanceof AuthenticationAPIError && cause.code === 'step_up_required'
}

// step-up 再認証用の WebAuthn assertion challenge を取得し、パスキーで署名した結果を返す。
async function stepUpWebAuthnAssertion(csrfToken: string): Promise<unknown> {
  const response = await fetch(tenantURL('/api/account/v1/step_up/webauthn/challenge'), {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (!response.ok) {
    throw await responseAPIError(response)
  }
  return getPasskeyAssertion((await response.json()) as { publicKey: never })
}

export async function startStepUp(csrfToken: string): Promise<StepUpMethod[]> {
  const response = await fetch(tenantURL('/api/account/v1/step_up/start'), {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.ok) {
    return ((await response.json()) as { methods: StepUpMethod[] }).methods
  }
  throw await responseAPIError(response)
}

export async function completeStepUp(
  csrfToken: string,
  method: StepUpMethod,
  credential: string,
): Promise<void> {
  let payload: Record<string, unknown>
  if (method === 'password') {
    payload = { method, password: credential }
  } else if (method === 'webauthn') {
    // パスキーは challenge 応答型のため、credential 文字列ではなく assertion を送る。
    payload = { method, assertion: await stepUpWebAuthnAssertion(csrfToken) }
  } else {
    // totp / recovery_code はコード入力型。
    payload = { method, code: credential }
  }
  const response = await fetch(tenantURL('/api/account/v1/step_up/complete'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify(payload),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export async function startTotpEnrollment(csrfToken: string): Promise<TotpEnrollmentStart> {
  const response = await fetch(tenantURL('/api/account/v1/mfa/totp/enroll/start'), {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.ok) return (await response.json()) as TotpEnrollmentStart
  throw await responseAPIError(response)
}

export async function confirmTotpEnrollment(
  csrfToken: string,
  secret: string,
  code: string,
): Promise<void> {
  const response = await fetch(tenantURL('/api/account/v1/mfa/totp/enroll/confirm'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ secret, code }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export async function removeTotpFactor(csrfToken: string, code: string): Promise<void> {
  const response = await fetch(tenantURL('/api/account/v1/mfa/totp/remove'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ code }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  throw await responseAPIError(response)
}

// registerPasskey は登録 challenge を取得し、navigator.credentials.create で作成した
// パスキーを attestation としてサーバーに登録する (wi-26)。
export async function registerPasskey(csrfToken: string, label?: string): Promise<void> {
  const startResponse = await fetch(tenantURL('/api/account/v1/mfa/webauthn/register/start'), {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (!startResponse.ok) {
    throw await responseAPIError(startResponse)
  }
  const attestation = await createPasskey((await startResponse.json()) as { publicKey: never })
  const finishResponse = await fetch(tenantURL('/api/account/v1/mfa/webauthn/register/finish'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ attestation, label: label?.trim() ? label.trim() : undefined }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (finishResponse.status === 204) return
  throw await responseAPIError(finishResponse)
}

export async function removePasskey(csrfToken: string, credentialId: string): Promise<void> {
  const response = await fetch(tenantURL('/api/account/v1/mfa/webauthn/remove'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ credential_id: credentialId }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export type RecoveryCodesResult = {
  codes: string[]
  generated_at: string
}

export async function generateRecoveryCodes(csrfToken: string): Promise<RecoveryCodesResult> {
  const response = await fetch(tenantURL('/api/account/v1/mfa/recovery-codes/generate'), {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.ok) return (await response.json()) as RecoveryCodesResult
  throw await responseAPIError(response)
}

export async function revokeRecoveryCodes(csrfToken: string): Promise<void> {
  const response = await fetch(tenantURL('/api/account/v1/mfa/recovery-codes/revoke'), {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  throw await responseAPIError(response)
}

export async function confirmEmailChange(csrfToken: string, token: string): Promise<void> {
  const response = await fetch(tenantURL('/api/account/v1/email/verify'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ token }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.ok) return
  throw await responseAPIError(response)
}

// 利用者ポータルの割当済みアプリ一覧とカテゴリ定義 (wi-69, wi-70)。visible 割当のみ返り、
// categories は管理者定義のセクション見出しを position 昇順で含む。
export type MyPortal = {
  applications: MyApplication[]
  categories: PortalCategory[]
}

export async function listMyApplications(): Promise<MyPortal> {
  const body = await request<{ applications: MyApplication[]; categories: PortalCategory[] }>(
    '/api/account/v1/applications',
  )
  return { applications: body.applications, categories: body.categories ?? [] }
}

// 利用者ごとの手動並び順 (wi-70)。未保存なら空配列が返る。
export async function getMyApplicationOrder(): Promise<string[]> {
  return (await request<{ application_ids: string[] }>('/api/account/v1/applications/order'))
    .application_ids
}

export async function reorderMyApplications(
  csrfToken: string,
  applicationIds: string[],
): Promise<string[]> {
  return (
    await request<{ application_ids: string[] }>(
      '/api/account/v1/applications/order',
      adminRequest(csrfToken, 'PUT', { application_ids: applicationIds }),
    )
  ).application_ids
}
