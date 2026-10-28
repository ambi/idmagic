import type { ReleasePhase } from './release-state.ts'

export type FeatureMaturity = 'experimental' | 'preview' | 'supported' | 'deprecated'

export type MaturityChange = {
  feature: string
  from?: FeatureMaturity
  to?: FeatureMaturity
}

export type MaturityEnvironment = {
  releasePhase: ReleasePhase
  read: (path: string) => string | undefined
  maturityChanges: MaturityChange[]
  /**
   * Ids of the work items this working tree is changing, by filename stem. The
   * maturity diff is workspace-wide, so a completed record identical to the
   * baseline does not own it. Undefined treats every completed record as owner.
   */
  changedRecords?: ReadonlySet<string>
}

type WorkItemRecord = {
  id?: unknown
  status?: unknown
  primary_use_cases?: unknown
  maturity_evidence?: unknown
}

type MaturityEvidence = {
  feature?: unknown
  from?: unknown
  to?: unknown
  security?: unknown
  compatibility?: unknown
  migration?: unknown
  documentation?: unknown
}

const MATURITY_ORDER: Record<FeatureMaturity, number> = {
  experimental: 0,
  preview: 1,
  supported: 2,
  deprecated: 3,
}

const MATURITIES = new Set<FeatureMaturity>(Object.keys(MATURITY_ORDER) as FeatureMaturity[])

function object(value: unknown): Record<string, unknown> | undefined {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

function nonEmpty(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0
}

function isPromotion(change: MaturityChange): boolean {
  if (!change.from || !change.to) return false
  if (change.to !== 'preview' && change.to !== 'supported') return false
  return MATURITY_ORDER[change.to] > MATURITY_ORDER[change.from]
}

function featureMaturities(source: string): Map<string, FeatureMaturity> {
  const features = new Map<string, FeatureMaturity>()
  const ids = [...source.matchAll(/\bID:\s*"([a-z0-9][a-z0-9-]*)"/g)]
  for (const [index, match] of ids.entries()) {
    const feature = match[1]
    if (!feature || match.index === undefined) continue
    const end = ids[index + 1]?.index ?? source.length
    const maturity = source
      .slice(match.index, end)
      .match(/\bMaturity:\s*Feature(Experimental|Preview|Supported|Deprecated)\b/)?.[1]
      ?.toLowerCase()
    if (maturity && MATURITIES.has(maturity as FeatureMaturity)) {
      features.set(feature, maturity as FeatureMaturity)
    }
  }
  return features
}

/** 基準と作業ツリーの機能の登録を比べ、成熟度の変わった機能を返す。 */
export function diffFeatureMaturities(base: string, head: string): MaturityChange[] {
  const before = featureMaturities(base)
  const after = featureMaturities(head)
  const changes: MaturityChange[] = []
  for (const feature of new Set([...before.keys(), ...after.keys()])) {
    const from = before.get(feature)
    const to = after.get(feature)
    if (from !== to) changes.push({ feature, from, to })
  }
  return changes.sort((left, right) => left.feature.localeCompare(right.feature))
}

/**
 * 成熟度を preview または supported へ上げる記録に、主要ユースケース、セキュリティ確認、
 * 互換性または移行の結果を求める。公開後は、昇格を示す告知文書も求める。
 */
export function verifyMaturityPromotions(
  record: WorkItemRecord,
  environment: MaturityEnvironment,
): string[] {
  if (record.status !== 'completed') return []
  const changed = environment.changedRecords
  if (changed !== undefined && !(typeof record.id === 'string' && changed.has(record.id))) return []
  const findings: string[] = []
  const evidence = Array.isArray(record.maturity_evidence)
    ? record.maturity_evidence.map(object).filter((value): value is MaturityEvidence => !!value)
    : []
  for (const promotion of environment.maturityChanges.filter(isPromotion)) {
    const label = `${promotion.feature}: ${promotion.from} -> ${promotion.to}`
    if (!Array.isArray(record.primary_use_cases) || record.primary_use_cases.length === 0) {
      findings.push(`primary_use_cases is required for maturity promotion ${label}`)
    }
    const result = evidence.find(
      (candidate) =>
        candidate.feature === promotion.feature &&
        candidate.from === promotion.from &&
        candidate.to === promotion.to,
    )
    if (!result) {
      findings.push(`maturity_evidence is required for promotion ${label}`)
      continue
    }
    if (!nonEmpty(result.security)) {
      findings.push(`maturity_evidence ${promotion.feature} has no security result`)
    }
    if (!nonEmpty(result.compatibility) && !nonEmpty(result.migration)) {
      findings.push(
        `maturity_evidence ${promotion.feature} has neither compatibility nor migration information`,
      )
    }
    if (!nonEmpty(result.documentation)) {
      if (environment.releasePhase === 'published') {
        findings.push(`maturity_evidence ${promotion.feature} has no documentation path`)
      }
      continue
    }
    const source = environment.read(result.documentation)
    if (source === undefined) {
      findings.push(`maturity documentation does not exist: ${result.documentation}`)
    } else if (!source.includes(promotion.feature) || !source.includes(promotion.to ?? '')) {
      findings.push(
        `maturity documentation ${result.documentation} does not show ${promotion.feature} as ${promotion.to}`,
      )
    }
  }
  return findings
}
