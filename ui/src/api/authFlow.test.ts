import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import {
  changePassword,
  continueBrowserFlow,
  login,
  PasswordPolicyError,
  requestPasswordReset,
  resetPassword,
  submitConsent,
  submitDevice,
  submitRecoveryCode,
  submitTOTP,
} from './authFlow'
import { AuthenticationAPIError } from './core'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: vi.fn().mockResolvedValue(body),
})

describe('auth flow API client', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(200, { next: '/next' })))
  })

  afterEach(() => vi.unstubAllGlobals())

  it('submits browser-flow forms with CSRF protection and returns the destination', async () => {
    await expect(login('csrf', 'alice', 'secret', '/continue')).resolves.toEqual({ next: '/next' })
    await expect(submitConsent('csrf', 'allow')).resolves.toEqual({ next: '/next' })
    await expect(submitTOTP('csrf', '123456')).resolves.toEqual({ next: '/next' })
    await expect(submitRecoveryCode('csrf', 'recovery')).resolves.toEqual({ next: '/next' })
    await expect(submitDevice('csrf', 'device-code', 'deny')).resolves.toEqual({ next: '/next' })

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/login'),
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ 'X-CSRF-Token': 'csrf' }),
        body: JSON.stringify({ username: 'alice', password: 'secret', return_to: '/continue' }),
      }),
    )
  })

  it('sends password changes and accepts a no-content response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(204)))
    await expect(changePassword('csrf', 'old', 'new')).resolves.toBeUndefined()
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/auth/change_password'),
      expect.objectContaining({
        body: JSON.stringify({ current_password: 'old', new_password: 'new' }),
      }),
    )
  })

  it('exposes password-policy violations from change and reset requests', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        response(400, {
          error: 'password_policy',
          message: 'weak password',
          violations: ['length'],
        }),
      ),
    )
    await expect(changePassword('csrf', 'old', 'new')).rejects.toMatchObject({
      name: 'PasswordPolicyError',
      violations: ['length'],
    })
    await expect(resetPassword('csrf', 'reset-token', 'new')).rejects.toBeInstanceOf(
      PasswordPolicyError,
    )
  })

  it('reports ordinary failed password-reset requests as API errors', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(500, { error: 'unavailable' })))
    await expect(requestPasswordReset('csrf', 'alice@example.com')).rejects.toBeInstanceOf(
      AuthenticationAPIError,
    )
  })

  it('redirects to next or redirect_to and rejects a malformed result', () => {
    const assign = vi.fn()
    vi.stubGlobal('location', { ...window.location, assign })
    continueBrowserFlow({ next: '/next' })
    continueBrowserFlow({ redirect_to: '/redirect' })
    expect(assign).toHaveBeenNthCalledWith(1, '/next')
    expect(assign).toHaveBeenNthCalledWith(2, '/redirect')
    expect(() => continueBrowserFlow({})).toThrow(AuthenticationAPIError)
  })
})
