import { describe, expect, it } from 'bun:test'
import { isPortalApiCallTarget, unmarkedPortalApiFetches } from './portal-api-calls.ts'

describe('isPortalApiCallTarget', () => {
  it('targets production frontend sources other than the shared request path', () => {
    expect(isPortalApiCallTarget('frontend/src/api/account.ts')).toBe(true)
    expect(isPortalApiCallTarget('frontend/src/features/account/AccountSecurityPage.tsx')).toBe(
      true,
    )
  })

  it('skips tests, the shared request path, and code outside frontend/src', () => {
    expect(isPortalApiCallTarget('frontend/src/api/account.test.ts')).toBe(false)
    expect(isPortalApiCallTarget('frontend/src/api/core.ts')).toBe(false)
    expect(isPortalApiCallTarget('frontend/tests/e2e/ui-scenario-actions.spec.ts')).toBe(false)
  })
})

describe('unmarkedPortalApiFetches', () => {
  it('reports a direct fetch of the account and admin APIs', () => {
    const source = [
      'export async function removeTotpFactor() {',
      "  const response = await fetch(tenantURL('/api/account/v1/mfa/totp/remove'), {",
      '  })',
      '}',
      'export async function upload() {',
      '  const response = await fetch(',
      '    tenantURL(`/api/admin/v1/tenant/branding/assets/logo`),',
      '  )',
      '}',
      "await fetch(tenantURL('/api/auth/change-password'), {})",
    ].join('\n')
    expect(unmarkedPortalApiFetches(source)).toEqual([
      { line: 2, problem: 'unmarked' },
      { line: 6, problem: 'unmarked' },
      { line: 10, problem: 'unmarked' },
    ])
  })

  it('accepts a cookie call marked with a reason and reports a marker without one', () => {
    const source = [
      '// session-cookie: どれが現在のセッションかを Cookie で決める。',
      'export async function listAccountSessions() {',
      "  const response = await fetch(tenantURL('/api/account/v1/sessions'), {})",
      '}',
      '// session-cookie:',
      'export async function confirmEmailChange() {',
      "  const response = await fetch(tenantURL('/api/account/v1/email/verify'), {})",
      '}',
    ].join('\n')
    expect(unmarkedPortalApiFetches(source)).toEqual([{ line: 7, problem: 'missing-reason' }])
  })

  it('leaves login, token, and non-portal requests alone', () => {
    const source = [
      "await fetch(tenantURL('/api/auth/login'), {})",
      "await fetch(tenantURL('/token'), {})",
      "await fetch(tenantURL('/api/auth/negotiate'), {})",
    ].join('\n')
    expect(unmarkedPortalApiFetches(source)).toEqual([])
  })
})
