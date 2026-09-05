import { describe, expect, it } from 'bun:test'
import { checkNormativeCoverage, citedNormativeIds } from './normative-coverage.ts'

const DEBT = 'tools/check/scenario-coverage-debt.json'

const declared = [
  { id: 'REQ-DEMO-001', path: 'docs/contexts/demo/scenarios.md' },
  { id: 'REQ-DEMO-002', path: 'docs/contexts/demo/scenarios.md' },
]

const messages = (findings: Array<{ message: string }>) =>
  findings.map((finding) => finding.message)

describe('citedNormativeIds', () => {
  it('reads both scenario and standard ids out of a test source', () => {
    const cited = citedNormativeIds(
      [
        'func TestAuthorize(t *testing.T) { // REQ-OAUTH2-001, WCAG22-KEYBOARD }',
        "it('keeps the audit record', () => { /* NIST63B4-PASSWORD-MINIMUM */ })",
      ],
      ['REQ-OAUTH2-001', 'REQ-OAUTH2-002', 'WCAG22-KEYBOARD', 'NIST63B4-PASSWORD-MINIMUM'],
    )
    expect([...cited].sort()).toEqual([
      'NIST63B4-PASSWORD-MINIMUM',
      'REQ-OAUTH2-001',
      'WCAG22-KEYBOARD',
    ])
  })

  // Thirteen SAML and WS-Federation rows are named this way, and a shape that
  // expected upper-case segments could never have credited any of them.
  it('reads a mixed-case standard id', () => {
    const cited = citedNormativeIds(
      ['func TestBearerAssertion(t *testing.T) { // SAML2Core-BearerAssertion }'],
      ['SAML2Core-BearerAssertion', 'WSFed-PassiveSignIn'],
    )
    expect([...cited]).toEqual(['SAML2Core-BearerAssertion'])
  })

  it('does not let a longer id count as a mention of its prefix', () => {
    const cited = citedNormativeIds(
      ['// REQ-DEMO-0011 and RFC6750-API-TOKEN-HEADER-EXTRA'],
      ['REQ-DEMO-001', 'REQ-DEMO-0011', 'RFC6750-API-TOKEN-HEADER'],
    )
    expect([...cited]).toEqual(['REQ-DEMO-0011'])
  })

  it('reads nothing when the specification declares nothing', () => {
    expect([...citedNormativeIds(['// REQ-DEMO-001'], [])]).toEqual([])
  })
})

describe('checkNormativeCoverage', () => {
  it('accepts a declaration a test names', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001', 'REQ-DEMO-002']),
      debt: [],
      debtPath: DEBT,
    })
    expect(findings).toEqual([])
  })

  it('rejects a declaration no test names and no debt entry covers', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      debt: [],
      debtPath: DEBT,
    })
    expect(findings).toHaveLength(1)
    expect(findings[0]?.path).toBe('docs/contexts/demo/scenarios.md')
    expect(findings[0]?.message).toContain('REQ-DEMO-002')
    expect(findings[0]?.message).toContain('no test names it')
  })

  it('accepts an untested declaration the debt file carries with a reason', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      debt: [{ id: 'REQ-DEMO-002', reason: 'present when the check was introduced' }],
      debtPath: DEBT,
    })
    expect(findings).toEqual([])
  })

  it('rejects a debt entry with no reason', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      debt: [{ id: 'REQ-DEMO-002', reason: '  ' }],
      debtPath: DEBT,
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-002 is listed without a reason. State why it has no test yet.',
    ])
    expect(findings[0]?.path).toBe(DEBT)
  })

  it('rejects a debt entry that has grown a test, so the list only shrinks', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001', 'REQ-DEMO-002']),
      debt: [{ id: 'REQ-DEMO-002', reason: 'present when the check was introduced' }],
      debtPath: DEBT,
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-002 now has a test that names it. Remove it from the list; the list only shrinks.',
    ])
  })

  it('rejects a debt entry nothing declares any more', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001', 'REQ-DEMO-002']),
      debt: [{ id: 'REQ-DEMO-404', reason: 'present when the check was introduced' }],
      debtPath: DEBT,
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-404 is listed as untested but nothing declares it any more. Remove it.',
    ])
  })

  it('rejects a debt entry listed twice', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      debt: [
        { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
        { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
      ],
      debtPath: DEBT,
    })
    expect(messages(findings)).toEqual(['REQ-DEMO-002 is listed twice. Keep one entry per id.'])
  })

  it('rejects a debt file that is not in id order', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(),
      debt: [
        { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
        { id: 'REQ-DEMO-001', reason: 'present when the check was introduced' },
      ],
      debtPath: DEBT,
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-001 is listed after REQ-DEMO-002. Keep the list in id order so its diffs stay readable.',
    ])
  })

  // The refusal debt and this list are meant to be two disjoint lists, not two
  // overlapping ones: an id in both has to be removed from both when a test
  // finally names it, and the pair drifts the first time only one is edited.
  it('accepts an untested declaration another list already accounts for', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      debt: [],
      debtPath: DEBT,
      accounted: new Set(['REQ-DEMO-002']),
      accountedPath: 'tools/check/security-refusal-debt.json',
    })
    expect(findings).toEqual([])
  })

  it('rejects a debt entry another list already accounts for', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      debt: [{ id: 'REQ-DEMO-002', reason: 'present when the check was introduced' }],
      debtPath: DEBT,
      accounted: new Set(['REQ-DEMO-002']),
      accountedPath: 'tools/check/security-refusal-debt.json',
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-002 is already listed in tools/check/security-refusal-debt.json. Keep the two lists disjoint.',
    ])
  })
})
