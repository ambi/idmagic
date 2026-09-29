export type GoTest = { path: string; name: string; body: string }

export type EffectEvidence =
  | 'read-back'
  | 'state comparison'
  | 'recorded effect'
  | 'protected representation absent'
  | 'asserted observation'
  | 'effect assertion'

export type SecurityTestGapReport = {
  refused: number
  gaps: Array<{ path: string; name: string }>
}

const MUTATING =
  /http\.Method(Post|Put|Patch|Delete)|\b(Create|Update|Delete|Cancel|Revoke|Disable|Enable|Kill|Rotate|Add|Remove|Write(?!Header)|Save|Issue|Approve|Deny|Bind|Unbind|Import|Apply)[A-Z]\w*\(/
const REFUSAL =
  /StatusForbidden|StatusUnauthorized|StatusConflict|StatusNotFound|StatusBadRequest|StatusUnprocessableEntity|Err\w*(Denied|Forbidden|Unauthorized|NotFound|AlreadyTerminal|LeaseLost|Unscoped|Mismatch)/
const READBACK = /\b(Get|Find|List|Load|Count|Lookup|Resolve)[A-Za-z]*\(|\.(Get|Find|List|Count)\(/
const STATE_COMPARISON =
  /\bif\b[^\n{]*(?:\b\w*before\w*\b[^\n{]*\b\w*after\w*\b|\b\w*after\w*\b[^\n{]*\b\w*before\w*\b)/i
const RECORDED_EFFECT =
  /\bif\b[^\n{]*(?:len\([^\n)]*(?:\.(?:Sent|Calls|Requests|Events|Published|Delivered|Jobs|Artifacts|Streams|Saved|ApplicationIDs|ResourceServers|TrustBundles|Tokens|Types)|\b(?:events|calls|requests|jobs|artifacts)\b)[^\n)]*\)|\.(?:Sent|Calls|Requests|Events|Published|Delivered|Jobs|Artifacts|Streams|Saved|ApplicationIDs|ResourceServers|TrustBundles|Tokens|Types)\b)/i
const PROTECTED_REPRESENTATION_ABSENT =
  /\bif\s+(?:(?:\w+)\s*:=\s*[^;\n]*(?:Body|body)[^;\n]*;\s*)?(?:strings|bytes)\.Contains\([^\n]*(?:Body|body|\w+)[^\n]*\)\s*\{[^}]*\bt\.(?:Fatal|Error)/s
const EFFECT_ASSERTION =
  /\bassert[A-Za-z]*(?:No|Not|Unchanged|Absent|Empty|Indistinguishable)[A-Za-z]*\(/

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
  const source = body
    .split('\n')
    .filter((line) => !line.trimStart().startsWith('//'))
    .join('\n')
  const evidence: EffectEvidence[] = []
  if (READBACK.test(source)) evidence.push('read-back')
  if (STATE_COMPARISON.test(source)) evidence.push('state comparison')
  if (RECORDED_EFFECT.test(source)) evidence.push('recorded effect')
  if (PROTECTED_REPRESENTATION_ABSENT.test(source)) evidence.push('protected representation absent')
  if (hasAssertedObservation(source)) evidence.push('asserted observation')
  if (EFFECT_ASSERTION.test(source)) evidence.push('effect assertion')
  return evidence
}

const responseValues =
  /^(?:res|resp|response|rec|recorder|refused|problem|result|err|code|status|body|create|update|deleted|accepted|request)$/i
const responseOperations =
  /(?:send|request|serveHTTP|decode|unmarshal|problem|newRecorder|newRequest)$/i

function hasAssertedObservation(source: string): boolean {
  const inlineAssertions = source.matchAll(
    /\bif\s+(\w+)\s*:=\s*([\w.]+)\([^\n;]*\);[^\n{]*\{[^}]*\bt\.(?:Fatal|Error)/gs,
  )
  for (const assertion of inlineAssertions) {
    const variable = assertion[1] ?? ''
    const operation = assertion[2] ?? ''
    if (!responseValues.test(variable) && !responseOperations.test(operation)) return true
  }

  const twoValueAssignments = source.matchAll(
    /^\s*(?:\w+|_)\s*,\s*(\w+)\s*:=\s*([\w.]+)\([^\n]*\)$/gm,
  )
  for (const assignment of twoValueAssignments) {
    const variable = assignment[1] ?? ''
    const operation = assignment[2] ?? ''
    if (responseValues.test(variable) || responseOperations.test(operation)) continue
    const remainder = source.slice((assignment.index ?? 0) + assignment[0].length)
    const asserted = new RegExp(
      `\\bif\\b[^\\n{]*\\b${variable}\\b[^\\n{]*\\{[^}]*\\bt\\.(?:Fatal|Error)`,
      's',
    )
    if (asserted.test(remainder)) return true
  }

  const assignments = source.matchAll(/^\s*(\w+)\s*:=\s*([\w.]+)\([^\n]*\)(?:\.\w+)*$/gm)
  for (const assignment of assignments) {
    const variable = assignment[1] ?? ''
    const operation = assignment[2] ?? ''
    if (responseValues.test(variable) || responseOperations.test(operation)) continue
    const remainder = source.slice((assignment.index ?? 0) + assignment[0].length)
    const asserted = new RegExp(
      `\\bif\\b[^\\n{]*\\b${variable}\\b[^\\n{]*\\{[^}]*\\bt\\.(?:Fatal|Error)`,
      's',
    )
    if (asserted.test(remainder)) return true
  }
  return false
}

export function reportSecurityTestGaps(tests: GoTest[]): SecurityTestGapReport {
  const report: SecurityTestGapReport = { refused: 0, gaps: [] }
  for (const test of tests) {
    if (
      test.path.startsWith('backend/provisioning/client_scim/') ||
      test.path === 'backend/oauth2/handlers_http/client_auth_test.go' ||
      (test.path === 'backend/sourcing/scim/handlers_http/scim_test.go' &&
        test.name === 'TestScimListUsersDateTimeFilterAndURNPrefix')
    )
      continue
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
