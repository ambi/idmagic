/**
 * Check that security controls can actually refuse, and that every refusal the
 * specification declares is reached by a test.
 *
 * The rules here exist because of a defect that survived review, a passing test
 * suite, and full coverage of the offending line: `VerifyBrowserRequest` wrote
 * its 403 and returned the result of writing it, which is nil. Every caller
 * guarded with `if err := ...; err != nil { return err }`, so none of them
 * stopped, and a refused request went on to commit its side effect while the
 * client read a 403.
 *
 * R1 and R2 make that shape unrepresentable. R3 makes the absence of a test for
 * a declared refusal visible instead of silent.
 */

export type GoFile = { path: string; source: string }

export type Finding = { path: string; rule: 'R1' | 'R2' | 'R3'; message: string }

/**
 * A function used as a guard: somewhere a caller writes
 * `if err := f(...); err != nil { ... }` and decides whether to continue from
 * the result. Whether a function is a guard is not a property of its name —
 * `handle*` and `write*Error` also write a response and return, and that is
 * correct for them. It is a property of how callers use the value.
 */
function guardCallees(files: GoFile[]): Set<string> {
  const names = new Set<string>()
  // `if err := <recv>.<name>(` and `if err := <name>(`, plus the `_, err :=`
  // form some guards use. gofumpt keeps these on one line.
  const pattern =
    /if\s+(?:[\w,\s]+,\s*)?err\s*:?=\s*(?:[\w.]+\.)?(\w+)\s*\([^\n]*;\s*err\s*!=\s*nil/g
  for (const file of files) {
    for (const match of file.source.matchAll(pattern)) {
      const name = match[1]
      if (name) names.add(name)
    }
  }
  return names
}

/**
 * Split a Go file into top-level function bodies. gofumpt guarantees a
 * top-level declaration closes with `}` in column zero, which is enough to cut
 * a body without parsing the language.
 */
function functions(source: string): Array<{ name: string; signature: string; body: string }> {
  const out: Array<{ name: string; signature: string; body: string }> = []
  const lines = source.split('\n')
  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i] ?? ''
    const header = line.match(/^func\s+(?:\([^)]*\)\s*)?(\w+)\s*\(/)
    if (!header) continue
    const name = header[1] ?? ''
    let end = i
    while (end < lines.length && lines[end] !== '}') end += 1
    out.push({ name, signature: line, body: lines.slice(i, end + 1).join('\n') })
    i = end
  }
  return out
}

/** Whether a signature takes the HTTP context and answers with an error. */
function isRequestScoped(signature: string): boolean {
  return /\*echo\.Context/.test(signature) && /\)\s*error\s*\{/.test(signature)
}

/**
 * R1: a guard must report its refusal through the return value.
 *
 * `return WriteProblem(...)` hands back what writing the response returned,
 * which is nil on success. A guard that does this has written "denied" to the
 * client and told its caller to carry on.
 */
function checkGuardsReportRefusal(files: GoFile[], guards: Set<string>): Finding[] {
  const findings: Finding[] = []
  for (const file of files) {
    for (const fn of functions(file.source)) {
      if (!guards.has(fn.name) || !isRequestScoped(fn.signature)) continue
      if (!/return\s+(?:\w+\.)?WriteProblem\s*\(/.test(fn.body)) continue
      findings.push({
        path: file.path,
        rule: 'R1',
        message:
          `${fn.name} is used as a guard but returns the result of WriteProblem, which is nil. ` +
          'Write the response, then return a non-nil error so the caller stops.',
      })
    }
  }
  return findings
}

/**
 * The guards R2 cares about: not merely "called in guard position somewhere"
 * but request-scoped functions that write a refusal. Matching on the name alone
 * is far too coarse -- `Save`, `Revoke`, and `Delete` are guarded on one type
 * and discarded on another, and neither has anything to do with refusing a
 * request.
 */
function requestGuardNames(files: GoFile[], guards: Set<string>): Set<string> {
  const names = new Set<string>()
  for (const file of files) {
    for (const fn of functions(file.source)) {
      if (!guards.has(fn.name) || !isRequestScoped(fn.signature)) continue
      if (!/(?:\w+\.)?WriteProblem\s*\(/.test(fn.body)) continue
      names.add(fn.name)
    }
  }
  return names
}

/**
 * R2: a guard's result must not be discarded.
 *
 * If one call site decides whether to continue from this value and another
 * throws it away, one of the two is wrong, and the one throwing it away is
 * running past a refusal.
 */
function checkGuardResultsAreUsed(files: GoFile[], guards: Set<string>): Finding[] {
  const findings: Finding[] = []
  for (const file of files) {
    for (const line of file.source.split('\n')) {
      const discarded = line.match(/^\s*_\s*=\s*(?:[\w.]+\.)?(\w+)\s*\(/)
      const name = discarded?.[1]
      if (!name || !guards.has(name)) continue
      findings.push({
        path: file.path,
        rule: 'R2',
        message:
          `the result of ${name} is discarded here, but elsewhere it decides whether the ` +
          'request continues. A refusal thrown away is a refusal that did not happen.',
      })
    }
  }
  return findings
}

/** A scenario id whose ALT branches declare a refusal a security control owns. */
const REFUSAL =
  /(拒否|却下|失敗させる|Denied|denied|Unauthorized|Forbidden|AccessDenied|存在しないものとして)/

export function refusalScenarioIds(scenarios: string): string[] {
  const ids: string[] = []
  let current = ''
  for (const line of scenarios.split('\n')) {
    const heading = line.match(/^### (REQ-[A-Z0-9]+-\d+)/)
    if (heading) {
      current = heading[1] ?? ''
      continue
    }
    if (!current) continue
    if (/^\s+- ALT /.test(line) && REFUSAL.test(line) && !ids.includes(current)) ids.push(current)
  }
  return ids
}

/**
 * R3: a declared refusal must be reached by a test that names it.
 *
 * `allowed` carries the refusals that had no test when this check was
 * introduced. It is a ratchet, not an exemption: an id missing from it must
 * have a test, and an id on it that has grown one has to come off, so the list
 * can only shrink.
 */
export function checkRefusalCoverage(
  declared: string[],
  citedByTests: Set<string>,
  allowed: string[],
): Finding[] {
  const findings: Finding[] = []
  const allowedSet = new Set(allowed)
  for (const id of declared) {
    if (citedByTests.has(id) || allowedSet.has(id)) continue
    findings.push({
      path: 'spec/contexts',
      rule: 'R3',
      message:
        `${id} declares a refusal a security control owns, but no test names it. ` +
        'Cite the id from the test that exercises the refusal.',
    })
  }
  for (const id of allowed) {
    if (!declared.includes(id)) {
      findings.push({
        path: 'tools/check/security-refusal-debt.json',
        rule: 'R3',
        message: `${id} is listed as an untested refusal but no longer declares one. Remove it.`,
      })
      continue
    }
    if (citedByTests.has(id)) {
      findings.push({
        path: 'tools/check/security-refusal-debt.json',
        rule: 'R3',
        message: `${id} now has a test that names it. Remove it from the list; the list only shrinks.`,
      })
    }
  }
  return findings
}

/** Run the guard rules (R1, R2) over the non-test Go sources. */
export function checkSecurityGuards(files: GoFile[]): Finding[] {
  const production = files.filter((file) => !file.path.endsWith('_test.go'))
  // Guard positions are read from production code only. A test may call a
  // response writer inside `if err := ...` to assert what it returns, which
  // says nothing about how the product uses it.
  const guards = guardCallees(production)
  return [
    ...checkGuardsReportRefusal(production, guards),
    ...checkGuardResultsAreUsed(production, requestGuardNames(production, guards)),
  ]
}
