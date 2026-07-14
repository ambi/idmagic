import { request, tenantURL, validReturnTo } from '../api/core'
import type { ConsentDetailView } from '../types'
import { PageMarker } from './-page'
import { ConsentPage as ConsentPageComponent } from '../features/auth-flow/ConsentPage'
import { LoginPage as LoginPageComponent } from '../features/auth-flow/LoginPage'
import { TotpPage as TotpPageComponent } from '../features/auth-flow/TotpPage'
import { MfaEnrollmentPage as MfaEnrollmentPageComponent } from '../features/auth-flow/MfaEnrollmentPage'

type TransactionResponse = {
  kind: 'login' | 'totp' | 'mfa_enrollment' | 'consent'
  csrf_token: string
  client_name?: string
  scopes?: string[]
  authorization_details?: ConsentDetailView[]
  second_factor_methods?: string[]
}

export type BrowserFlowPage =
  | {
      kind: 'login'
      csrfToken: string
      returnTo?: string
    }
  | {
      kind: 'totp'
      csrfToken: string
      returnTo?: string
      secondFactorMethods: string[]
    }
  | {
      kind: 'consent'
      csrfToken: string
      clientName: string
      scopes: string[]
      authorizationDetails: ConsentDetailView[]
    }
  | {
      kind: 'mfa_enrollment'
      csrfToken: string
      returnTo?: string
    }

export async function loadBrowserFlowData(path: string, search: string): Promise<BrowserFlowPage> {
  const requestedReturnTo = new URLSearchParams(search).get('return_to') ?? ''
  const returnTo = requestedReturnTo
    ? validReturnTo(requestedReturnTo)
      ? requestedReturnTo
      : tenantURL('/admin')
    : undefined
  const transactionURL = returnTo
    ? `/api/auth/transaction?return_to=${encodeURIComponent(returnTo)}`
    : '/api/auth/transaction'
  const data = await request<TransactionResponse>(transactionURL)
  if (data.kind === 'consent') {
    if (path !== '/consent') {
      window.history.replaceState(null, '', tenantURL('/consent'))
    }
    return {
      kind: 'consent',
      csrfToken: data.csrf_token,
      clientName: data.client_name ?? '',
      scopes: data.scopes ?? [],
      authorizationDetails: data.authorization_details ?? [],
    }
  }
  if (data.kind === 'totp') {
    if (path !== '/totp') {
      window.history.replaceState(null, '', tenantURL('/totp'))
    }
    return {
      kind: 'totp',
      csrfToken: data.csrf_token,
      returnTo,
      secondFactorMethods: data.second_factor_methods ?? ['totp'],
    }
  }
  if (data.kind === 'mfa_enrollment') {
    if (path !== '/mfa-enrollment') {
      window.history.replaceState(null, '', tenantURL('/mfa-enrollment'))
    }
    return { kind: 'mfa_enrollment', csrfToken: data.csrf_token, returnTo }
  }
  if (path !== '/login') {
    window.history.replaceState(null, '', tenantURL('/login'))
  }
  return { kind: 'login', csrfToken: data.csrf_token, returnTo }
}

export function BrowserFlowRoute({ data }: { data: BrowserFlowPage }) {
  return (
    <PageMarker kind={data.kind}>
      {data.kind === 'consent' ? (
        <ConsentPageComponent {...data} />
      ) : data.kind === 'mfa_enrollment' ? (
        <MfaEnrollmentPageComponent {...data} />
      ) : data.kind === 'totp' ? (
        <TotpPageComponent {...data} />
      ) : (
        <LoginPageComponent {...data} />
      )}
    </PageMarker>
  )
}
