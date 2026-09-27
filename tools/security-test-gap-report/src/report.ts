export type GoTest = { path: string; name: string; body: string }

export type EffectEvidence =
  | 'read-back'
  | 'state comparison'
  | 'recorded effect'
  | 'protected representation absent'

export type SecurityTestGapReport = {
  refused: number
  gaps: Array<{ path: string; name: string }>
}

const MUTATING =
  /http\.Method(Post|Put|Patch|Delete)|\b(Create|Update|Delete|Cancel|Revoke|Disable|Enable|Kill|Rotate|Add|Remove|Write|Save|Issue|Approve|Deny|Bind|Unbind|Import|Apply)[A-Z]\w*\(/
const REFUSAL =
  /StatusForbidden|StatusUnauthorized|StatusConflict|StatusNotFound|StatusBadRequest|StatusUnprocessableEntity|Err\w*(Denied|Forbidden|Unauthorized|NotFound|AlreadyTerminal|LeaseLost|Unscoped|Mismatch)/
const READBACK = /\b(Get|Find|List|Load|Count|Lookup|Resolve)[A-Za-z]*\(|\.(Get|Find|List|Count)\(/

/** Cut top-level test functions out of a file; gofumpt closes them at column zero. */
export function testFunctions(source: string): Array<{ name: string; body: string }> {
  const out: Array<{ name: string; body: string }> = []
  const lines = source.split('\n')
  for (let i = 0; i < lines.length; i += 1) {
    const header = (lines[i] ?? '').match(/^func (Test\w+)\(t \*testing\.T\) \{/)
    if (!header) continue
    let end = i
    while (end < lines.length && lines[end] !== '}') end += 1
    out.push({ name: header[1] ?? '', body: lines.slice(i, end + 1).join('\n') })
    i = end
  }
  return out
}

export function effectEvidenceIn(body: string): EffectEvidence[] {
  return READBACK.test(body) ? ['read-back'] : []
}

export function reportSecurityTestGaps(tests: GoTest[]): SecurityTestGapReport {
  const report: SecurityTestGapReport = { refused: 0, gaps: [] }
  for (const test of tests) {
    if (!REFUSAL.test(test.body) || !MUTATING.test(test.body)) continue
    report.refused += 1
    if (effectEvidenceIn(test.body).length === 0) {
      report.gaps.push({ path: test.path, name: test.name })
    }
  }
  return report
}

export function renderReport(report: SecurityTestGapReport, list: boolean): string {
  const byArea = new Map<string, number>()
  for (const gap of report.gaps) {
    const area = gap.path.split('/').slice(0, 3).join('/')
    byArea.set(area, (byArea.get(area) ?? 0) + 1)
  }

  const lines = [
    `refused state changes covered by a test : ${report.refused}`,
    `... with no read-back proving no effect : ${report.gaps.length}`,
    '',
  ]
  for (const [area, count] of [...byArea.entries()].sort((a, b) => b[1] - a[1])) {
    lines.push(`${String(count).padStart(4)}  ${area}`)
  }
  if (list) {
    lines.push('')
    for (const gap of report.gaps) lines.push(`${gap.path}  ${gap.name}`)
  }
  return lines.join('\n')
}
