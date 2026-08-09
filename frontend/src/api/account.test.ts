import { afterEach, beforeEach, describe, expect, it, mock } from 'bun:test'
import { restoreGlobals, stubGlobal } from '../test/globals'
import {
  completeStepUp,
  confirmEmailChange,
  confirmTotpEnrollment,
  generateRecoveryCodes,
  isStepUpRequired,
  registerPasskey,
  removePasskey,
  removeTotpFactor,
  requestEmailChange,
  revokeAccountConsent,
  revokeAccountSession,
  revokeOtherAccountSessions,
  revokeRecoveryCodes,
  startStepUp,
  startTotpEnrollment,
} from './account'
import { AuthenticationAPIError } from './core'

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
  afterEach(() => restoreGlobals())

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
      expect.stringContaining('/api/account/v1/email/change_request'),
      expect.objectContaining({ body: JSON.stringify({ new_email: 'new@example.com' }) }),
    )
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/account/v1/consents/client%2Fid/revoke'),
      expect.objectContaining({ headers: expect.objectContaining({ 'X-CSRF-Token': 'csrf' }) }),
    )
  })

  it('returns available step-up methods and serializes each credential form', async () => {
    stubGlobal(
      'fetch',
      mock()
        .mockResolvedValueOnce(response(200, { methods: ['password', 'totp'] }))
        .mockResolvedValue(response(204)),
    )
    await expect(startStepUp('csrf')).resolves.toEqual(['password', 'totp'])
    await completeStepUp('csrf', 'password', 'secret')
    await completeStepUp('csrf', 'totp', '123456')
    await completeStepUp('csrf', 'recovery_code', 'recovery')
    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/account/v1/step_up/complete'),
      expect.objectContaining({ body: JSON.stringify({ method: 'password', password: 'secret' }) }),
    )
  })

  it('obtains a WebAuthn assertion before completing step-up', async () => {
    stubGlobal('navigator', {
      credentials: { get: mock().mockResolvedValue(passkey('assertion')) },
    })
    stubGlobal(
      'fetch',
      mock()
        .mockResolvedValueOnce(response(200, { publicKey: { challenge: 'AA' } }))
        .mockResolvedValueOnce(response(204)),
    )
    await completeStepUp('csrf', 'webauthn', '')
    expect(fetch).toHaveBeenLastCalledWith(
      expect.stringContaining('/api/account/v1/step_up/complete'),
      expect.objectContaining({ body: expect.stringContaining('"method":"webauthn"') }),
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

  it('turns API failures into a typed error and identifies step-up requirements', async () => {
    stubGlobal(
      'fetch',
      mock().mockResolvedValue(
        response(403, {
          type: 'urn:idmagic:error:step_up_required',
          title: 'Step up required',
          detail: 'Reauthentication is required.',
        }),
      ),
    )
    await expect(startStepUp('csrf')).rejects.toMatchObject({
      message: 'Reauthentication is required.',
      code: 'step_up_required',
    })
    expect(isStepUpRequired(new AuthenticationAPIError('step up', 'step_up_required'))).toBe(true)
    expect(isStepUpRequired(new Error('other'))).toBe(false)
  })
})
