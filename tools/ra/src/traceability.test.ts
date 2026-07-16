import { describe, expect, it } from 'bun:test'
import { buildSclWorkspaceIndex } from '../../yaml-check/src/scl-element-reference.ts'
import {
  buildTraceabilityReport,
  type TraceabilityEvidence,
  type TraceabilityManifest,
} from './traceability.ts'

const target = { context: 'demo', kind: 'interface' as const, element: 'DoThing' }

function index() {
  const built = buildSclWorkspaceIndex(
    {
      system: 'demo',
      spec_version: '3.0',
      interfaces: { DoThing: { access: 'internal' } },
    },
    {},
  )
  if (!built.ok) throw new Error('fixture index failed')
  return built.index
}

function manifest(): TraceabilityManifest {
  return {
    version: 1,
    policies: [
      {
        id: 'interfaces',
        selector: { contexts: ['demo'], kinds: ['interface'] },
        requires_realization: true,
        minimum_checks: 1,
        evidence_kinds: ['test'],
      },
    ],
    realizations: [{ id: 'implementation', module: 'tools', targets: [target] }],
    checks: [
      {
        id: 'unit',
        recipe: 'test-tools',
        kind: 'test',
        implementation: { path: 'tools/ra/src/traceability.test.ts', symbol: 'accepts' },
        targets: [target],
      },
    ],
    baselines: [],
  }
}

function evidence(revision = 'abc'): TraceabilityEvidence {
  return {
    version: 1,
    source_revision: revision,
    executions: [
      {
        check: 'unit',
        kind: 'test',
        executed_at: '2026-07-17T00:00:00Z',
        target_revision: revision,
        result: 'passed',
      },
    ],
  }
}

describe('workspace traceability graph', () => {
  it('accepts a fully realized, verified, current graph', () => {
    const report = buildTraceabilityReport({
      manifest: manifest(),
      evidence: evidence(),
      index: index(),
      architectureModules: new Set(['tools']),
      sourceRevision: 'abc',
      strict: true,
    })
    expect(report.passed).toBe(true)
    expect(report.findings).toEqual([])
    expect(report.nodes).toEqual({ specifications: 1, realizations: 1, checks: 1, evidence: 1 })
  })

  it('classifies missing realization, verification, and evidence independently', () => {
    const missing = manifest()
    missing.realizations = []
    missing.checks = []
    const report = buildTraceabilityReport({
      manifest: missing,
      index: index(),
      architectureModules: new Set(['tools']),
      sourceRevision: 'abc',
      strict: true,
    })
    expect(report.findings.map((finding) => finding.code)).toEqual([
      'specified_without_realization',
      'specified_without_verification',
    ])

    const withoutEvidence = buildTraceabilityReport({
      manifest: manifest(),
      index: index(),
      architectureModules: new Set(['tools']),
      sourceRevision: 'abc',
      strict: true,
    })
    expect(withoutEvidence.findings.map((finding) => finding.code)).toEqual(['missing_evidence'])
  })

  it('distinguishes stale evidence and unknown targets', () => {
    const broken = manifest()
    broken.realizations.push({
      id: 'orphan-realization',
      module: 'tools',
      targets: [{ context: 'demo', kind: 'interface', element: 'Missing' }],
    })
    broken.checks.push({
      id: 'orphan-check',
      recipe: 'test-tools',
      kind: 'test',
      implementation: { path: 'tools/ra/src/traceability.test.ts', symbol: 'unknown targets' },
      targets: [{ context: 'demo', kind: 'interface', element: 'Missing' }],
    })
    const report = buildTraceabilityReport({
      manifest: broken,
      evidence: evidence('old'),
      index: index(),
      architectureModules: new Set(['tools']),
      sourceRevision: 'abc',
      strict: true,
    })
    expect(report.findings.map((finding) => finding.code)).toEqual([
      'realized_without_spec',
      'stale_evidence',
      'verification_without_target',
    ])
  })

  it('keeps report-only green and rejects expired baselines in strict mode', () => {
    const debt = manifest()
    debt.realizations = []
    debt.baselines = [
      {
        id: 'legacy',
        finding: 'specified_without_realization',
        owner: 'maintainers',
        reason: 'migration',
        expires_at: '2026-01-01',
      },
    ]
    const reportOnly = buildTraceabilityReport({
      manifest: debt,
      index: index(),
      architectureModules: new Set(['tools']),
      sourceRevision: 'abc',
      strict: false,
      now: new Date('2026-07-17T00:00:00Z'),
    })
    expect(reportOnly.passed).toBe(true)
    expect(reportOnly.findings.some((finding) => finding.code === 'expired_baseline')).toBe(true)

    const strict = buildTraceabilityReport({
      manifest: debt,
      index: index(),
      architectureModules: new Set(['tools']),
      sourceRevision: 'abc',
      strict: true,
      now: new Date('2026-07-17T00:00:00Z'),
    })
    expect(strict.passed).toBe(false)
  })

  it('rejects unknown recipes and missing executable check symbols', () => {
    const report = buildTraceabilityReport({
      manifest: manifest(),
      evidence: evidence(),
      index: index(),
      architectureModules: new Set(['tools']),
      sourceRevision: 'abc',
      strict: true,
      availableRecipes: new Set(['verify']),
      implementationContains: () => false,
    })
    expect(report.findings.map((finding) => finding.code)).toEqual([
      'missing_check_implementation',
      'unknown_recipe',
    ])
    expect(report.passed).toBe(false)
  })
})
