import type { BrowserFlowResponse } from '../types'
import { AuthenticationAPIError, base64URL, request, tenantURL, type APIError } from './core'
import { getPasskeyAssertion } from './webauthn'
import { commonDictionary } from '../lib/i18n/common.i18n'
import { getCurrentLocale } from '../lib/i18n/currentLocale'

const uiFallback = () => commonDictionary[getCurrentLocale()].networkError

export async function login(
  csrfToken: string,
  username: string,
  password: string,
  returnTo?: string,
): Promise<BrowserFlowResponse> {
  return request('/api/auth/login', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify({ username, password, return_to: returnTo }),
  })
}

export async function submitConsent(
  csrfToken: string,
  action: 'allow' | 'deny',
): Promise<BrowserFlowResponse> {
  return request('/api/auth/consent', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify({ action }),
  })
}

export async function submitTOTP(
  csrfToken: string,
  code: string,
  returnTo?: string,
): Promise<BrowserFlowResponse> {
  return request('/api/auth/totp', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify({ code, return_to: returnTo }),
  })
}

// loginWithPasskey は login 第二要素の WebAuthn フロー: challenge 取得 →
// navigator.credentials.get → assertion 検証 (wi-26 / ADR-087)。
export async function loginWithPasskey(
  csrfToken: string,
  returnTo?: string,
): Promise<BrowserFlowResponse> {
  const challengeResponse = await fetch(tenantURL('/api/auth/webauthn/challenge'), {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (!challengeResponse.ok) {
    const body = (await challengeResponse.json().catch(() => ({}))) as APIError
    throw new AuthenticationAPIError(body.message ?? uiFallback(), body.error)
  }
  const assertion = await getPasskeyAssertion(
    (await challengeResponse.json()) as { publicKey: never },
  )
  return request('/api/auth/webauthn', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ assertion, return_to: returnTo }),
  })
}

export async function submitRecoveryCode(
  csrfToken: string,
  code: string,
  returnTo?: string,
): Promise<BrowserFlowResponse> {
  return request('/api/auth/recovery-code', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify({ code, return_to: returnTo }),
  })
}

export type MfaEnrollmentStart = {
  secret: string
  otpauth_uri: string
  account_name: string
  issuer: string
}

export async function startMfaEnrollment(csrfToken: string): Promise<MfaEnrollmentStart> {
  return request('/api/auth/mfa/enrollment/totp/start', {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrfToken },
  })
}

export async function confirmMfaEnrollment(
  csrfToken: string,
  secret: string,
  code: string,
  returnTo?: string,
): Promise<BrowserFlowResponse> {
  return request('/api/auth/mfa/enrollment/totp/confirm', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ secret, code, return_to: returnTo }),
  })
}

export class PasswordPolicyError extends AuthenticationAPIError {
  violations: string[]

  constructor(message: string, violations: string[]) {
    super(message, 'password_policy')
    this.name = 'PasswordPolicyError'
    this.violations = violations
  }
}

export async function changePassword(
  csrfToken: string,
  currentPassword: string,
  newPassword: string,
): Promise<void> {
  const response = await fetch(tenantURL('/api/auth/change_password'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify({ current_password: currentPassword, new_password: newPassword }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  const body = (await response.json().catch(() => ({}))) as {
    error?: string
    message?: string
    violations?: string[]
  }
  if (body.error === 'password_policy') {
    throw new PasswordPolicyError(body.message ?? uiFallback(), body.violations ?? [])
  }
  throw new AuthenticationAPIError(body.message ?? uiFallback(), body.error)
}

export async function requestPasswordReset(csrfToken: string, email: string): Promise<void> {
  const response = await fetch(tenantURL('/api/auth/forgot_password'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ email }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.status === 204) return
  const body = (await response.json().catch(() => ({}))) as APIError
  throw new AuthenticationAPIError(body.message ?? uiFallback(), body.error)
}

export async function resetPassword(
  csrfToken: string,
  token: string,
  newPassword: string,
): Promise<void> {
  const response = await fetch(tenantURL('/api/auth/reset_password'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrfToken },
    body: JSON.stringify({ token, new_password: newPassword }),
    credentials: 'same-origin',
    cache: 'no-store',
  })
  if (response.ok) return
  const body = (await response.json().catch(() => ({}))) as APIError & { violations?: string[] }
  if (body.error === 'password_policy') {
    throw new PasswordPolicyError(body.message ?? uiFallback(), body.violations ?? [])
  }
  throw new AuthenticationAPIError(body.message ?? uiFallback(), body.error)
}

export async function submitDevice(
  csrfToken: string,
  userCode: string,
  action: 'allow' | 'deny',
): Promise<BrowserFlowResponse> {
  return request('/api/auth/device', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-CSRF-Token': csrfToken,
    },
    body: JSON.stringify({ user_code: userCode, action }),
  })
}
export function continueBrowserFlow(result: BrowserFlowResponse) {
  const destination = result.redirect_to ?? result.next
  if (!destination) {
    throw new AuthenticationAPIError(uiFallback())
  }
  window.location.assign(destination)
}

export async function startDemoAuthorization() {
  const verifierBytes = crypto.getRandomValues(new Uint8Array(32))
  const verifier = base64URL(verifierBytes)
  const digest = await crypto.subtle.digest('SHA-256', new TextEncoder().encode(verifier))
  const state = base64URL(crypto.getRandomValues(new Uint8Array(16)))
  const nonce = base64URL(crypto.getRandomValues(new Uint8Array(16)))

  sessionStorage.setItem('idmagic-demo-code-verifier', verifier)
  const parameters = new URLSearchParams({
    response_type: 'code',
    // demo-client の固定 UUID (ADR-084)。bootstrap seed と一致させる。
    client_id: '00000000-0000-4000-8000-000000000021',
    redirect_uri: `${window.location.origin}${tenantURL('/callback')}`,
    scope: 'openid profile email offline_access',
    state,
    nonce,
    code_challenge: base64URL(new Uint8Array(digest)),
    code_challenge_method: 'S256',
  })
  window.location.assign(`${tenantURL('/authorize')}?${parameters.toString()}`)
}
