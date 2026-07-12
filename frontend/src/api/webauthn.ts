// WebAuthn / Passkey の JSON <-> ブラウザ API 変換 (wi-26 / ADR-087)。
// サーバー (go-webauthn) は challenge / credential id を base64url 文字列で受け渡すが、
// navigator.credentials.create / get は ArrayBuffer を要求する。ここでその相互変換を担う。
import { base64URL } from './core'

// base64URL 文字列を ArrayBuffer に復号する (base64URL() の逆変換)。
export function base64URLToBytes(value: string): ArrayBuffer {
  const normalized = value.replaceAll('-', '+').replaceAll('_', '/')
  const padding = (4 - (normalized.length % 4)) % 4
  const binary = atob(normalized + '='.repeat(padding))
  const bytes = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i += 1) {
    bytes[i] = binary.charCodeAt(i)
  }
  return bytes.buffer.slice(bytes.byteOffset, bytes.byteOffset + bytes.byteLength)
}

type CredentialDescriptorJSON = {
  id: string
  type: string
  transports?: string[]
}

type CreationOptionsJSON = {
  challenge: string
  user: { id: string; name: string; displayName: string }
  excludeCredentials?: CredentialDescriptorJSON[]
} & Record<string, unknown>

type RequestOptionsJSON = {
  challenge: string
  allowCredentials?: CredentialDescriptorJSON[]
} & Record<string, unknown>

function toDescriptors(
  list?: CredentialDescriptorJSON[],
): PublicKeyCredentialDescriptor[] | undefined {
  return list?.map((c) => ({
    id: base64URLToBytes(c.id),
    type: 'public-key',
    transports: c.transports as AuthenticatorTransport[] | undefined,
  }))
}

// サーバーが返した PublicKeyCredentialCreationOptions(JSON) をブラウザ用に変換する。
export function preparePublicKeyCreation(
  pk: CreationOptionsJSON,
): PublicKeyCredentialCreationOptions {
  return {
    ...(pk as unknown as PublicKeyCredentialCreationOptions),
    challenge: base64URLToBytes(pk.challenge),
    user: {
      ...(pk.user as unknown as PublicKeyCredentialUserEntity),
      id: base64URLToBytes(pk.user.id),
    },
    excludeCredentials: toDescriptors(pk.excludeCredentials),
  }
}

// サーバーが返した PublicKeyCredentialRequestOptions(JSON) をブラウザ用に変換する。
export function preparePublicKeyRequest(pk: RequestOptionsJSON): PublicKeyCredentialRequestOptions {
  return {
    ...(pk as unknown as PublicKeyCredentialRequestOptions),
    challenge: base64URLToBytes(pk.challenge),
    allowCredentials: toDescriptors(pk.allowCredentials),
  }
}

// 登録 (attestation) の結果をサーバーが解釈できる JSON に整形する。
export function serializeRegistrationCredential(credential: PublicKeyCredential): unknown {
  const response = credential.response as AuthenticatorAttestationResponse
  const transports =
    typeof response.getTransports === 'function' ? response.getTransports() : undefined
  return {
    id: credential.id,
    rawId: base64URL(new Uint8Array(credential.rawId)),
    type: credential.type,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      attestationObject: base64URL(new Uint8Array(response.attestationObject)),
      clientDataJSON: base64URL(new Uint8Array(response.clientDataJSON)),
      transports,
    },
  }
}

// 認証 (assertion) の結果をサーバーが解釈できる JSON に整形する。
export function serializeAssertionCredential(credential: PublicKeyCredential): unknown {
  const response = credential.response as AuthenticatorAssertionResponse
  return {
    id: credential.id,
    rawId: base64URL(new Uint8Array(credential.rawId)),
    type: credential.type,
    clientExtensionResults: credential.getClientExtensionResults(),
    response: {
      authenticatorData: base64URL(new Uint8Array(response.authenticatorData)),
      clientDataJSON: base64URL(new Uint8Array(response.clientDataJSON)),
      signature: base64URL(new Uint8Array(response.signature)),
      userHandle: response.userHandle ? base64URL(new Uint8Array(response.userHandle)) : undefined,
    },
  }
}

// navigator.credentials.create を実行し、attestation JSON を返す。
export async function createPasskey(optionsJSON: {
  publicKey: CreationOptionsJSON
}): Promise<unknown> {
  const credential = (await navigator.credentials.create({
    publicKey: preparePublicKeyCreation(optionsJSON.publicKey),
  })) as PublicKeyCredential | null
  if (!credential) {
    throw new Error('Could not create a passkey.')
  }
  return serializeRegistrationCredential(credential)
}

// navigator.credentials.get を実行し、assertion JSON を返す。
export async function getPasskeyAssertion(optionsJSON: {
  publicKey: RequestOptionsJSON
}): Promise<unknown> {
  const credential = (await navigator.credentials.get({
    publicKey: preparePublicKeyRequest(optionsJSON.publicKey),
  })) as PublicKeyCredential | null
  if (!credential) {
    throw new Error('Passkey authentication was cancelled.')
  }
  return serializeAssertionCredential(credential)
}

// ブラウザが WebAuthn (Passkey) をサポートしているか。
export function isWebAuthnSupported(): boolean {
  return typeof window !== 'undefined' && typeof window.PublicKeyCredential !== 'undefined'
}
