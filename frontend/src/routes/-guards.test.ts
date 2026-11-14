import { afterEach, beforeEach, describe, expect, it, mock } from 'bun:test'
import { ensureLoggedIn } from '../api/oidc'
import { restoreGlobals, stubGlobal } from '../test/globals'
import {
  ensureControlPlaneAssurance,
  isControlPlaneAccount,
  type AccountContextResponse,
} from './-guards'

const account = (
  realm: string | undefined,
  roles: string[] | undefined,
): AccountContextResponse => ({
  csrf_token: 'csrf',
  sub: 'operator',
  realm,
  roles,
})

describe('isControlPlaneAccount', () => {
  it('制御面 realm の system_admin だけを許可する', () => {
    expect(isControlPlaneAccount(account('default', ['system_admin']))).toBeTrue()
  })

  it('制御面以外の realm に保存された system_admin を拒否する', () => {
    expect(isControlPlaneAccount(account('acme', ['system_admin']))).toBeFalse()
  })

  it('制御面 realm でも system_admin がなければ拒否する', () => {
    expect(isControlPlaneAccount(account('default', ['admin']))).toBeFalse()
    expect(isControlPlaneAccount(account('default', undefined))).toBeFalse()
  })
})

describe('ensureControlPlaneAssurance', () => {
  const origin = 'https://idp.test'

  beforeEach(async () => {
    sessionStorage.clear()
    stubGlobal('location', {
      origin,
      href: `${origin}/realms/default/system`,
      pathname: '/realms/default/system',
      search: '',
      assign: mock(),
      replace: mock(),
    })
    sessionStorage.setItem(
      'ra_oidc_token_admin',
      JSON.stringify({ accessToken: 'password-only', expiresAt: Date.now() + 120_000 }),
    )
    await ensureLoggedIn('admin', '/realms/default/system')
  })

  afterEach(() => restoreGlobals())

  //spec:covers REQ-SYSTEM-033: システムの経路が MFA の水準を求めると、入場の時点で応答の acr_values を付けて画面全体で再認可し、新しさを求めない読み出しには max_age を付けないこと
  it('requireSystemAccount reauthorizes when the system API requires MFA', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue({
        ok: false,
        status: 401,
        headers: {
          get: (name: string) =>
            name === 'WWW-Authenticate'
              ? 'Bearer error="insufficient_user_authentication", acr_values="urn:idmagic:acr:mfa"'
              : null,
        },
        json: async () => ({ type: 'urn:idmagic:error:insufficient_user_authentication' }),
      }),
    )
    const pending = ensureControlPlaneAssurance('/system/tenants', '?page=2')
    await waitForCall(window.location.replace as ReturnType<typeof mock>)
    const target = new URL(
      String((window.location.replace as ReturnType<typeof mock>).mock.calls[0][0]),
      origin,
    )
    expect(target.pathname).toBe('/realms/default/authorize')
    expect(target.searchParams.get('acr_values')).toBe('urn:idmagic:acr:mfa')
    expect(target.searchParams.has('max_age')).toBeFalse()
    const login = JSON.parse(sessionStorage.getItem('ra_oidc_login') ?? '{}') as {
      returnTo?: string
    }
    expect(login.returnTo).toBe('/realms/default/system/tenants?page=2')
    void pending
  })

  it('passes when the system API accepts the authentication', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue({
        ok: true,
        status: 200,
        headers: { get: () => null },
        json: async () => ({}),
      }),
    )
    await ensureControlPlaneAssurance('/system', '')
    expect(window.location.replace).not.toHaveBeenCalled()
  })
})

async function waitForCall(fn: ReturnType<typeof mock>) {
  for (let i = 0; i < 50 && fn.mock.calls.length === 0; i++) {
    await new Promise((resolve) => setTimeout(resolve, 10))
  }
}
