import { describe, expect, it } from 'bun:test'
import {
  type DocumentationImpactEnvironment,
  type MaturityChange,
  claimsSpecificationAddition,
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
  it('attributes the workspace specification diff only to the records that change', () => {
    // The specification diff is one workspace-wide difference and carries no
    // record of its own. Without this scoping, adding a scenario made every
    // completed `none` record fail at once (wi-396).
    const addedScenario = {
      ...noSpecificationChange,
      addedScenarios: ['REQ-SYSTEM-018'],
    }
    const completed = {
      ...record,
      id: 'wi-452-feature-maturity-documentation-gates',
      status: 'completed',
    }

    expect(
      verifyDocumentationImpact(
        completed,
        environment({ specificationDiff: addedScenario, changedRecords: new Set() }),
      ),
    ).toEqual([])

    // The record the working tree is changing still owns the diff, whether it
    // is still in progress or has just flipped to completed for the final gate.
    expect(
      verifyDocumentationImpact(
        record,
        environment({ specificationDiff: addedScenario, changedRecords: new Set() }),
      ),
    ).toContain('documentation_impact none is weaker than inferred release_note')
    expect(
      verifyDocumentationImpact(
        completed,
        environment({
          specificationDiff: addedScenario,
          changedRecords: new Set([completed.id]),
        }),
      ),
    ).toContain('documentation_impact none is weaker than inferred release_note')
  })

  it('gives an added element to the record that declares it, not to its open siblings', () => {
    // "In progress" and "being written right now" stop coinciding once a parent
    // waits on children: wi-495 stays in progress through nine child items, and
    // every child's added scenario used to land on the parent too (wi-509).
    const addedScenario = { ...noSpecificationChange, addedScenarios: ['REQ-SYSTEM-018'] }
    const author = {
      ...record,
      id: 'wi-508-author',
      status: 'in_progress',
      affected_spec: [
        { path: 'docs/contexts/system/scenarios.feature.md', requirement: 'REQ-SYSTEM-018' },
      ],
    }
    const sibling = { ...record, id: 'wi-495-parent', status: 'in_progress' }
    const claimed = { specificationDiff: addedScenario, specificationAdditionsClaimed: true }

    expect(verifyDocumentationImpact(author, environment(claimed))).toContain(
      'documentation_impact none is weaker than inferred release_note',
    )
    expect(verifyDocumentationImpact(sibling, environment(claimed))).toEqual([])

    // An addition no record declares is reported rather than attributed to
    // nobody: every record that owns the workspace diff still inherits it.
    const unclaimed = { specificationDiff: addedScenario, specificationAdditionsClaimed: false }
    expect(verifyDocumentationImpact(sibling, environment(unclaimed))).toContain(
      'documentation_impact none is weaker than inferred release_note',
    )
  })

  it('gives an added TypeSpec declaration to the record that declares it', () => {
    // The diff spells a declaration `<path>:<name>`, while a record names it by
    // TypeSpec symbol under the file's `path` — the spelling
    // `work-item-references.ts` resolves. Comparing the two spellings directly
    // matched nothing, so no record could ever claim an added declaration and
    // every open record inherited it instead (wi-523 inflated wi-495).
    const addedDeclaration = {
      ...noSpecificationChange,
      addedDeclarations: ['spec/contexts/demo/models.tsp:PresentationTokenType'],
    }
    const author = {
      ...record,
      id: 'wi-523-author',
      status: 'in_progress',
      affected_spec: [
        {
          path: 'spec/contexts/demo/models.tsp',
          symbol: 'Demo.Contract.PresentationTokenType',
        },
      ],
    }
    const sibling = { ...record, id: 'wi-495-parent', status: 'in_progress' }
    const claimed = { specificationDiff: addedDeclaration, specificationAdditionsClaimed: true }

    expect(verifyDocumentationImpact(author, environment(claimed))).toContain(
      'documentation_impact none is weaker than inferred release_note',
    )
    expect(verifyDocumentationImpact(sibling, environment(claimed))).toEqual([])
  })

  it('matches an added declaration by file and declaration name together', () => {
    const declaration = 'spec/contexts/demo/models.tsp:PresentationTokenType'
    const reference = (overrides: Record<string, string>) => ({
      affected_spec: [
        {
          path: 'spec/contexts/demo/models.tsp',
          symbol: 'Demo.Contract.PresentationTokenType',
          ...overrides,
        },
      ],
    })

    expect(claimsSpecificationAddition(reference({}), [declaration])).toBe(true)

    // The path pins the file. Without it, a same-named declaration added to
    // another context would be claimed by an unrelated record.
    expect(
      claimsSpecificationAddition(reference({ path: 'spec/contexts/other/models.tsp' }), [
        declaration,
      ]),
    ).toBe(false)
    expect(
      claimsSpecificationAddition(reference({ symbol: 'Demo.Contract.SomethingElse' }), [
        declaration,
      ]),
    ).toBe(false)

    // A scenario or standard is added under its own id and keeps matching the
    // `requirement` verbatim.
    expect(
      claimsSpecificationAddition(
        {
          affected_spec: [
            { path: 'docs/contexts/demo/scenarios.feature.md', requirement: 'REQ-DEMO-001' },
          ],
        },
        ['REQ-DEMO-001'],
      ),
    ).toBe(true)
    expect(claimsSpecificationAddition({}, [declaration])).toBe(false)
  })

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
      affected_spec: [
        { path: 'docs/contexts/demo/scenarios.feature.md', requirement: 'REQ-DEMO-001' },
      ],
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
