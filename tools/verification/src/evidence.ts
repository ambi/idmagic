import type { OperationVerification } from '../../check/src/operation-verification.ts'
import { specCitingTests } from '../../spec-review-candidates/src/candidates.ts'

export type ExecutionState = 'passed' | 'failed' | 'skipped' | 'not-run'
export type GoTestEvent = { Action: string; Package: string; Test?: string; Output?: string }
export type TestExecution = { package: string; name: string; state: ExecutionState }
export type TestReference = {
  package: string
  name: string
  path: string
  normativeIds: string[]
  operationIds: string[]
}
export type ReferenceResult = TestReference & { state: ExecutionState }
export type DBVerification = {
  package: string
  state: 'verified' | 'unverified' | 'failed'
  reasons: string[]
}

/** Go が出したイベント以外を成功の証拠にせず、壊れた出力も結果へ残す。 */
export function parseGoEvents(output: string): { events: GoTestEvent[]; failures: string[] } {
  const events: GoTestEvent[] = []
  const failures: string[] = []
  for (const line of output.split('\n').filter(Boolean)) {
    try {
      const event = JSON.parse(line) as GoTestEvent
      if (
        typeof event.Action !== 'string' ||
        typeof event.Package !== 'string' ||
        (event.Test !== undefined && typeof event.Test !== 'string') ||
        (event.Output !== undefined && typeof event.Output !== 'string')
      )
        throw new Error('invalid event shape')
      events.push(event)
    } catch {
      failures.push(`invalid go test JSON event: ${line.slice(0, 200)}`)
    }
  }
  return { events, failures }
}

export function executionResults(events: readonly GoTestEvent[]): TestExecution[] {
  const states = new Map<string, TestExecution>()
  for (const event of events) {
    if (event.Action === 'output') continue
    const name = event.Test ?? ''
    const state: ExecutionState =
      event.Action === 'pass'
        ? 'passed'
        : event.Action === 'fail'
          ? 'failed'
          : event.Action === 'skip'
            ? 'skipped'
            : 'not-run'
    states.set(`${event.Package}\0${name}`, { package: event.Package, name, state })
  }
  const all = [...states.values()]
  const packages = new Map<string, TestExecution[]>()
  for (const test of all) {
    const members = packages.get(test.package) ?? []
    members.push(test)
    packages.set(test.package, members)
  }
  return all.map((test) => {
    const members = packages.get(test.package)!
    const enclosing = states.get(`${test.package}\0`)
    const related = members.filter(
      (entry) => entry.name === test.name || entry.name.startsWith(`${test.name}/`),
    )
    const state =
      enclosing?.state === 'failed' || related.some((entry) => entry.state === 'failed')
        ? 'failed'
        : related.some((entry) => entry.state === 'skipped')
          ? 'skipped'
          : enclosing?.state !== 'passed' || related.some((entry) => entry.state === 'not-run')
            ? 'not-run'
            : test.state
    return { ...test, state }
  })
}

export function referenceResults(
  references: readonly TestReference[],
  tests: readonly TestExecution[],
): ReferenceResult[] {
  return references.map((reference) => ({
    ...reference,
    state:
      tests.find((test) => test.package === reference.package && test.name === reference.name)
        ?.state ?? 'not-run',
  }))
}

export function evaluateVerification(
  operations: readonly OperationVerification[],
  changed: ReadonlySet<string>,
  references: readonly ReferenceResult[],
  databases: readonly DBVerification[],
  requireDB: boolean,
): string[] {
  const failures: string[] = []
  for (const operation of operations) {
    if (operation.state === 'failed')
      failures.push(`${operation.operationId}: static contract verification failed`)
    else if (operation.state === 'unverified' && changed.has(operation.operationId)) {
      const evidence = references.filter((reference) =>
        reference.operationIds.includes(operation.operationId),
      )
      if (!evidence.some((reference) => reference.state === 'passed')) {
        failures.push(
          `${operation.operationId}: no successful contract test for changed unverified operation`,
        )
      } else {
        const incomplete = evidence.filter((reference) => reference.state !== 'passed')
        if (incomplete.length > 0) {
          failures.push(
            `${operation.operationId}: incomplete contract test evidence (${incomplete.map((reference) => `${reference.name}: ${reference.state}`).join(', ')})`,
          )
        }
      }
    }
  }
  if (requireDB) {
    for (const db of databases) {
      if (db.state !== 'verified')
        failures.push(
          `${db.package}: required DB verification is ${db.state} (${db.reasons.join(', ')})`,
        )
    }
  }
  return [...new Set(failures)]
}

export function changedOperations(
  files: readonly string[],
  owners: readonly { operationId: string; module?: string }[],
  dependencies: ReadonlyMap<string, ReadonlySet<string>> = new Map(),
): Set<string> {
  const production = files.filter(
    (file) =>
      file.startsWith('backend/') &&
      file.endsWith('.go') &&
      !file.endsWith('_test.go') &&
      !file.includes('/testing_') &&
      !file.includes('/testdata/'),
  )
  const global =
    files.some(
      (file) =>
        file.endsWith('.tsp') ||
        file === 'go.mod' ||
        file === 'go.sum' ||
        file === 'infra/schema/postgres.sql',
    ) || production.some((file) => ['shared', 'cmd'].includes(file.split('/')[1]!))
  const modules = new Set(production.map((file) => file.split('/')[1]!))
  if (!global && modules.size === 0) return new Set()
  let expanded = true
  while (expanded) {
    expanded = false
    for (const [module, imports] of dependencies) {
      if (!modules.has(module) && [...imports].some((dependency) => modules.has(dependency))) {
        modules.add(module)
        expanded = true
      }
    }
  }
  return new Set(
    owners
      .filter((owner) => global || owner.module === undefined || modules.has(owner.module))
      .map((owner) => owner.operationId),
  )
}

/** Go のリポジトリ内 import から、影響を受けるモジュールの候補を保守的に導く。 */
export function moduleDependencies(
  files: readonly { path: string; source: string }[],
  modulePath: string,
): Map<string, Set<string>> {
  const dependencies = new Map<string, Set<string>>()
  for (const file of files) {
    if (file.path.endsWith('_test.go')) continue
    const module = file.path.split('/')[1]!
    const imports = dependencies.get(module) ?? new Set<string>()
    for (const occurrence of file.source.split(`"${modulePath}/backend/`).slice(1)) {
      const target = occurrence.slice(0, occurrence.indexOf('"')).split('/')[0]
      if (target && target !== module) imports.add(target)
    }
    dependencies.set(module, imports)
  }
  return dependencies
}

/** 収集と診断で同じ宣言を解釈し、本文中のコメントを契約証拠にしない。 */
export function collectOperationReferences(
  source: string,
  operationIds: readonly string[],
): {
  references: { name: string; operationId: string }[]
  findings: { line: number; message: string }[]
} {
  const references: { name: string; operationId: string }[] = []
  const findings: { line: number; message: string }[] = []
  const known = new Set(operationIds)
  const lines = source.split('\n')
  const prefix = '//verify:operation'
  for (const [index, line] of lines.entries()) {
    const comment = line.trimStart()
    if (!comment.startsWith(prefix)) continue
    const report = (message: string) => findings.push({ line: index + 1, message })
    if (!comment.startsWith(`${prefix} `)) {
      report('expected space after //verify:operation')
      continue
    }
    const directive = comment.slice(prefix.length + 1)
    const colon = directive.indexOf(':')
    if (colon < 0) {
      report('missing colon after operation ID')
      continue
    }
    if (!directive.slice(colon + 1).trim()) {
      report('missing contract observation after colon')
      continue
    }
    const operationId = directive.slice(0, colon).trim()
    if (!known.has(operationId)) {
      report(`unknown operation ID: ${operationId || '(empty)'}`)
      continue
    }
    let next = index + 1
    while (lines[next]?.trimStart().startsWith('//')) next += 1
    // 本体を渡さず直上コメントと宣言だけを既存の関数判定へ渡す。
    const [name] = specCitingTests(`//spec:covers ${directive}\n${lines[next] ?? ''}`, [
      operationId,
    ])
    if (!name) {
      report('directive is not directly above a Go test')
      continue
    }
    references.push({ name, operationId })
  }
  return { references, findings }
}

/** ソース上の参照と実行結果は、同じ package とトップレベルのテスト名だけで結ぶ。 */
export function collectReferences(
  modulePath: string,
  files: readonly { path: string; source: string }[],
  declaredIds: readonly string[],
  operationIds: readonly string[],
): TestReference[] {
  const references = new Map<string, TestReference>()
  for (const file of files) {
    if (!file.path.endsWith('_test.go')) continue
    const packageName = `${modulePath}/${file.path.slice(0, file.path.lastIndexOf('/'))}`
    const add = (name: string, id: string, field: 'normativeIds' | 'operationIds') => {
      const key = `${packageName}\0${name}`
      let reference = references.get(key)
      if (!reference) {
        reference = {
          package: packageName,
          name,
          path: file.path,
          normativeIds: [],
          operationIds: [],
        }
        references.set(key, reference)
      }
      if (!reference[field].includes(id)) reference[field].push(id)
    }
    for (const id of declaredIds.filter((id) => file.source.includes(id))) {
      for (const name of specCitingTests(file.source, [id])) add(name, id, 'normativeIds')
    }
    for (const { name, operationId } of collectOperationReferences(file.source, operationIds)
      .references) {
      add(name, operationId, 'operationIds')
    }
  }
  return [...references.values()]
}

export function databaseResults(
  packages: readonly string[],
  events: readonly GoTestEvent[],
  tests: readonly TestExecution[],
): DBVerification[] {
  return packages.map((packageName) => {
    const members = tests.filter((test) => test.package === packageName && test.name !== '')
    const packageResult = tests.find((test) => test.package === packageName && test.name === '')
    const reasons: string[] = []
    if (
      events.some(
        (event) => event.Package === packageName && event.Output?.includes('skipping DB tests'),
      )
    )
      reasons.push('DB unavailable')
    if (members.length === 0) reasons.push('no DB tests executed')
    if (members.some((test) => test.state === 'skipped')) reasons.push('test skipped')
    if (
      members.some((test) => test.state === 'not-run') ||
      !packageResult ||
      packageResult.state === 'not-run'
    )
      reasons.push('execution incomplete')
    const failed =
      packageResult?.state === 'failed' || members.some((test) => test.state === 'failed')
    if (failed) reasons.push('test execution failed')
    return {
      package: packageName,
      state: failed ? 'failed' : reasons.length ? 'unverified' : 'verified',
      reasons,
    }
  })
}
