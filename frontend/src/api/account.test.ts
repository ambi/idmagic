import { afterEach, beforeEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import {
  confirmEmailChange,
  confirmTotpEnrollment,
  generateRecoveryCodes,
  isStepUpRequired,
  listAccountSessions,
  registerPasskey,
  removePasskey,
  removeTotpFactor,
  requestEmailChange,
  revokeAccountConsent,
  revokeAccountSession,
  revokeOtherAccountSessions,
  revokeRecoveryCodes,
  startTotpEnrollment,
} from './account'
import { setBearerTokenProvider, StepUpRequiredError, UnauthenticatedError } from './core'

const response = (status: number, body: unknown = {}) => ({
  ok: status >= 200 && status < 300,
  status,
  json: mock().mockResolvedValue(body),
})

const passkey = (kind: 'assertion' | 'registration') => ({
  id: 'credential',
  rawId: new Uint8Array([1]).buffer,
  type: 'public-key',
  getClientExtensionResults: () => ({}),
  response:
    kind === 'assertion'
      ? {
          authenticatorData: new Uint8Array([1]).buffer,
          clientDataJSON: new Uint8Array([2]).buffer,
          signature: new Uint8Array([3]).buffer,
          userHandle: null,
        }
      : {
          attestationObject: new Uint8Array([1]).buffer,
          clientDataJSON: new Uint8Array([2]).buffer,
          getTransports: () => ['internal'],
        },
})

describe('account API client', () => {
  beforeEach(() => stubGlobal('fetch', mock().mockResolvedValue(response(204))))
  afterEach(() => {
    setBearerTokenProvider(() => null)
    restoreGlobals()
  })

  //spec:covers REQ-AUTHENTICATION-013: 一覧と失効が同じブラウザーセッションを見ること
  it('セッション一覧は Bearer があっても失効操作と同じ Cookie 認証を使う', async () => {
    setBearerTokenProvider(() => 'portal-token')
    const sessions = [
      {
        id: 'current',
        current: true,
        amr: ['pwd'],
        acr: 'urn:idmagic:acr:password',
        started_at: '2026-10-31T00:00:00Z',
        expires_at: '2026-10-31T01:00:00Z',
      },
    ]
    const fetcher = mock((_url: RequestInfo | URL, _options?: RequestInit) =>
      Promise.resolve(response(200, { sessions })),
    )
    stubGlobal('fetch', fetcher)
    expect(await listAccountSessions()).toEqual(sessions)
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/account/v1/sessions'),
      expect.objectContaining({ credentials: 'same-origin', cache: 'no-store' }),
    )
    const options = fetcher.mock.calls[0]?.[1]
    expect(new Headers(options?.headers).has('Authorization')).toBe(false)
  })

  it('sends protected no-content account operations to their endpoint', async () => {
    await requestEmailChange('csrf', 'new@example.com')
    await revokeAccountConsent('csrf', 'client/id')
    await revokeAccountSession('csrf', 'session/id')
    await revokeOtherAccountSessions('csrf')
    await confirmTotpEnrollment('csrf', 'secret', '123456')
    await removeTotpFactor('csrf', '123456')
    await removePasskey('csrf', 'credential')
    await revokeRecoveryCodes('csrf')
    await confirmEmailChange('csrf', 'verification-token')

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/account/v1/email/change-request'),
      expect.objectContaining({ body: JSON.stringify({ new_email: 'new@example.com' }) }),
    )
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/account/v1/consents/client%2Fid/revoke'),
      expect.objectContaining({ headers: expect.objectContaining({ 'X-CSRF-Token': 'csrf' }) }),
    )
  })

  it('handles enrollment, passkey registration, and recovery-code output', async () => {
    stubGlobal('navigator', {
      credentials: { create: mock().mockResolvedValue(passkey('registration')) },
    })
    stubGlobal(
      'fetch',
      mock()
        .mockResolvedValueOnce(response(200, { secret: 'secret', qr_code: 'qr' }))
        .mockResolvedValueOnce(
          response(200, {
            publicKey: { challenge: 'AA', user: { id: 'AQ', name: 'alice', displayName: 'Alice' } },
          }),
        )
        .mockResolvedValueOnce(response(204))
        .mockResolvedValueOnce(response(200, { codes: ['one'], generated_at: 'now' })),
    )
    await expect(startTotpEnrollment('csrf')).resolves.toMatchObject({ secret: 'secret' })
    await registerPasskey('csrf', ' Laptop ')
    await expect(generateRecoveryCodes('csrf')).resolves.toEqual({
      codes: ['one'],
      generated_at: 'now',
    })
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/register/finish'),
      expect.anything(),
    )
  })

  //spec:covers REQ-AUTHENTICATION-039: 機微な操作を Bearer で呼び、再認証の要求を RFC 9470 の 401 と Cookie の 403 から読み取り、invalid_token の 401 はトークンの失効として残すこと
  it('sends sensitive operations with the bearer and recognizes step-up challenges', async () => {
    setBearerTokenProvider(() => 'portal-token')
    const challenged = (status: number, body: unknown, header?: string) => ({
      ...response(status, body),
      headers: { get: (name: string) => (name === 'WWW-Authenticate' ? (header ?? null) : null) },
    })
    stubGlobal(
      'fetch',
      mock()
        .mockResolvedValueOnce(
          challenged(
            401,
            { type: 'urn:idmagic:error:insufficient_user_authentication' },
            'Bearer error="insufficient_user_authentication", max_age="120", acr_values="urn:idmagic:acr:mfa"',
          ),
        )
        .mockResolvedValueOnce(challenged(403, { type: 'urn:idmagic:error:step_up_required' }))
        .mockResolvedValueOnce(
          challenged(
            401,
            { type: 'urn:idmagic:error:invalid_token' },
            'Bearer error="invalid_token"',
          ),
        ),
    )

    const bearer = await removeTotpFactor('csrf', '123456').catch((cause: unknown) => cause)
    expect(fetch).toHaveBeenLastCalledWith(
      expect.stringContaining('/api/account/v1/mfa/totp/remove'),
      expect.objectContaining({
        headers: expect.objectContaining({ Authorization: 'Bearer portal-token' }),
      }),
    )
    expect(bearer).toBeInstanceOf(StepUpRequiredError)
    expect(isStepUpRequired(bearer)).toBe(true)
    expect((bearer as StepUpRequiredError).challenge).toEqual({
      maxAge: 120,
      acrValues: 'urn:idmagic:acr:mfa',
    })

    const cookie = await revokeOtherAccountSessions('csrf').catch((cause: unknown) => cause)
    expect(isStepUpRequired(cookie)).toBe(true)
    expect((cookie as StepUpRequiredError).challenge).toEqual({ maxAge: 300 })

    const expired = await removeTotpFactor('csrf', '123456').catch((cause: unknown) => cause)
    expect(expired).toBeInstanceOf(UnauthenticatedError)
    expect(isStepUpRequired(expired)).toBe(false)
    expect(isStepUpRequired(new Error('other'))).toBe(false)
  })
})
