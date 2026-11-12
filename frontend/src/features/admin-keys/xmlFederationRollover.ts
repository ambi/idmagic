import type { AdminKey, EntraFederationVerification } from '../../types'

// RolloverEntraDomain は、共有 XML 鍵を信頼する一つの Entra のドメインと、その照合の記録である。
export type RolloverEntraDomain = {
  domain: string
  verification: EntraFederationVerification
  updateCommand: string
}

// RolloverTrust は、IdMagic が登録を確かめられず、管理者の確認だけに頼る信頼先である。
export type RolloverTrust = { id: string; kind: 'wsfed' | 'saml'; label: string }

export type RolloverSnapshot = {
  keys: AdminKey[]
  entraDomains: RolloverEntraDomain[]
  trusts: RolloverTrust[]
}

export type RolloverConfirmations = { trusts: ReadonlySet<string>; overall: boolean }

export type ActivationBlocker =
  | { kind: 'no_prepared_key' }
  | { kind: 'entra_not_matched'; domain: string }
  | { kind: 'trust_unconfirmed'; trust: RolloverTrust }
  | { kind: 'overall_unconfirmed' }

export function activeKey(keys: AdminKey[]): AdminKey | undefined {
  return keys.find((key) => key.active)
}

export function preparedKey(keys: AdminKey[]): AdminKey | undefined {
  return keys.find((key) => key.prepared && !key.active)
}

export function verifyingKeys(keys: AdminKey[]): AdminKey[] {
  return keys.filter((key) => !key.active && !key.prepared)
}

// entraConfirmsPrepared は、ドメインの照合が一致し、比べた次の証明書が今の準備済みの鍵であることを確かめる。
// 一致の状態だけを見ると、取り消した以前の準備済みの鍵で照合した記録を確認済みとして扱ってしまう。
export function entraConfirmsPrepared(domain: RolloverEntraDomain, prepared: AdminKey): boolean {
  const expected = prepared.certificate_fingerprint_sha256?.toLowerCase()
  return (
    domain.verification.status === 'matched' &&
    expected !== undefined &&
    expected !== '' &&
    domain.verification.next_signing_certificate_thumbprint?.toLowerCase() === expected
  )
}

// confirmationKey は、管理者の確認が前提にした準備済みの鍵、Entra の照合、信頼先の集合を表す。
// 値が変われば、以前の確認は今の対象を確かめたことにならない。
export function confirmationKey(snapshot: RolloverSnapshot): string {
  const entra = snapshot.entraDomains
    .map((domain) => [
      domain.domain,
      domain.verification.status,
      domain.verification.next_signing_certificate_thumbprint ?? '',
    ])
    .sort(([a], [b]) => a.localeCompare(b))
  const trusts = snapshot.trusts.map((trust) => trust.id).sort()
  return JSON.stringify([preparedKey(snapshot.keys)?.kid ?? '', entra, trusts])
}

// activationBlockers は切り替えを妨げる理由を返す。空なら切り替えを要求できる。
export function activationBlockers(
  snapshot: RolloverSnapshot,
  confirmations: RolloverConfirmations,
): ActivationBlocker[] {
  const prepared = preparedKey(snapshot.keys)
  if (!prepared) return [{ kind: 'no_prepared_key' }]
  const blockers: ActivationBlocker[] = []
  for (const domain of snapshot.entraDomains) {
    if (!entraConfirmsPrepared(domain, prepared)) {
      blockers.push({ kind: 'entra_not_matched', domain: domain.domain })
    }
  }
  for (const trust of snapshot.trusts) {
    if (!confirmations.trusts.has(trust.id)) blockers.push({ kind: 'trust_unconfirmed', trust })
  }
  if (!confirmations.overall) blockers.push({ kind: 'overall_unconfirmed' })
  return blockers
}
