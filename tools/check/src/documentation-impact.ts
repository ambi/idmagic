import type { SpecificationDiff } from './spec-diff.ts'

export type DocumentationImpact =
  | 'none'
  | 'release_note'
  | 'upgrade_note'
  | 'deprecation_notice'
  | 'removal_notice'

export type FeatureMaturity = 'experimental' | 'preview' | 'supported' | 'deprecated'

export type MaturityChange = {
  feature: string
  from?: FeatureMaturity
  to?: FeatureMaturity
}

export type DocumentationImpactEnvironment = {
  read: (path: string) => string | undefined
  specificationDiff: SpecificationDiff
  maturityChanges: MaturityChange[]
  breakingApiChanges?: string[]
  /**
   * Ids of the work items this working tree is changing, by filename stem.
   *
   * The specification diff is one workspace-wide difference against the
   * baseline revision, so it says nothing about which record produced it.
   * Without this set, every record under the documentation contract inherits
   * today's diff, and a completed `none` record from months ago starts failing
   * the moment somebody else adds a scenario. Undefined keeps the previous
   * behaviour for callers that cannot resolve it (minimal fixtures, tests that
   * exercise the inference directly).
   */
  changedRecords?: ReadonlySet<string>
  /**
   * Whether some record's `affected_spec` names an element the diff added.
   *
   * The specification half of the diff has an owner the records state
   * themselves: the item that declares the added scenario, standard, or
   * declaration in `affected_spec`. When one does, its open siblings are not
   * also responsible for it. When none does, every record that owns the
   * workspace diff still inherits it, so an addition nobody declared is caught
   * rather than attributed to nobody.
   */
  specificationAdditionsClaimed?: boolean
}

type WorkItemRecord = {
  id?: unknown
  status?: unknown
  evidence_policy?: unknown
  change_kind?: unknown
  affected_spec?: unknown
  primary_use_cases?: unknown
  documentation_impact?: unknown
  maturity_evidence?: unknown
}

type DocumentationReference = {
  kind?: unknown
  path?: unknown
}

type DocumentationImpactDeclaration = {
  level?: unknown
  reason?: unknown
  references?: unknown
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

const IMPACT_ORDER: Record<DocumentationImpact, number> = {
  none: 0,
  release_note: 1,
  upgrade_note: 2,
  deprecation_notice: 3,
  removal_notice: 4,
}

const MATURITY_ORDER: Record<FeatureMaturity, number> = {
  experimental: 0,
  preview: 1,
  supported: 2,
  deprecated: 3,
}

const IMPACTS = new Set<DocumentationImpact>(Object.keys(IMPACT_ORDER) as DocumentationImpact[])
const MATURITIES = new Set<FeatureMaturity>(Object.keys(MATURITY_ORDER) as FeatureMaturity[])

function object(value: unknown): Record<string, unknown> | undefined {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

function nonEmpty(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0
}

function activeUnderDocumentationContract(record: WorkItemRecord): boolean {
  if (record.evidence_policy !== 'risk-based-v3') return false
  if (record.status === 'in_progress') return true
  if (record.status !== 'completed' || typeof record.id !== 'string') return false
  const sequence = Number(record.id.match(/^wi-(\d+)-/)?.[1] ?? 0)
  return sequence >= 452
}

function stronger(
  current: DocumentationImpact,
  candidate: DocumentationImpact,
): DocumentationImpact {
  return IMPACT_ORDER[candidate] > IMPACT_ORDER[current] ? candidate : current
}

function isPromotion(change: MaturityChange): boolean {
  if (!change.from || !change.to) return false
  if (change.to !== 'preview' && change.to !== 'supported') return false
  return MATURITY_ORDER[change.to] > MATURITY_ORDER[change.from]
}

/**
 * Whether the workspace-wide specification and API diff belongs to this record.
 *
 * An `in_progress` record is always making the change being checked. A record
 * that is already `completed` only owns the diff while its own file is part of
 * the same working tree — which covers the completion step itself, where the
 * status flips to `completed` and the file moves into `work-items/done/`
 * before the final gate runs. A completed record identical to the baseline
 * owns nothing here.
 */
function ownsWorkspaceDiff(
  record: WorkItemRecord,
  environment: DocumentationImpactEnvironment,
): boolean {
  if (record.status === 'in_progress') return true
  const changed = environment.changedRecords
  if (changed === undefined) return true
  return typeof record.id === 'string' && changed.has(record.id)
}

/** The requirement ids and TypeSpec symbols a record's `affected_spec` names. */
function declaredSpecificationElements(record: WorkItemRecord): Set<string> {
  const declared = new Set<string>()
  if (!Array.isArray(record.affected_spec)) return declared
  for (const entry of record.affected_spec) {
    const reference = object(entry)
    if (!reference) continue
    for (const field of ['requirement', 'symbol']) {
      if (nonEmpty(reference[field])) declared.add(reference[field].trim())
    }
  }
  return declared
}

/**
 * Whether the elements this diff adds belong to this record.
 *
 * "In progress" and "being written right now" stop coinciding as soon as a
 * long-running parent waits on children: wi-495 stays `in_progress` until nine
 * child items finish, so every child's added scenario also landed on the
 * parent, which had declared `documentation_impact: none` for work that
 * genuinely has none (wi-509). Ownership of an addition is not a fact about
 * which records happen to be open — the record that declares the added element
 * in `affected_spec` is the one that made it, and that is written down.
 *
 * When no record declares any of the additions, they stay with every record
 * that owns the workspace diff, so an addition nobody claimed is reported
 * rather than attributed to nobody.
 */
function ownsSpecificationAdditions(
  record: WorkItemRecord,
  environment: DocumentationImpactEnvironment,
): boolean {
  if (environment.specificationAdditionsClaimed !== true) return true
  const declared = declaredSpecificationElements(record)
  const diff = environment.specificationDiff
  const added = [...diff.addedScenarios, ...diff.addedStandards, ...diff.addedDeclarations]
  return added.some((element) => declared.has(element))
}

export function minimumDocumentationImpact(
  record: WorkItemRecord,
  environment: DocumentationImpactEnvironment,
): DocumentationImpact {
  let impact: DocumentationImpact = record.change_kind === 'feature' ? 'release_note' : 'none'
  if (!ownsWorkspaceDiff(record, environment)) return impact
  const diff = environment.specificationDiff

  if (
    ownsSpecificationAdditions(record, environment) &&
    (diff.addedScenarios.length > 0 ||
      diff.addedStandards.length > 0 ||
      diff.addedDeclarations.length > 0)
  ) {
    impact = stronger(impact, 'release_note')
  }
  if ((environment.breakingApiChanges?.length ?? 0) > 0) {
    impact = stronger(impact, 'upgrade_note')
  }
  if (diff.addedDeprecations.length > 0) {
    impact = stronger(impact, 'deprecation_notice')
  }
  if (
    diff.removedScenarios.length > 0 ||
    diff.removedStandards.length > 0 ||
    diff.removedDeclarations.length > 0
  ) {
    impact = stronger(impact, 'removal_notice')
  }

  for (const change of environment.maturityChanges) {
    if (change.to === undefined) impact = stronger(impact, 'removal_notice')
    else if (change.to === 'deprecated') impact = stronger(impact, 'deprecation_notice')
    else if (isPromotion(change) || change.from === undefined) {
      impact = stronger(impact, 'release_note')
    } else if (change.from && MATURITY_ORDER[change.to] < MATURITY_ORDER[change.from]) {
      impact = stronger(impact, 'upgrade_note')
    }
  }
  return impact
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

function referencesOf(declaration: DocumentationImpactDeclaration): DocumentationReference[] {
  return Array.isArray(declaration.references)
    ? declaration.references.map(object).filter((value): value is DocumentationReference => !!value)
    : []
}

function stableSpecificationTokens(record: WorkItemRecord): string[] {
  if (!Array.isArray(record.affected_spec)) return []
  const tokens: string[] = []
  for (const raw of record.affected_spec) {
    const reference = object(raw)
    if (!reference) continue
    if (nonEmpty(reference.requirement)) tokens.push(reference.requirement)
    if (nonEmpty(reference.symbol)) tokens.push(reference.symbol)
  }
  return tokens
}

function requiredReferenceKinds(
  level: DocumentationImpact,
): Array<'release_note' | 'upgrade_note'> {
  if (level === 'none') return []
  if (level === 'release_note') return ['release_note']
  return ['release_note', 'upgrade_note']
}

function verifyMaturityPromotions(
  record: WorkItemRecord,
  environment: DocumentationImpactEnvironment,
): string[] {
  if (record.status !== 'completed') return []
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
      findings.push(`maturity_evidence ${promotion.feature} has no documentation path`)
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

export function verifyDocumentationImpact(
  record: WorkItemRecord,
  environment: DocumentationImpactEnvironment,
): string[] {
  if (!activeUnderDocumentationContract(record)) return []
  const findings: string[] = []
  const declaration = object(record.documentation_impact) as
    | DocumentationImpactDeclaration
    | undefined
  if (!declaration) return ['documentation_impact is required after implementation starts']
  const level = declaration.level
  if (typeof level !== 'string' || !IMPACTS.has(level as DocumentationImpact)) {
    return ['documentation_impact.level is not a recognized impact']
  }
  const impact = level as DocumentationImpact
  const references = referencesOf(declaration)
  if (!nonEmpty(declaration.reason) || declaration.reason.trim().length < 10) {
    findings.push('documentation_impact.reason must be concrete')
  }
  if (impact === 'none' && references.length > 0) {
    findings.push('documentation_impact none must not declare release-document references')
  }
  for (const kind of requiredReferenceKinds(impact)) {
    if (!references.some((reference) => reference.kind === kind)) {
      const article = kind === 'upgrade_note' ? 'an' : 'a'
      findings.push(`documentation_impact ${impact} requires ${article} ${kind} reference`)
    }
  }

  const minimum = minimumDocumentationImpact(record, environment)
  if (IMPACT_ORDER[impact] < IMPACT_ORDER[minimum]) {
    findings.push(`documentation_impact ${impact} is weaker than inferred ${minimum}`)
  }

  for (const reference of references) {
    if (reference.kind !== 'release_note' && reference.kind !== 'upgrade_note') continue
    if (!nonEmpty(reference.path)) continue
    const prefix =
      reference.kind === 'release_note' ? 'docs/releases/changes/wi-' : 'docs/releases/upgrades/wi-'
    if (!reference.path.startsWith(prefix) || !reference.path.endsWith('.md')) {
      findings.push(
        `documentation reference has the wrong path for ${reference.kind}: ${reference.path}`,
      )
      continue
    }
    if (record.status !== 'completed') continue
    const source = environment.read(reference.path)
    if (source === undefined) {
      findings.push(`documentation reference does not exist: ${reference.path}`)
      continue
    }
    if (nonEmpty(record.id) && !source.includes(record.id)) {
      findings.push(`documentation reference ${reference.path} does not name ${record.id}`)
    }
    const tokens = stableSpecificationTokens(record)
    if (tokens.length === 0 || !tokens.some((token) => source.includes(token))) {
      findings.push(
        `documentation reference ${reference.path} does not name an affected_spec requirement or symbol`,
      )
    }
  }
  findings.push(...verifyMaturityPromotions(record, environment))
  return findings
}
