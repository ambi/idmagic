import { describe, expect, it } from 'bun:test'
import {
  type DocumentationImpactEnvironment,
  type MaturityChange,
  diffFeatureMaturities,
  verifyDocumentationImpact,
} from './documentation-impact.ts'
import { diffSpecifications } from './spec-diff.ts'

const noSpecificationChange = diffSpecifications(new Map(), new Map())

const environment = (
  overrides: Partial<DocumentationImpactEnvironment> = {},
): DocumentationImpactEnvironment => ({
  read: () => undefined,
  specificationDiff: noSpecificationChange,
  maturityChanges: [],
  ...overrides,
})

const record = {
  id: 'wi-999-documentation-impact',
  status: 'in_progress',
  evidence_policy: 'risk-based-v3',
  change_kind: 'tooling',
  documentation_impact: {
    level: 'none',
    reason: 'Repository tooling has no user-visible release difference.',
    references: [],
  },
}

describe('verifyDocumentationImpact', () => {
  it('derives feature maturity changes from registry definitions', () => {
    const base = `return FeatureRegistry{
      {ID: "demo-v1", Maturity: FeatureExperimental},
      {ID: "removed-v1", Maturity: FeatureSupported},
    }`
    const head = `return FeatureRegistry{
      {ID: "demo-v1", Maturity: FeaturePreview},
      {ID: "new-v1", Maturity: FeatureExperimental},
    }`
    expect(diffFeatureMaturities(base, head)).toEqual([
      { feature: 'demo-v1', from: 'experimental', to: 'preview' },
      { feature: 'new-v1', from: undefined, to: 'experimental' },
      { feature: 'removed-v1', from: 'supported', to: undefined },
    ])
  })

  it('infers minimum impact and rejects weaker declarations', () => {
    expect(
      verifyDocumentationImpact({ ...record, change_kind: 'feature' }, environment()),
    ).toContain('documentation_impact none is weaker than inferred release_note')

    expect(
      verifyDocumentationImpact(
        record,
        environment({
          specificationDiff: {
            ...noSpecificationChange,
            addedDeprecations: ['spec/contexts/demo/main.tsp:LegacyDemo'],
          },
        }),
      ),
    ).toContain('documentation_impact none is weaker than inferred deprecation_notice')

    expect(
      verifyDocumentationImpact(
        record,
        environment({
          specificationDiff: {
            ...noSpecificationChange,
            removedDeclarations: ['spec/contexts/demo/main.tsp:RemovedDemo'],
          },
        }),
      ),
    ).toContain('documentation_impact none is weaker than inferred removal_notice')

    expect(
      verifyDocumentationImpact(
        record,
        environment({ breakingApiChanges: ['GET /demo: response removed'] }),
      ),
    ).toContain('documentation_impact none is weaker than inferred upgrade_note')
  })

  it('requires a reason for none and the documents implied by every non-none impact', () => {
    expect(
      verifyDocumentationImpact(
        {
          ...record,
          documentation_impact: { level: 'none', reason: '', references: [] },
        },
        environment(),
      ),
    ).toContain('documentation_impact.reason must be concrete')

    for (const level of [
      'release_note',
      'upgrade_note',
      'deprecation_notice',
      'removal_notice',
    ] as const) {
      const findings = verifyDocumentationImpact(
        {
          ...record,
          documentation_impact: {
            level,
            reason: 'The release documentation obligation is explicit.',
            references: [],
          },
        },
        environment(),
      )
      expect(findings).toContain(`documentation_impact ${level} requires a release_note reference`)
      if (level !== 'release_note') {
        expect(findings).toContain(
          `documentation_impact ${level} requires an upgrade_note reference`,
        )
      }
    }
  })

  it('requires release and upgrade references selected by the impact', () => {
    const findings = verifyDocumentationImpact(
      {
        ...record,
        documentation_impact: {
          level: 'upgrade_note',
          reason: 'Existing users must change configuration during the upgrade.',
          references: [
            {
              kind: 'release_note',
              path: 'docs/releases/changes/wi-999-documentation-impact.md',
            },
          ],
        },
      },
      environment(),
    )
    expect(findings).toContain(
      'documentation_impact upgrade_note requires an upgrade_note reference',
    )
  })

  it('checks completed release documents for the work item and a stable specification reference', () => {
    const path = 'docs/releases/changes/wi-999-documentation-impact.md'
    const completed = {
      ...record,
      status: 'completed',
      affected_spec: [{ path: 'docs/contexts/demo/scenarios.md', requirement: 'REQ-DEMO-001' }],
      documentation_impact: {
        level: 'release_note',
        reason: 'The completed feature is noteworthy to release readers.',
        references: [{ kind: 'release_note', path }],
      },
    }
    expect(verifyDocumentationImpact(completed, environment())).toContain(
      `documentation reference does not exist: ${path}`,
    )
    expect(
      verifyDocumentationImpact(
        completed,
        environment({ read: () => '# Change\n\nNo stable references.\n' }),
      ),
    ).toEqual([
      `documentation reference ${path} does not name wi-999-documentation-impact`,
      `documentation reference ${path} does not name an affected_spec requirement or symbol`,
    ])
  })

  it('requires promotion evidence for each detected maturity promotion', () => {
    const promotion: MaturityChange = {
      feature: 'demo-v1',
      from: 'preview',
      to: 'supported',
    }
    const findings = verifyDocumentationImpact(
      { ...record, status: 'completed' },
      environment({ maturityChanges: [promotion] }),
    )
    expect(findings).toContain('documentation_impact none is weaker than inferred release_note')
    expect(findings).toContain(
      'maturity_evidence is required for promotion demo-v1: preview -> supported',
    )
    expect(findings).toContain(
      'primary_use_cases is required for maturity promotion demo-v1: preview -> supported',
    )
  })

  it('checks every maturity-promotion evidence result', () => {
    const promotion: MaturityChange = {
      feature: 'demo-v1',
      from: 'experimental',
      to: 'preview',
    }
    const documentation = 'docs/releases/changes/wi-999-documentation-impact.md'
    const findings = verifyDocumentationImpact(
      {
        ...record,
        status: 'completed',
        primary_use_cases: [{ id: 'demo' }],
        documentation_impact: {
          level: 'release_note',
          reason: 'The promoted feature is noteworthy to release readers.',
          references: [{ kind: 'release_note', path: documentation }],
        },
        maturity_evidence: [
          {
            feature: 'demo-v1',
            from: 'experimental',
            to: 'preview',
            security: '',
            documentation,
          },
        ],
      },
      environment({
        maturityChanges: [promotion],
        read: () => '# Release\n\nwi-999-documentation-impact\n',
      }),
    )
    expect(findings).toContain('maturity_evidence demo-v1 has no security result')
    expect(findings).toContain(
      'maturity_evidence demo-v1 has neither compatibility nor migration information',
    )
    expect(findings).toContain(
      `maturity documentation ${documentation} does not show demo-v1 as preview`,
    )
  })
})
