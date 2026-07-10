import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  completeLoginFromCallback,
  currentBearer,
  ensureLoggedIn,
  logout,
  markPortalAuthenticated,
  recoverPortalSession,
} from './oidc'

const accountKey = 'ra_oidc_token_account'

describe('portal OIDC client', () => {
  const originalLocation = window.location

  beforeEach(() => {
    sessionStorage.clear()
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        json: async () => ({
          access_token: 'renewed',
          refresh_token: 'refresh',
          expires_in: 600,
        }),
      }),
    )
    vi.stubGlobal('location', {
      ...originalLocation,
      pathname: '/realms/acme/account',
      search: '',
      assign: vi.fn(),
    })
  })

  afterEach(() => vi.unstubAllGlobals())

  it('uses a fresh session as the bearer without a network request', async () => {
    sessionStorage.setItem(
      accountKey,
      JSON.stringify({ accessToken: 'fresh', expiresAt: Date.now() + 120_000 }),
    )
    await ensureLoggedIn('account', '/account')
    expect(currentBearer()).toBe('fresh')
    expect(fetch).not.toHaveBeenCalled()
  })

  it('refreshes an expired session and persists the replacement token', async () => {
    sessionStorage.setItem(
      accountKey,
      JSON.stringify({ accessToken: 'old', refreshToken: 'refresh', expiresAt: 0 }),
    )
    await ensureLoggedIn('account', '/account')
    expect(currentBearer()).toBe('renewed')
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/token'),
      expect.objectContaining({
        body: expect.stringContaining('grant_type=refresh_token'),
      }),
    )
    expect(JSON.parse(sessionStorage.getItem(accountKey) ?? '{}')).toMatchObject({
      accessToken: 'renewed',
    })
  })

  it('exchanges a valid callback and redirects to its safe return path', async () => {
    sessionStorage.setItem(
      'ra_oidc_login',
      JSON.stringify({
        state: 'expected',
        verifier: 'verifier',
        audience: 'account',
        returnTo: '/account/profile',
      }),
    )
    vi.stubGlobal('location', {
      ...originalLocation,
      pathname: '/realms/acme/callback',
      search: '?code=code&state=expected',
      assign: vi.fn(),
    })
    await expect(completeLoginFromCallback()).resolves.toBe(true)
    expect(fetch).toHaveBeenCalledWith(expect.stringContaining('/token'), expect.anything())
    expect(window.location.assign).toHaveBeenCalledWith('/account/profile')
  })

  it('rejects invalid callback state and recognizes non-portal callbacks', async () => {
    await expect(completeLoginFromCallback()).resolves.toBe(false)
    sessionStorage.setItem(
      'ra_oidc_login',
      JSON.stringify({ state: 'expected', verifier: 'v', audience: 'admin', returnTo: '/admin' }),
    )
    vi.stubGlobal('location', {
      ...originalLocation,
      search: '?code=code&state=wrong',
      assign: vi.fn(),
    })
    await expect(completeLoginFromCallback()).rejects.toThrow('state')
  })

  it('suppresses repeated 401 recovery and clears stale state', async () => {
    sessionStorage.setItem('ra_oidc_reauth_account', String(Date.now()))
    sessionStorage.setItem(accountKey, JSON.stringify({ accessToken: 'stale', expiresAt: 0 }))
    await expect(recoverPortalSession('account', '/account')).resolves.toBe(false)
    expect(sessionStorage.getItem(accountKey)).toBeNull()
    markPortalAuthenticated('account')
    expect(sessionStorage.getItem('ra_oidc_reauth_account')).toBeNull()
  })

  it('revokes the saved token before ending the portal session', async () => {
    sessionStorage.setItem(
      accountKey,
      JSON.stringify({ accessToken: 'access', refreshToken: 'refresh', expiresAt: 0 }),
    )
    await logout('account')
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/revoke'),
      expect.objectContaining({ method: 'POST' }),
    )
    expect(window.location.assign).toHaveBeenCalledWith(expect.stringContaining('/end_session'))
  })
})
