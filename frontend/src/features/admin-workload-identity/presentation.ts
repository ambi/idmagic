import type { Locale } from '../../lib/i18n'
import type { AdminAgent, AdminAuditEvent, WorkloadTrustBundle } from '../../types'
import type { adminWorkloadIdentityDictionary } from './AdminWorkloadIdentityPage.i18n'

export type Dictionary = (typeof adminWorkloadIdentityDictionary)['en']

// presentation は画面から表示判断を切り出した純関数の集まり。DOM も fetch も持たないので、
// 単体テストが描画を経由せずに済む。

// REASON_KEYS は仕様 (WorkloadAttestationRejected.reason) が列挙する理由コードを辞書の鍵へ写す。
// 仕様が理由を増やしたときに未知の値を握り潰さないよう、引けなかった値は生のまま返す。
const REASON_KEYS: Record<string, keyof Dictionary> = {
  unregistered_issuer: 'reasonUnregisteredIssuer',
  trust_bundle_disabled: 'reasonTrustBundleDisabled',
  jwks_unavailable: 'reasonJwksUnavailable',
  invalid_signature: 'reasonInvalidSignature',
  expired: 'reasonExpired',
  audience_mismatch: 'reasonAudienceMismatch',
  ttl_exceeded: 'reasonTtlExceeded',
  no_binding_match: 'reasonNoBindingMatch',
  ambiguous_match: 'reasonAmbiguousMatch',
  agent_not_active: 'reasonAgentNotActive',
  agent_unbound: 'reasonAgentUnbound',
}

export function rejectionReasonLabel(reason: string, t: Dictionary): string {
  const key = REASON_KEYS[reason]
  return key ? t[key] : reason
}

export type AttestationRejection = {
  id: string
  occurredAt: string
  reason: string
  trustBundleId?: string
}

// attestationRejections は監査イベントの payload を表示可能な行へ写す。専用の読み取り API は
// 作らず、WorkloadAttestationRejected を type 完全一致で引いた監査検索の結果をそのまま渡す。
export function attestationRejections(events: AdminAuditEvent[]): AttestationRejection[] {
  const rejections: AttestationRejection[] = []
  for (const event of events) {
    const reason = event.payload.reason
    // reason を持たない行は理由を示せず、拒否の一覧として意味を成さないので落とす。
    if (typeof reason !== 'string' || reason === '') continue
    const trustBundleId = event.payload.trustBundleId
    rejections.push({
      id: event.id,
      occurredAt: event.occurred_at,
      reason,
      // trustBundleId は issuer が登録済みだった場合のみ載る。未登録の発行者による拒否は
      // どの信頼バンドルにも属さないまま一覧側にだけ現れる。
      ...(typeof trustBundleId === 'string' && trustBundleId !== '' ? { trustBundleId } : {}),
    })
  }
  return rejections
}

export function rejectionsForTrustBundle(
  rejections: AttestationRejection[],
  trustBundleID: string,
): AttestationRejection[] {
  return rejections.filter((rejection) => rejection.trustBundleId === trustBundleID)
}

// jwksSource は鍵素材の出どころを 1 行で示す。inline JWKS の本体は API が返さないため、
// 設定済みであるという事実だけを名前で示す。
export function jwksSource(bundle: WorkloadTrustBundle, t: Dictionary): string {
  if (bundle.jwks_uri) return bundle.jwks_uri
  if (bundle.has_inline_jwks) return t.jwksSourceInline
  return t.jwksSourceNone
}

// formatDateTime は lib/i18n の useFormatters と重なるが、あちらはフックで、未設定と
// 解釈できない値を扱わない。ここは純関数のまま置き、未設定を em ダッシュ、解釈できない値を
// 生のまま返す振る舞いをテストで固定する。
export function formatDateTime(value: string | undefined, locale: Locale): string {
  if (!value) return '—'
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return parsed.toLocaleString(locale === 'ja' ? 'ja-JP' : 'en-US')
}

export type Confirmation = { title: string; message: string }

// bundleConfirmation は破壊的操作の確認文を組み立てる。削除はバインディングをカスケードで
// 消すため、影響を受ける件数を必ず本文へ埋め込む。件数は詳細画面が読み込み済みのものを使う。
export function bundleConfirmation(
  action: 'delete' | 'disable',
  bundle: WorkloadTrustBundle,
  bindingCount: number,
  t: Dictionary,
): Confirmation {
  const withBindings = action === 'delete' ? t.deleteBundleMessage : t.disableBundleMessage
  const withoutBindings =
    action === 'delete' ? t.deleteBundleMessageNoBindings : t.disableBundleMessageNoBindings
  const template = bindingCount > 0 ? withBindings : withoutBindings
  return {
    title: action === 'delete' ? t.deleteBundleTitle : t.disableBundleTitle,
    message: template.replace('{name}', bundle.name).replace('{count}', String(bindingCount)),
  }
}

export function bindingConfirmation(
  action: 'delete' | 'disable',
  subjectPattern: string,
  t: Dictionary,
): Confirmation {
  return {
    title: action === 'delete' ? t.deleteBindingTitle : t.disableBindingTitle,
    message: (action === 'delete' ? t.deleteBindingMessage : t.disableBindingMessage).replace(
      '{pattern}',
      subjectPattern,
    ),
  }
}

// displayNameForID は id を読める名前へ解決する。エージェントも信頼バンドルも、画面が持つのは
// 上限付きの一覧なので、解決できない id はそのまま出す (空欄にすると、対象が失われたのか
// 単に手元の一覧に無いだけなのかを区別できない)。
export function displayNameForID(id: string, named: { id: string; name: string }[]): string {
  return named.find((candidate) => candidate.id === id)?.name ?? id
}

// multipleCredentialBindings は、対応先の Agent が複数の資格情報バインディングを持つかを返す。
// 交換は最初のバインディングを採るが、これは仕様に無い挙動なので、規則を決めないまま画面で
// 見えるようにする ([[wi-379]] の Out of Scope)。
export function multipleCredentialBindings(agentID: string, agents: AdminAgent[]): boolean {
  return (agents.find((agent) => agent.id === agentID)?.client_ids.length ?? 0) > 1
}

export function parseAudiences(input: string): string[] {
  return input.split(/[\s,]+/).filter(Boolean)
}

export type InlineJwksResult = { ok: true; value?: Record<string, unknown> } | { ok: false }

// parseInlineJwks は入力欄の文字列を JWKS オブジェクトへ変換する。空欄は「インライン JWKS を
// 使わない」であって誤りではない。JSON として読めても オブジェクトでないものは受け付けない。
export function parseInlineJwks(input: string): InlineJwksResult {
  const trimmed = input.trim()
  if (trimmed === '') return { ok: true, value: undefined }
  try {
    const parsed: unknown = JSON.parse(trimmed)
    if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) return { ok: false }
    return { ok: true, value: parsed as Record<string, unknown> }
  } catch {
    return { ok: false }
  }
}
