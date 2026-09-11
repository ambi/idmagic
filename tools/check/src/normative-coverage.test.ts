import { describe, expect, it } from 'bun:test'
import { checkNormativeCoverage, citedNormativeIds } from './normative-coverage.ts'

const DEBT = 'tools/check/example-coverage-debt.json'

const declared = [
  { id: 'REQ-DEMO-001', path: 'docs/contexts/demo/scenarios.feature.md' },
  { id: 'REQ-DEMO-002', path: 'docs/contexts/demo/scenarios.feature.md' },
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

  // 例の住所は末尾に連番を足して作るため、隣の id を部分文字列として含む。
  // 前後どちらの端でも、別の id の一部を引用と数えてはならない。
  it('does not let a longer id count as a mention of the id it ends with', () => {
    const cited = citedNormativeIds(
      ['// EX-DEMO-001-01 の前段として PRE-EX-DEMO-001-02 を用意する'],
      ['EX-DEMO-001-01', 'EX-DEMO-001-02'],
    )
    expect([...cited]).toEqual(['EX-DEMO-001-01'])
  })

  it('reads nothing when the specification declares nothing', () => {
    expect([...citedNormativeIds(['// REQ-DEMO-001'], [])]).toEqual([])
  })

  // 旧形式の粗い追跡を名前だけ変えて残さないための境界。親を引用しただけの
  // テストが、その規則に属する具体例まで確認したことにはならない。
  it('does not let a rule citation cover the examples under it', () => {
    const cited = citedNormativeIds(
      ['func TestExchange(t *testing.T) { // REQ-OAUTH2-005 }'],
      ['REQ-OAUTH2-005', 'EX-OAUTH2-005-01', 'EX-OAUTH2-005-02'],
    )
    expect([...cited]).toEqual(['REQ-OAUTH2-005'])
  })

  it('credits every example a single table-driven test names', () => {
    const cited = citedNormativeIds(
      [
        'func TestPromptNone(t *testing.T) {\n' +
          '\tcases := []struct{ id string }{\n' +
          '\t\t{id: "EX-OAUTH2-005-03"},\n' +
          '\t\t{id: "EX-OAUTH2-005-05"},\n' +
          '\t}\n}',
      ],
      ['EX-OAUTH2-005-03', 'EX-OAUTH2-005-04', 'EX-OAUTH2-005-05'],
    )
    expect([...cited].sort()).toEqual(['EX-OAUTH2-005-03', 'EX-OAUTH2-005-05'])
  })
})

describe('checkNormativeCoverage', () => {
  it('accepts a declaration a test names', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001', 'REQ-DEMO-002']),
      ledger: { entries: [], path: DEBT },
    })
    expect(findings).toEqual([])
  })

  it('rejects a declaration no test names and no debt entry covers', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      ledger: { entries: [], path: DEBT },
    })
    expect(findings).toHaveLength(1)
    expect(findings[0]?.path).toBe('docs/contexts/demo/scenarios.feature.md')
    expect(findings[0]?.message).toContain('REQ-DEMO-002')
    expect(findings[0]?.message).toContain('no test names it')
  })

  // 標準の側は台帳を持たない (wi-495)。台帳が無い検査は、逃げ道を案内しては
  // ならない。存在しないファイルへ載せろと言う指示は、読み手を実在しない手順へ
  // 送るうえ、そのファイルを作れば通ると誤解させる。
  it('names no escape hatch when the caller has no debt ledger', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
    })
    expect(findings).toHaveLength(1)
    expect(findings[0]?.message).toContain('REQ-DEMO-002')
    expect(findings[0]?.message).toContain('no test names it')
    expect(findings[0]?.message).not.toContain('debt')
    expect(findings[0]?.message).not.toContain('list it in')
  })

  it('accepts an untested declaration the debt file carries with a reason', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      ledger: {
        entries: [{ id: 'REQ-DEMO-002', reason: 'present when the check was introduced' }],
        path: DEBT,
      },
    })
    expect(findings).toEqual([])
  })

  it('rejects a debt entry with no reason', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      ledger: { entries: [{ id: 'REQ-DEMO-002', reason: '  ' }], path: DEBT },
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
      ledger: {
        entries: [{ id: 'REQ-DEMO-002', reason: 'present when the check was introduced' }],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-002 now has a test that names it. Remove it from the list; the list only shrinks.',
    ])
  })

  it('rejects a debt entry nothing declares any more', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001', 'REQ-DEMO-002']),
      ledger: {
        entries: [{ id: 'REQ-DEMO-404', reason: 'present when the check was introduced' }],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-404 is listed as untested but nothing declares it any more. Remove it.',
    ])
  })

  it('rejects a debt entry listed twice', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      ledger: {
        entries: [
          { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
          { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
        ],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual(['REQ-DEMO-002 is listed twice. Keep one entry per id.'])
  })

  it('rejects a debt file that is not in id order', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(),
      ledger: {
        entries: [
          { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
          { id: 'REQ-DEMO-001', reason: 'present when the check was introduced' },
        ],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual([
      'REQ-DEMO-001 is listed after REQ-DEMO-002. Keep the list in id order so its diffs stay readable.',
    ])
  })

  // One id, one list. The refusal-shaped half of this comparison used to live
  // in security-controls.ts with its own debt file, and the two were kept
  // disjoint by reading each other. wi-490 measured what the split bought and
  // folded it back in: an id is listed here or it has a test, and nothing else
  // accounts for it.
  it('rejects the same id listed twice in one list', () => {
    const findings = checkNormativeCoverage({
      declared,
      cited: new Set(['REQ-DEMO-001']),
      ledger: {
        entries: [
          { id: 'REQ-DEMO-002', reason: 'present when the check was introduced' },
          { id: 'REQ-DEMO-002', reason: 'declared a refusal when the refusal check arrived' },
        ],
        path: DEBT,
      },
    })
    expect(messages(findings)).toEqual(['REQ-DEMO-002 is listed twice. Keep one entry per id.'])
  })
})
