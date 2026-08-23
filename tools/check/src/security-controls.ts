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
 * a declared refusal visible instead of silent. R4 asks the question R3 cannot:
 * not whether a declared refusal is tested, but whether the refusal the contract
 * already promises was declared at all.
 *
 * R1 and R2 first named `WriteProblem`, and naming one writer is what let the
 * shape back out: four admin guards returned `WriteAdminAccessError`, which
 * returned `WriteProblem`, and one call of distance was enough to be invisible.
 * The writer is now derived rather than named -- see responseWriters.
 */

export type GoFile = { path: string; source: string }

export type Finding = { path: string; rule: 'R1' | 'R2' | 'R3' | 'R4'; message: string }

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
 * echo's own response methods: the point where a response actually goes on the
 * wire. The seed is the framework's boundary rather than one of this
 * repository's writer names, because choosing a name of ours is what needs
 * revisiting every time someone wraps it.
 */
const ECHO_WRITERS =
  /^(?:JSON|JSONPretty|JSONBlob|JSONP|XML|XMLBlob|HTML|HTMLBlob|String|Blob|Stream|File|Attachment|Inline|NoContent|Redirect|Render)$/

/**
 * The name the `*echo.Context` arrives under, so `c.JSON(...)` is told from
 * `enc.JSON(...)`. A method named like a writer on anything else is an encoder
 * or a DTO, not a response.
 */
function contextParam(signature: string): string | undefined {
  return signature.match(/(\w+)\s+\*echo\.Context/)?.[1]
}

type Call = { receiver?: string; name: string }

/** The calls whose result a function hands straight back. */
function returnedCalls(body: string): Call[] {
  return [...body.matchAll(/return\s+(?:([\w.]+)\.)?(\w+)\s*\(/g)].map((match) => ({
    receiver: match[1],
    name: match[2] ?? '',
  }))
}

function isEchoWrite(signature: string, call: Call): boolean {
  const ctx = contextParam(signature)
  return ctx !== undefined && call.receiver === ctx && ECHO_WRITERS.test(call.name)
}

/**
 * The functions that can hand a caller the result of writing a response.
 *
 * A fixpoint from echo's writers outward: a function joins the set when one of
 * its returns hands back a call to something already in it. `WriteProblem` is
 * not privileged, it is derived through `NoStoreJSON` down to `c.JSON`, and so
 * is any helper written on top of it tomorrow.
 *
 * Existence, not universality. Collecting only functions whose every return is
 * a write reads tidier and would have missed the case this exists for: the
 * `WriteAdminAccessError` that shipped ended in `return err` for the errors it
 * could not map, and that one honest branch would have exempted the three that
 * handed back a nil.
 *
 * Returns each writer with the call that put it in the set, so a finding can
 * show the chain down to the framework instead of just its first link.
 */
export function responseWriters(files: GoFile[]): Map<string, string> {
  const all = files.flatMap((file) => functions(file.source))
  const writers = new Map<string, string>()
  for (;;) {
    let grew = false
    for (const fn of all) {
      if (writers.has(fn.name)) continue
      for (const call of returnedCalls(fn.body)) {
        const direct = isEchoWrite(fn.signature, call)
        if (!direct && !writers.has(call.name)) continue
        writers.set(fn.name, direct ? `${call.receiver}.${call.name}` : call.name)
        grew = true
        break
      }
    }
    if (!grew) return writers
  }
}

/** `requireWorkflowAdmin -> WriteAdminAccessError -> WriteProblem -> c.JSON`. */
function writerChain(name: string, writers: Map<string, string>): string {
  const chain = [name]
  for (let next = writers.get(name); next !== undefined; next = writers.get(next)) {
    if (chain.includes(next)) break
    chain.push(next)
  }
  return chain.join(' -> ')
}

/**
 * R1: a guard must report its refusal through the return value.
 *
 * `return WriteProblem(...)` hands back what writing the response returned,
 * which is nil on success. A guard that does this has written "denied" to the
 * client and told its caller to carry on -- and so has a guard that returns
 * anything else which ends up doing the same, however many calls away.
 */
function checkGuardsReportRefusal(
  files: GoFile[],
  guards: Set<string>,
  writers: Map<string, string>,
): Finding[] {
  const findings: Finding[] = []
  for (const file of files) {
    for (const fn of functions(file.source)) {
      if (!guards.has(fn.name) || !isRequestScoped(fn.signature)) continue
      for (const call of returnedCalls(fn.body)) {
        const direct = isEchoWrite(fn.signature, call)
        if (!direct && !writers.has(call.name)) continue
        const chain = direct ? `${call.receiver}.${call.name}` : writerChain(call.name, writers)
        findings.push({
          path: file.path,
          rule: 'R1',
          message:
            `${fn.name} is used as a guard but returns the result of ${chain}, which is nil ` +
            'once the response is written. Write the response, then return a non-nil error so ' +
            'the caller stops.',
        })
        break
      }
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
function requestGuardNames(
  files: GoFile[],
  guards: Set<string>,
  writers: Map<string, string>,
): Set<string> {
  const names = new Set<string>()
  for (const file of files) {
    for (const fn of functions(file.source)) {
      if (!guards.has(fn.name) || !isRequestScoped(fn.signature)) continue
      // "Writes a refusal" is asked of the writer set too, for the same reason
      // R1 asks it: a guard that refuses through a helper is still a guard, and
      // one that only ever named WriteProblem stopped seeing it there.
      const calls = [...fn.body.matchAll(/(?:([\w.]+)\.)?(\w+)\s*\(/g)].map((match) => ({
        receiver: match[1],
        name: match[2] ?? '',
      }))
      if (!calls.some((call) => writers.has(call.name) || isEchoWrite(fn.signature, call))) continue
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

/**
 * Signals that a scenario step declares a refusal.
 *
 * Naming an error type is the sturdy one: the specification's errors are
 * declared in TypeSpec and written into the step, so it survives rephrasing.
 * The vocabulary is the fallback for a refusal stated as an outcome rather than
 * an error — "treated as nonexistent", "no token is issued".
 */
const REFUSAL_ERROR = /\b[A-Z][A-Za-z0-9]*Error\b/
const REFUSAL_WORDS =
  /(拒否|却下|禁じ|認めな|できない|返らな|見えな|含まれな|発行しない|失敗させる|存在しないものとして|Denied|Unauthorized|Forbidden)/

/**
 * The scenarios that declare a refusal.
 *
 * Both `ALT` and `THEN` are read. An earlier version looked only at `ALT` and
 * matched a list of keywords, and it missed two whole shapes: a refusal that is
 * the scenario's own outcome rather than an alternative
 * (REQ-SIGNINGKEYS-009, "THEN AccessDeniedError で拒否される"), and a tenant
 * boundary phrased as an outcome (REQ-IDGOVERNANCE-012, "THEN ワークフローは
 * 存在しないものとして扱われる").
 *
 * No attempt is made to sort a refusal an authorization control owns from one
 * an input validation owns. Drawing that line was what made the first version
 * fragile, and the property being checked — a declared refusal has a test that
 * names it — is worth having either way.
 */
function refusalSteps(scenarios: string): Array<{ id: string; line: string }> {
  const steps: Array<{ id: string; line: string }> = []
  let current = ''
  for (const line of scenarios.split('\n')) {
    const heading = line.match(/^### (REQ-[A-Z0-9]+-\d+)/)
    if (heading) {
      current = heading[1] ?? ''
      continue
    }
    if (!current) continue
    const step = /^\s+- ALT /.test(line) || /^- THEN /.test(line)
    if (!step) continue
    if (!REFUSAL_ERROR.test(line) && !REFUSAL_WORDS.test(line)) continue
    steps.push({ id: current, line })
  }
  return steps
}

export function refusalScenarioIds(scenarios: string): string[] {
  const ids: string[] = []
  for (const step of refusalSteps(scenarios)) {
    if (!ids.includes(step.id)) ids.push(step.id)
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
        `${id} declares a refusal, but no test names it. ` +
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

/**
 * The refusals the contract promises for the operations that change state.
 *
 * TypeSpec already says, per operation, which error body a 403 carries, and a
 * 403 on a state-changing operation is the contract stating who may not perform
 * it. That is the join R4 needs: the error type is written on both sides — in
 * TypeSpec as the response body, in the scenario as the name of what refuses —
 * so no reference has to be invented for the scenario to carry.
 *
 * Returns error type -> the operations that answer 403 with it.
 */
export function contractRefusalsOfStateChanges(typespec: string): Map<string, string[]> {
  const bodies = new Map<string, string[]>()
  for (const union of typespec.matchAll(/union\s+(\w+)Error(\d{3})Body\s*\{([^}]*)\}/g)) {
    const types = [...(union[3] ?? '').matchAll(/\b(\w+Error)\b/g)].map((match) => match[1] ?? '')
    bodies.set(`${union[1]}:${union[2]}`, types)
  }
  const refusals = new Map<string, string[]>()
  for (const op of typespec.matchAll(/@(get|post|put|patch|delete)\s*\nop\s+(\w+)\s*\(/g)) {
    if (op[1] === 'get') continue
    for (const type of bodies.get(`${op[2]}:403`) ?? []) {
      refusals.set(type, [...(refusals.get(type) ?? []), op[2] ?? ''])
    }
  }
  return refusals
}

/** The error types the scenarios name in a step that refuses. */
export function declaredRefusalTypes(scenarios: string): Set<string> {
  const types = new Set<string>()
  for (const step of refusalSteps(scenarios)) {
    for (const match of step.line.matchAll(/\b([A-Z][A-Za-z0-9]*Error)\b/g)) types.add(match[1] ?? '')
  }
  return types
}

/**
 * R4: a refusal the contract promises must be declared as behavior.
 *
 * R3 checks the refusals that were written down. A context that writes none
 * passes it without being asked anything, and that is the hole: a 403 on a
 * state-changing operation is a control the product relies on, and if no
 * scenario says when it fires, an implementation that stops refusing
 * contradicts nothing.
 *
 * The judgment is per context, not per operation, so a refusal declared for a
 * read satisfies the state-changing operations that answer with the same type.
 * Reaching per-operation needs a link between an operation and a scenario that
 * neither side carries today; see wi-391 for the measurement.
 */
export function checkContractRefusalsAreDeclared(
  context: string,
  contract: Map<string, string[]>,
  declared: Set<string>,
): Finding[] {
  const findings: Finding[] = []
  for (const [type, operations] of contract) {
    if (declared.has(type)) continue
    findings.push({
      path: `spec/contexts/${context}/scenarios.md`,
      rule: 'R4',
      message:
        `${operations.join(', ')} answer 403 with ${type}, but no scenario declares that refusal. ` +
        'State when the operation refuses and what the refusal leaves untouched.',
    })
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
  const writers = responseWriters(production)
  return [
    ...checkGuardsReportRefusal(production, guards, writers),
    ...checkGuardResultsAreUsed(production, requestGuardNames(production, guards, writers)),
  ]
}
