import { describe, expect, it } from 'bun:test'
import { SCHEMAS, lintRawText, locatePointer, parseArgs, validateAgainstSchema } from './lib.ts'

describe('parseArgs', () => {
  it('captures positional files', () => {
    const r = parseArgs(['a.yaml', 'b.yaml'])
    expect(r).toEqual({
      kind: 'ok',
      opts: { schema: null, files: ['a.yaml', 'b.yaml'], listSchemas: false, help: false },
    })
  })

  it('accepts --schema=name', () => {
    const r = parseArgs(['--schema=work-item', 'a.yaml'])
    expect(r.kind).toBe('ok')
    if (r.kind === 'ok') expect(r.opts.schema).toBe('work-item')
  })

  it('accepts --schema name', () => {
    const r = parseArgs(['--schema', 'scl', 'a.yaml'])
    expect(r.kind).toBe('ok')
    if (r.kind === 'ok') {
      expect(r.opts.schema).toBe('scl')
      expect(r.opts.files).toEqual(['a.yaml'])
    }
  })

  it('errors when --schema is the final arg', () => {
    const r = parseArgs(['--schema'])
    expect(r).toEqual({ kind: 'error', code: 2, message: '--schema requires a value' })
  })

  it('errors on unknown flag', () => {
    const r = parseArgs(['--what'])
    expect(r.kind).toBe('error')
    if (r.kind === 'error') expect(r.code).toBe(2)
  })

  it('captures --list-schemas', () => {
    const r = parseArgs(['--list-schemas'])
    expect(r.kind).toBe('ok')
    if (r.kind === 'ok') expect(r.opts.listSchemas).toBe(true)
  })

  it('captures --help / -h', () => {
    expect(parseArgs(['--help'])).toMatchObject({ kind: 'ok', opts: { help: true } })
    expect(parseArgs(['-h'])).toMatchObject({ kind: 'ok', opts: { help: true } })
  })
})

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

describe('SCHEMAS', () => {
  it('exposes exactly the documented schemas', () => {
    expect(Object.keys(SCHEMAS).sort()).toEqual(['work-item'])
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
  const validCompletion = {
    completed_at: '2026-06-17',
    summary: 'done',
    verification: [{ cmd: 'go test ./...', result: 'ok' }],
  }
  const validInitialContext = {
    specification: ['docs/scenarios.md#REQ-DEMO-001'],
  }
  const validApproval = {
    by: 'tn',
    at: '2026-06-17',
    scope: 'Implement the approved work item.',
    baseline: '3cb041f1d61007a3213ead7c1bba989d1d19824a',
  }
  const validRedEvidence = {
    test: 'TestDemo rejects an invalid transition',
    requirement: 'REQ-DEMO-001',
    observed_failure: 'expected rejection, received success',
    detection_reason: 'the assertion distinguishes the invalid transition from success',
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
            { path: 'docs/contexts/demo/scenarios.md', requirement: 'REQ-DEMO-CHECK' },
          ],
        },
        '',
      ),
    ).toEqual([])
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

  it('keeps completed records without depends_on compatible', () => {
    const { depends_on: _omitted, ...legacy } = validWorkItem
    expect(
      validateAgainstSchema(
        'work-item',
        { ...legacy, status: 'completed', completion: validCompletion },
        '',
      ),
    ).toEqual([])
  })

  it('rejects an out-of-enum risk', () => {
    const f = validateAgainstSchema('work-item', { ...validWorkItem, risk: 'catastrophic' }, '')
    expect(f.some((x) => x.message.includes('risk'))).toBe(true)
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

  it('accepts completion embedded in a completed work item', () => {
    const data = { ...validWorkItem, status: 'completed', completion: validCompletion }
    expect(validateAgainstSchema('work-item', data, '')).toEqual([])
  })

  it('tolerates fields only legacy completed records carry', () => {
    const data = {
      ...validWorkItem,
      status: 'completed',
      completion: { ...validCompletion, evidence: [{ id: 'go-test', kind: 'test' }] },
    }
    expect(validateAgainstSchema('work-item', data, '')).toEqual([])
  })

  it('requires completion when status is completed', () => {
    const data = { ...validWorkItem, status: 'completed' }
    const f = validateAgainstSchema('work-item', data, '')
    expect(f.some((x) => x.message.includes('completion'))).toBe(true)
  })

  it('requires initial context once the item is in progress', () => {
    const started = { ...validWorkItem, status: 'in_progress' }
    expect(validateAgainstSchema('work-item', started, '')).not.toEqual([])
    expect(
      validateAgainstSchema(
        'work-item',
        { ...started, evidence_policy: 'risk-based-v2', initial_context: validInitialContext },
        '',
      ),
    ).toEqual([])
  })

  it('requires the current evidence policy once the item is in progress', () => {
    const started = {
      ...validWorkItem,
      status: 'in_progress',
      initial_context: validInitialContext,
    }
    const f = validateAgainstSchema('work-item', started, '')
    expect(f.some((x) => x.message.includes('evidence_policy'))).toBe(true)
  })

  it('requires approval before medium-risk implementation starts', () => {
    const started = {
      ...validWorkItem,
      status: 'in_progress',
      risk: 'medium',
      evidence_policy: 'risk-based-v2',
      initial_context: validInitialContext,
    }
    expect(validateAgainstSchema('work-item', started, '')).not.toEqual([])
    expect(validateAgainstSchema('work-item', { ...started, approval: validApproval }, '')).toEqual(
      [],
    )
  })

  it('requires RED evidence when a policy-governed item completes', () => {
    const completed = {
      ...validWorkItem,
      status: 'completed',
      evidence_policy: 'risk-based-v1',
      completion: validCompletion,
    }
    expect(validateAgainstSchema('work-item', completed, '')).not.toEqual([])
    expect(
      validateAgainstSchema(
        'work-item',
        { ...completed, completion: { ...validCompletion, red_evidence: validRedEvidence } },
        '',
      ),
    ).toEqual([])
  })

  it('requires the evidence policy on ratcheted completed work items', () => {
    const completed = {
      ...validWorkItem,
      id: 'wi-410-demo',
      status: 'completed',
      completion: validCompletion,
    }
    const f = validateAgainstSchema('work-item', completed, '')
    expect(f.some((x) => x.message.includes('evidence_policy'))).toBe(true)
  })

  it('requires risk-based-v2 whenever work starts', () => {
    const started = {
      ...validWorkItem,
      id: 'wi-1-demo',
      status: 'in_progress',
      risk: 'medium',
      initial_context: validInitialContext,
      approval: validApproval,
    }
    expect(
      validateAgainstSchema('work-item', { ...started, evidence_policy: 'risk-based-v1' }, ''),
    ).not.toEqual([])
    expect(
      validateAgainstSchema('work-item', { ...started, evidence_policy: 'risk-based-v2' }, ''),
    ).toEqual([])
  })

  it('requires risk-based-v2 when wi-412 and later work completes', () => {
    const completed = {
      ...validWorkItem,
      id: 'wi-412-demo',
      status: 'completed',
      risk: 'medium',
      approval: validApproval,
      completion: {
        ...validCompletion,
        post_approval_changes: 'none',
        independent_verification: 'reviewed by a separate agent',
        change_resistance: 'the schema rejects the legacy policy fixture',
        acceptance_red_evidence: validRedEvidence,
        unit_red_evidence: validRedEvidence,
      },
    }
    expect(
      validateAgainstSchema('work-item', { ...completed, evidence_policy: 'risk-based-v1' }, ''),
    ).not.toEqual([])
    expect(
      validateAgainstSchema('work-item', { ...completed, evidence_policy: 'risk-based-v2' }, ''),
    ).toEqual([])
  })

  it('requires separate acceptance and unit RED evidence for risk-based-v2', () => {
    const completed = {
      ...validWorkItem,
      id: 'wi-412-demo',
      status: 'completed',
      risk: 'medium',
      evidence_policy: 'risk-based-v2',
      approval: validApproval,
      completion: {
        ...validCompletion,
        post_approval_changes: 'none',
        independent_verification: 'reviewed by a separate agent',
        change_resistance: 'the representative fault was detected',
      },
    }
    expect(
      validateAgainstSchema(
        'work-item',
        {
          ...completed,
          completion: {
            ...completed.completion,
            unit_red_evidence: validRedEvidence,
          },
        },
        '',
      ),
    ).not.toEqual([])
    expect(
      validateAgainstSchema(
        'work-item',
        {
          ...completed,
          completion: {
            ...completed.completion,
            acceptance_red_evidence: validRedEvidence,
          },
        },
        '',
      ),
    ).not.toEqual([])
    expect(
      validateAgainstSchema(
        'work-item',
        {
          ...completed,
          completion: {
            ...completed.completion,
            acceptance_red_evidence: validRedEvidence,
            unit_red_evidence: validRedEvidence,
          },
        },
        '',
      ),
    ).toEqual([])
  })

  it('requires an exact requirement id or an explained N/A for v2 RED evidence', () => {
    const completed = {
      ...validWorkItem,
      id: 'wi-412-demo',
      status: 'completed',
      evidence_policy: 'risk-based-v2',
      completion: {
        ...validCompletion,
        acceptance_red_evidence: {
          ...validRedEvidence,
          requirement: 'REQ-DEMO-001 trailing text',
        },
        unit_red_evidence: validRedEvidence,
      },
    }
    expect(validateAgainstSchema('work-item', completed, '')).not.toEqual([])
    expect(
      validateAgainstSchema(
        'work-item',
        {
          ...completed,
          completion: {
            ...completed.completion,
            acceptance_red_evidence: validRedEvidence,
            unit_red_evidence: {
              ...validRedEvidence,
              requirement: 'REQ-DEMO-001 trailing text',
            },
          },
        },
        '',
      ),
    ).not.toEqual([])
  })

  it('requires stronger completion evidence for medium-risk work', () => {
    const completed = {
      ...validWorkItem,
      status: 'completed',
      risk: 'medium',
      evidence_policy: 'risk-based-v1',
      approval: validApproval,
      completion: {
        ...validCompletion,
        red_evidence: validRedEvidence,
      },
    }
    expect(validateAgainstSchema('work-item', completed, '')).not.toEqual([])
    expect(
      validateAgainstSchema(
        'work-item',
        {
          ...completed,
          completion: {
            ...completed.completion,
            post_approval_changes: 'none',
            independent_verification: 'reviewed by a separate agent',
            change_resistance: 'the representative fault was detected',
          },
        },
        '',
      ),
    ).toEqual([])
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
