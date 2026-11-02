const PRIVATE_JWK_MEMBERS = ['d', 'p', 'q', 'dp', 'dq', 'qi', 'k']

export type ParsedPublicKey = { keys: Record<string, unknown>[] } | 'private' | 'unsupported' | null

function fromJson(text: string): ParsedPublicKey {
  let parsed: unknown
  try {
    parsed = JSON.parse(text)
  } catch {
    return null
  }
  if (typeof parsed !== 'object' || parsed === null) return null
  const record = parsed as Record<string, unknown>
  const keys = Array.isArray(record.keys) ? record.keys : 'kty' in record ? [record] : []
  if (keys.length === 0) return null
  if (!keys.every((key) => typeof key === 'object' && key !== null && 'kty' in key)) return null
  const typed = keys as Record<string, unknown>[]
  if (typed.some((key) => PRIVATE_JWK_MEMBERS.some((member) => member in key))) return 'private'
  return { keys: typed }
}

// クライアントアサーションの検証が受け付ける PS256 と ES256 に対応する鍵だけを取り込む。
async function fromSpki(der: Uint8Array<ArrayBuffer>): Promise<ParsedPublicKey> {
  const candidates = [
    { algorithm: { name: 'RSA-PSS', hash: 'SHA-256' }, alg: 'PS256' },
    { algorithm: { name: 'ECDSA', namedCurve: 'P-256' }, alg: 'ES256' },
  ]
  for (const { algorithm, alg } of candidates) {
    try {
      const key = await crypto.subtle.importKey('spki', der, algorithm, true, ['verify'])
      const jwk = await crypto.subtle.exportKey('jwk', key)
      const members =
        jwk.kty === 'RSA' ? { n: jwk.n, e: jwk.e } : { crv: jwk.crv, x: jwk.x, y: jwk.y }
      return { keys: [{ kty: jwk.kty, use: 'sig', alg, ...members }] }
    } catch {
      // 次の鍵の種類を試す。
    }
  }
  return 'unsupported'
}

// parsePublicKey は貼られた公開鍵を JWKS に揃える。PEM、JWK、JWKS を受け付け、秘密鍵は送らない。
export async function parsePublicKey(text: string): Promise<ParsedPublicKey> {
  const input = text.trim()
  if (input.includes('PRIVATE KEY')) return 'private'
  if (!input.startsWith('-----BEGIN')) return fromJson(input)
  const match = input.match(/^-----BEGIN PUBLIC KEY-----([\s\S]+)-----END PUBLIC KEY-----$/)
  if (!match) return 'unsupported'
  try {
    const der = Uint8Array.from(atob(match[1].replace(/\s/g, '')), (char) => char.charCodeAt(0))
    return await fromSpki(der)
  } catch {
    return null
  }
}
