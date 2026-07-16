import {
  canonicalSclElementReference,
  listSclElementReferences,
  resolveSclElementReference,
  type SclElementReference,
  type SclWorkspaceIndex,
} from '../../yaml-check/src/scl-element-reference.ts'

export type EvidenceKind =
  | 'test'
  | 'property_test'
  | 'contract_test'
  | 'static_analysis'
  | 'build'
  | 'lint'
  | 'typecheck'
  | 'manual_inspection'

export type TraceabilityManifest = {
  version: 1
  policies: Array<{
    id: string
    selector: { contexts?: string[]; kinds?: string[]; elements?: string[] }
    requires_realization: boolean
    minimum_checks: number
    evidence_kinds: EvidenceKind[]
  }>
  realizations: Array<{ id: string; module: string; targets: SclElementReference[] }>
  checks: Array<{
    id: string
    recipe: string
    kind: EvidenceKind
    implementation: { path: string; symbol: string }
    targets: SclElementReference[]
  }>
  baselines: Array<{
    id: string
    finding: string
    target?: string
    owner: string
    reason: string
    expires_at: string
  }>
}

export type TraceabilityEvidence = {
  version: 1
  source_revision: string
  executions: Array<{
    check: string
    kind: EvidenceKind
    executed_at: string
    target_revision: string
    result: 'passed' | 'failed'
    artifacts?: Array<{ path: string; sha256: string }>
  }>
}

export type TraceabilityFinding = {
  code: string
  target: string
  detail: string
  baseline?: string
}

export type TraceabilityReport = {
  source_revision: string
  strict: boolean
  passed: boolean
  nodes: { specifications: number; realizations: number; checks: number; evidence: number }
  findings: TraceabilityFinding[]
}

function matchesSelector(
  reference: SclElementReference,
  selector: TraceabilityManifest['policies'][number]['selector'],
): boolean {
  const element =
    reference.kind === 'standard_requirement' ? reference.requirement : reference.element
  return (
    (!selector.contexts || selector.contexts.includes(reference.context)) &&
    (!selector.kinds || selector.kinds.includes(reference.kind)) &&
    (!selector.elements || selector.elements.includes(element))
  )
}

export function buildTraceabilityReport(input: {
  manifest: TraceabilityManifest
  evidence?: TraceabilityEvidence
  index: SclWorkspaceIndex
  architectureModules: ReadonlySet<string>
  sourceRevision: string
  strict: boolean
  now?: Date
  availableRecipes?: ReadonlySet<string>
  implementationContains?: (path: string, symbol: string) => boolean
}): TraceabilityReport {
  const { manifest, evidence, index, architectureModules, sourceRevision, strict } = input
  const now = input.now ?? new Date()
  const findings: TraceabilityFinding[] = []
  const add = (code: string, target: string, detail: string) =>
    findings.push({ code, target, detail })
  const resolvedRealizations = new Map<string, number>()
  const resolvedChecks = new Map<string, string[]>()

  const duplicateIds = (kind: string, bindings: Array<{ id: string }>) => {
    const seen = new Set<string>()
    for (const binding of bindings) {
      if (seen.has(binding.id)) add('duplicate_binding', binding.id, `duplicate ${kind} id`)
      seen.add(binding.id)
    }
  }
  duplicateIds('policy', manifest.policies)
  duplicateIds('realization', manifest.realizations)
  duplicateIds('check', manifest.checks)

  for (const realization of manifest.realizations) {
    if (!architectureModules.has(realization.module)) {
      add('unknown_module', realization.id, `unknown Architecture module '${realization.module}'`)
    }
    const seen = new Set<string>()
    for (const target of realization.targets) {
      const resolved = resolveSclElementReference(index, target)
      if (!resolved.ok) {
        add('realized_without_spec', realization.id, resolved.error.message)
        continue
      }
      if (seen.has(resolved.canonical)) {
        add('duplicate_binding', realization.id, `duplicate target '${resolved.canonical}'`)
        continue
      }
      seen.add(resolved.canonical)
      resolvedRealizations.set(
        resolved.canonical,
        (resolvedRealizations.get(resolved.canonical) ?? 0) + 1,
      )
    }
  }

  for (const check of manifest.checks) {
    if (input.availableRecipes && !input.availableRecipes.has(check.recipe)) {
      add('unknown_recipe', check.id, `unknown just recipe '${check.recipe}'`)
    }
    if (
      input.implementationContains &&
      !input.implementationContains(check.implementation.path, check.implementation.symbol)
    ) {
      add(
        'missing_check_implementation',
        check.id,
        `implementation '${check.implementation.path}#${check.implementation.symbol}' was not found`,
      )
    }
    const targets: string[] = []
    const seen = new Set<string>()
    for (const target of check.targets) {
      const resolved = resolveSclElementReference(index, target)
      if (!resolved.ok) {
        add('verification_without_target', check.id, resolved.error.message)
        continue
      }
      if (seen.has(resolved.canonical)) {
        add('duplicate_binding', check.id, `duplicate target '${resolved.canonical}'`)
        continue
      }
      seen.add(resolved.canonical)
      targets.push(resolved.canonical)
    }
    resolvedChecks.set(check.id, targets)
  }

  for (const policy of manifest.policies) {
    const selected = listSclElementReferences(index).filter((reference) =>
      matchesSelector(reference, policy.selector),
    )
    for (const reference of selected) {
      const canonical = canonicalSclElementReference(reference)
      if (policy.requires_realization && !resolvedRealizations.has(canonical)) {
        add(
          'specified_without_realization',
          canonical,
          `policy '${policy.id}' requires realization`,
        )
      }
      const checks = manifest.checks.filter((check) =>
        resolvedChecks.get(check.id)?.includes(canonical),
      )
      if (checks.length < policy.minimum_checks) {
        add(
          'specified_without_verification',
          canonical,
          `policy '${policy.id}' requires ${policy.minimum_checks} check(s)`,
        )
      }
      for (const check of checks) {
        const executions = evidence?.executions.filter((item) => item.check === check.id) ?? []
        const accepted = executions.some(
          (item) =>
            item.result === 'passed' &&
            item.target_revision === sourceRevision &&
            policy.evidence_kinds.includes(item.kind) &&
            item.kind === check.kind,
        )
        if (accepted) continue
        const passedOldRevision = executions.some(
          (item) => item.result === 'passed' && item.target_revision !== sourceRevision,
        )
        add(
          passedOldRevision ? 'stale_evidence' : 'missing_evidence',
          check.id,
          `no accepted evidence for '${canonical}' at revision '${sourceRevision}'`,
        )
      }
    }
  }

  for (const baseline of manifest.baselines) {
    if (baseline.finding.startsWith('legacy_')) {
      add(baseline.finding, baseline.target ?? baseline.id, baseline.reason)
    }
    const expires = new Date(`${baseline.expires_at}T23:59:59.999Z`)
    if (Number.isNaN(expires.valueOf()) || expires < now) {
      add('expired_baseline', baseline.id, `baseline expired at '${baseline.expires_at}'`)
      continue
    }
    for (const finding of findings) {
      if (
        finding.baseline === undefined &&
        finding.code === baseline.finding &&
        (baseline.target === undefined || baseline.target === finding.target)
      ) {
        finding.baseline = baseline.id
      }
    }
  }

  findings.sort((a, b) =>
    `${a.code}\0${a.target}\0${a.detail}`.localeCompare(`${b.code}\0${b.target}\0${b.detail}`),
  )
  const activeFindings = findings.filter((finding) => finding.baseline === undefined)
  return {
    source_revision: sourceRevision,
    strict,
    passed: !strict || activeFindings.length === 0,
    nodes: {
      specifications: listSclElementReferences(index).length,
      realizations: manifest.realizations.length,
      checks: manifest.checks.length,
      evidence: evidence?.executions.length ?? 0,
    },
    findings,
  }
}
