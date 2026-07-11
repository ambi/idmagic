import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  base64URLToBytes,
  createPasskey,
  getPasskeyAssertion,
  isWebAuthnSupported,
  preparePublicKeyCreation,
  preparePublicKeyRequest,
  serializeAssertionCredential,
  serializeRegistrationCredential,
} from './webauthn'
import { base64URL } from './core'

const registrationCredential = () => ({
  id: 'credential',
  rawId: new Uint8Array([1, 2, 3]).buffer,
  type: 'public-key',
  getClientExtensionResults: () => ({}),
  response: {
    attestationObject: new Uint8Array([4, 5]).buffer,
    clientDataJSON: new Uint8Array([6, 7]).buffer,
    getTransports: () => ['internal'],
  },
})

const assertionCredential = () => ({
  id: 'credential',
  rawId: new Uint8Array([1, 2, 3]).buffer,
  type: 'public-key',
  getClientExtensionResults: () => ({}),
  response: {
    authenticatorData: new Uint8Array([1]).buffer,
    clientDataJSON: new Uint8Array([2]).buffer,
    signature: new Uint8Array([3]).buffer,
    userHandle: null,
  },
})

describe('base64URLToBytes', () => {
  it('round-trips a base64url-encoded value', () => {
    const original = new Uint8Array([10, 20, 30, 255])
    const encoded = base64URL(original)
    expect(new Uint8Array(base64URLToBytes(encoded))).toEqual(original)
  })
})

describe('preparePublicKeyCreation', () => {
  it('decodes challenge, user id, and exclude credentials into ArrayBuffers', () => {
    const challenge = base64URL(new Uint8Array([1, 2]))
    const userId = base64URL(new Uint8Array([9]))
    const excludeId = base64URL(new Uint8Array([5]))
    const result = preparePublicKeyCreation({
      challenge,
      user: { id: userId, name: 'user', displayName: 'User' },
      excludeCredentials: [{ id: excludeId, type: 'public-key', transports: ['usb'] }],
    })
    expect(new Uint8Array(result.challenge as ArrayBuffer)).toEqual(new Uint8Array([1, 2]))
    expect(new Uint8Array(result.user.id as ArrayBuffer)).toEqual(new Uint8Array([9]))
    expect(result.excludeCredentials).toEqual([
      { id: expect.anything(), type: 'public-key', transports: ['usb'] },
    ])
  })

  it('leaves exclude credentials undefined when not provided', () => {
    const result = preparePublicKeyCreation({
      challenge: base64URL(new Uint8Array([1])),
      user: { id: base64URL(new Uint8Array([1])), name: 'user', displayName: 'User' },
    })
    expect(result.excludeCredentials).toBeUndefined()
  })
})

describe('preparePublicKeyRequest', () => {
  it('decodes challenge and allow credentials into ArrayBuffers', () => {
    const challenge = base64URL(new Uint8Array([3, 4]))
    const allowId = base64URL(new Uint8Array([7]))
    const result = preparePublicKeyRequest({
      challenge,
      allowCredentials: [{ id: allowId, type: 'public-key' }],
    })
    expect(new Uint8Array(result.challenge as ArrayBuffer)).toEqual(new Uint8Array([3, 4]))
    expect(result.allowCredentials).toHaveLength(1)
  })

  it('leaves allow credentials undefined when not provided', () => {
    const result = preparePublicKeyRequest({ challenge: base64URL(new Uint8Array([1])) })
    expect(result.allowCredentials).toBeUndefined()
  })
})

describe('serializeRegistrationCredential', () => {
  it('encodes attestation response fields as base64url', () => {
    const serialized = serializeRegistrationCredential(
      registrationCredential() as unknown as PublicKeyCredential,
    ) as Record<string, unknown>
    expect(serialized).toMatchObject({
      id: 'credential',
      type: 'public-key',
      response: {
        transports: ['internal'],
      },
    })
  })

  it('omits transports when the browser does not report them', () => {
    const credential = registrationCredential()
    credential.response = { ...credential.response, getTransports: undefined as never }
    const serialized = serializeRegistrationCredential(
      credential as unknown as PublicKeyCredential,
    ) as { response: { transports?: string[] } }
    expect(serialized.response.transports).toBeUndefined()
  })
})

describe('serializeAssertionCredential', () => {
  it('encodes assertion response fields as base64url', () => {
    const serialized = serializeAssertionCredential(
      assertionCredential() as unknown as PublicKeyCredential,
    ) as Record<string, unknown>
    expect(serialized).toMatchObject({ id: 'credential', type: 'public-key' })
    expect((serialized.response as { userHandle?: string }).userHandle).toBeUndefined()
  })

  it('encodes a present user handle', () => {
    const credential = assertionCredential() as { response: { userHandle: unknown } }
    credential.response = { ...credential.response, userHandle: new Uint8Array([9]).buffer }
    const serialized = serializeAssertionCredential(
      credential as unknown as PublicKeyCredential,
    ) as { response: { userHandle?: string } }
    expect(serialized.response.userHandle).toBe(base64URL(new Uint8Array([9])))
  })
})

describe('createPasskey', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('returns the serialized attestation on success', async () => {
    vi.stubGlobal('navigator', {
      credentials: { create: vi.fn().mockResolvedValue(registrationCredential()) },
    })
    const result = (await createPasskey({
      publicKey: {
        challenge: base64URL(new Uint8Array([1])),
        user: { id: base64URL(new Uint8Array([1])), name: 'user', displayName: 'User' },
      },
    })) as Record<string, unknown>
    expect(result.id).toBe('credential')
  })

  it('throws when the browser returns no credential', async () => {
    vi.stubGlobal('navigator', { credentials: { create: vi.fn().mockResolvedValue(null) } })
    await expect(
      createPasskey({
        publicKey: {
          challenge: base64URL(new Uint8Array([1])),
          user: { id: base64URL(new Uint8Array([1])), name: 'user', displayName: 'User' },
        },
      }),
    ).rejects.toThrow('パスキーを作成できませんでした。')
  })
})

describe('getPasskeyAssertion', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('returns the serialized assertion on success', async () => {
    vi.stubGlobal('navigator', {
      credentials: { get: vi.fn().mockResolvedValue(assertionCredential()) },
    })
    const result = (await getPasskeyAssertion({
      publicKey: { challenge: base64URL(new Uint8Array([1])) },
    })) as Record<string, unknown>
    expect(result.id).toBe('credential')
  })

  it('throws when the user cancels the authentication', async () => {
    vi.stubGlobal('navigator', { credentials: { get: vi.fn().mockResolvedValue(null) } })
    await expect(
      getPasskeyAssertion({ publicKey: { challenge: base64URL(new Uint8Array([1])) } }),
    ).rejects.toThrow('パスキー認証をキャンセルしました。')
  })
})

describe('isWebAuthnSupported', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('is true when the browser exposes PublicKeyCredential', () => {
    vi.stubGlobal('PublicKeyCredential', class {})
    expect(isWebAuthnSupported()).toBe(true)
  })

  it('is false when PublicKeyCredential is missing', () => {
    expect(isWebAuthnSupported()).toBe(false)
  })
})
