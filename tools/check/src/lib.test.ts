import { describe, expect, it } from 'bun:test'
import { lintRawText, locatePointer, validateAgainstSchema } from './lib.ts'

describe('lintRawText', () => {
  it('returns no findings for a single-newline-terminated clean file', () => {
    expect(lintRawText('a: 1\n')).toEqual([])
  })

  it('flags missing trailing newline', () => {
    expect(lintRawText('a: 1')).toEqual([
      { line: 1, column: 1, message: 'file does not end with a newline' },
    ])
  })

  it('flags double trailing newline', () => {
    const f = lintRawText('a: 1\n\n')
    expect(f).toContainEqual({
      line: 2,
      column: 1,
      message: 'file ends with multiple trailing newlines',
    })
  })

  it('flags tab indentation with its column', () => {
    const f = lintRawText('foo:\n\tbar: 1\n')
    expect(f).toEqual([{ line: 2, column: 1, message: 'tab character in indentation' }])
  })

  it('flags trailing whitespace pointing past the last content char', () => {
    const f = lintRawText('a: 1   \n')
    expect(f).toEqual([{ line: 1, column: 5, message: 'trailing whitespace' }])
  })

  it('treats an empty file as clean', () => {
    expect(lintRawText('')).toEqual([])
  })
})

describe('locatePointer', () => {
  const yaml = [
    'id: example',
    'scope:',
    '  ui:',
    '    pages:',
    '      - one',
    '      - two',
    '      - three',
    'authors:',
    '  - alice',
  ].join('\n')

  it('returns 1 for the empty pointer', () => {
    expect(locatePointer(yaml, '')).toBe(1)
  })

  it('walks plain keys', () => {
    expect(locatePointer(yaml, '/id')).toBe(1)
    expect(locatePointer(yaml, '/scope')).toBe(2)
    expect(locatePointer(yaml, '/scope/ui')).toBe(3)
    expect(locatePointer(yaml, '/scope/ui/pages')).toBe(4)
  })

  it('walks array indices into list items', () => {
    expect(locatePointer(yaml, '/scope/ui/pages/0')).toBe(5)
    expect(locatePointer(yaml, '/scope/ui/pages/2')).toBe(7)
    expect(locatePointer(yaml, '/authors/0')).toBe(9)
  })

  it('falls back to the parent line when a segment cannot be resolved', () => {
    expect(locatePointer(yaml, '/scope/missing')).toBe(2)
    expect(locatePointer(yaml, '/scope/ui/pages/99')).toBe(4)
  })
})

describe('validateAgainstSchema — work-item', () => {
  const validWorkItem = {
    id: 'wi-1-demo',
    title: 'Demo',
    created_at: '2026-06-17',
    authors: ['tn'],
    status: 'pending',
    depends_on: [],
    motivation: 'because',
    scope: {},
    out_of_scope: [],
    verification: [],
    risk: 'low',
    risk_notes: 'none',
  }
  const validCompleted = {
    ...validWorkItem,
    status: 'completed',
  }

  it('accepts a minimal valid work item', () => {
    expect(validateAgainstSchema('work-item', validWorkItem, '')).toEqual([])
  })

  it('requires direct specification targets for feature work', () => {
    const feature = { ...validWorkItem, change_kind: 'feature' }
    expect(validateAgainstSchema('work-item', feature, '')).not.toEqual([])
    expect(
      validateAgainstSchema(
        'work-item',
        {
          ...feature,
          affected_spec: [
            { path: 'docs/modules/demo/scenarios.feature.md', requirement: 'REQ-DEMO-CHECK' },
          ],
        },
        '',
      ),
    ).toEqual([])
  })

  it('accepts a nested whole-system specification target', () => {
    expect(
      validateAgainstSchema(
        'work-item',
        {
          ...validWorkItem,
          change_kind: 'feature',
          affected_spec: [
            { path: 'docs/requirements/quality.md', requirement: 'SLO-DEMO-AVAILABILITY' },
          ],
        },
        '',
      ),
    ).toEqual([])
  })

  it('accepts a specification target without impact, and rejects the retired impact field', () => {
    const target = (extra: Record<string, string>) =>
      validateAgainstSchema(
        'work-item',
        {
          ...validWorkItem,
          change_kind: 'bugfix',
          affected_spec: [
            {
              path: 'docs/modules/demo/scenarios.feature.md',
              requirement: 'REQ-DEMO-001',
              ...extra,
            },
            { path: 'spec/modules/demo/main.tsp', symbol: 'Demo.Task', ...extra },
          ],
        },
        '',
      )
    expect(target({})).toEqual([])
    expect(target({ impact: 'conforms' })).not.toEqual([])
  })

  it('requires a concrete no-impact reason for maintenance without specification targets', () => {
    const maintenance = { ...validWorkItem, change_kind: 'maintenance' }
    expect(validateAgainstSchema('work-item', maintenance, '')).not.toEqual([])
    expect(
      validateAgainstSchema(
        'work-item',
        { ...maintenance, spec_impact: { kind: 'none', reason: 'Only generated text changed.' } },
        '',
      ),
    ).toEqual([])
  })

  it('rejects a missing required field', () => {
    const { risk_notes: _omitted, ...broken } = validWorkItem
    const f = validateAgainstSchema('work-item', broken, '')
    expect(f.length).toBeGreaterThan(0)
    expect(f[0]?.message).toContain('risk_notes')
  })

  it('rejects an out-of-enum status', () => {
    const f = validateAgainstSchema('work-item', { ...validWorkItem, status: 'planned' }, '')
    expect(f.some((x) => x.message.includes('status'))).toBe(true)
  })

  it('requires depends_on for active work items and accepts valid IDs', () => {
    const { depends_on: _omitted, ...withoutDependencies } = validWorkItem
    expect(validateAgainstSchema('work-item', withoutDependencies, '')).not.toEqual([])
    expect(
      validateAgainstSchema('work-item', { ...validWorkItem, depends_on: ['wi-1-foundation'] }, ''),
    ).toEqual([])
  })

  it('rejects malformed or duplicate dependency IDs', () => {
    for (const depends_on of [['WI_1'], ['wi-1-foundation', 'wi-1-foundation']]) {
      expect(validateAgainstSchema('work-item', { ...validWorkItem, depends_on }, '')).not.toEqual(
        [],
      )
    }
  })

  it('does not require depends_on once the item completes', () => {
    const { depends_on: _omitted, ...closed } = validCompleted
    expect(validateAgainstSchema('work-item', closed, '')).toEqual([])
  })

  it('rejects an out-of-enum risk', () => {
    const f = validateAgainstSchema('work-item', { ...validWorkItem, risk: 'catastrophic' }, '')
    expect(f.some((x) => x.message.includes('risk'))).toBe(true)
  })

  it('accepts either reversibility value and rejects anything else', () => {
    for (const reversibility of ['reversible', 'irreversible']) {
      expect(validateAgainstSchema('work-item', { ...validWorkItem, reversibility }, '')).toEqual(
        [],
      )
    }
    const f = validateAgainstSchema('work-item', { ...validWorkItem, reversibility: 'mostly' }, '')
    expect(f.some((x) => x.message.includes('reversibility'))).toBe(true)
  })

  it('rejects an id that violates the kebab-case pattern', () => {
    const f = validateAgainstSchema('work-item', { ...validWorkItem, id: 'WI_1' }, '')
    expect(f.some((x) => x.message.toLowerCase().includes('pattern'))).toBe(true)
  })

  it('accepts a string verification step', () => {
    const data = { ...validWorkItem, verification: ['manual smoke'] }
    expect(validateAgainstSchema('work-item', data, '')).toEqual([])
  })

  it('accepts an object verification step with cmd', () => {
    const data = { ...validWorkItem, verification: [{ cmd: 'bun test' }] }
    expect(validateAgainstSchema('work-item', data, '')).toEqual([])
  })

  it('accepts plan and tasks sections for a medium work item', () => {
    const data = {
      ...validWorkItem,
      plan: 'Use the existing loader and renderer paths.',
      tasks: '- [ ] T001 [Tooling] Update parser.\n- [x] T002 [Verify] Run tests.',
    }
    expect(validateAgainstSchema('work-item', data, '')).toEqual([])
  })

  it('rejects tasks without a task checkbox id', () => {
    const data = { ...validWorkItem, tasks: '- parser work' }
    const f = validateAgainstSchema('work-item', data, '')
    expect(f.length).toBeGreaterThan(0)
  })

  it('rejects a verification object missing cmd', () => {
    const data = { ...validWorkItem, verification: [{ reason: 'why' }] }
    const f = validateAgainstSchema('work-item', data, '')
    expect(f.length).toBeGreaterThan(0)
  })

  it('accepts a completed work item without a completion section', () => {
    expect(validateAgainstSchema('work-item', validCompleted, '')).toEqual([])
  })

  it('accepts both compatibility and migration results in maturity evidence', () => {
    const started = {
      ...validWorkItem,
      status: 'in_progress',
      maturity_evidence: [
        {
          feature: 'demo-v1',
          from: 'preview',
          to: 'supported',
          security: 'The security review found no unresolved gap.',
          compatibility: 'Existing configuration remains valid.',
          migration: 'No migration is required.',
          documentation: 'docs/releases/changes/wi-999-demo.md',
        },
      ],
    }
    expect(validateAgainstSchema('work-item', started, '')).toEqual([])
  })

  it('成熟度の告知パスは状態依存の検査へ委ね、セキュリティと移行情報は必須のままにする', () => {
    const evidence = {
      feature: 'demo-v1',
      from: 'preview',
      to: 'supported',
      security: 'セキュリティ確認を実施した。',
      compatibility: '既存の設定とデータを保持する。',
    }
    const started = {
      ...validWorkItem,
      status: 'in_progress',
      maturity_evidence: [evidence],
    }
    expect(validateAgainstSchema('work-item', started, '')).toEqual([])
    const { security: _security, ...withoutSecurity } = evidence
    expect(
      validateAgainstSchema('work-item', { ...started, maturity_evidence: [withoutSecurity] }, ''),
    ).not.toEqual([])
    const { compatibility: _compatibility, ...withoutCompatibility } = evidence
    expect(
      validateAgainstSchema(
        'work-item',
        { ...started, maturity_evidence: [withoutCompatibility] },
        '',
      ),
    ).not.toEqual([])
  })

  it('starts medium-risk work without an approval record', () => {
    const started = {
      ...validWorkItem,
      status: 'in_progress',
      risk: 'medium',
    }
    expect(validateAgainstSchema('work-item', started, '')).toEqual([])
  })

  it('rejects an affected_spec path below the retired docs/domain tree', () => {
    const reference = (path: string) =>
      validateAgainstSchema(
        'work-item',
        { ...validWorkItem, affected_spec: [{ path, requirement: 'REQ-DEMO-001' }] },
        '',
      )
    expect(reference('docs/modules/demo/features/task.md')).toEqual([])
    expect(reference('docs/domain/demo/work/task/README.md')).not.toEqual([])
  })

  it('asks an irreversible item for no evidence beyond its risk row', () => {
    const completed = { ...validCompleted, risk: 'low', reversibility: 'irreversible' }
    expect(validateAgainstSchema('work-item', completed, '')).toEqual([])
  })

  it('leaves a reversible low-risk item free of the medium-risk evidence', () => {
    const completed = { ...validCompleted, risk: 'low', reversibility: 'reversible' }
    expect(validateAgainstSchema('work-item', completed, '')).toEqual([])
  })
})

describe('validateAgainstSchema — line resolution', () => {
  it('maps a top-level required-field failure to the document head', () => {
    const text = 'id: wi-1\ntitle: x\n'
    const f = validateAgainstSchema('work-item', { id: 'wi-1', title: 'x' }, text)
    expect(f.length).toBeGreaterThan(0)
    expect(f[0]?.line).toBe(1)
  })

  it('maps an enum violation to the field line', () => {
    const text = ['id: wi-1', 'title: x', 'status: planned', ''].join('\n')
    const data = {
      id: 'wi-1',
      title: 'x',
      created_at: '2026-06-17',
      authors: ['tn'],
      status: 'planned',
      motivation: 'x',
      scope: {},
      out_of_scope: [],
      affected_guarantees: [],
      verification: [],
      risk: 'low',
      risk_notes: 'x',
    }
    const f = validateAgainstSchema('work-item', data, text)
    const statusErr = f.find((x) => x.message.includes('status'))
    expect(statusErr?.line).toBe(3)
  })
})
