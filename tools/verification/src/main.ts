import { mkdir } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { repositoryGoFiles } from '../../check/src/repository-inputs.ts'
import { readWorkingTree } from '../../check/src/spec-diff.ts'
import { validateDocument } from '../../check/src/specification-doc.ts'
import {
  collectGoHandlers,
  collectRoutes,
  type OpenAPIDocument,
} from '../../check/src/contract-drift.ts'
import { operationIds } from '../../check/src/operation-verification.ts'
import { checkContractDrift } from '../../check/src/check-contract-drift.ts'
import { checkStatusDrift } from '../../check/src/check-status-drift.ts'
import { defaultBaseRevision, runChecks } from '../../check/src/runner.ts'
import {
  changedOperations,
  moduleDependencies,
  collectReferences,
  databaseResults,
  evaluateVerification,
  executionResults,
  referenceResults,
  parseGoEvents,
} from './evidence.ts'
import { changedFiles, verificationInput } from './input.ts'

const snapshot = createWorkspaceSnapshot(process.cwd())
const args = process.argv.slice(2)
const separator = args.indexOf('--')
if (separator < 0) throw new Error('usage: main.ts [--full] [--report path] -- <go test arguments>')
const options = args.slice(0, separator)
const full = options.includes('--full')
const reportIndex = options.indexOf('--report')
const reportPath = resolve(
  snapshot.root,
  reportIndex < 0 ? '.verification/go-tests.json' : options[reportIndex + 1]!,
)
const baseRevision = process.env.VERIFICATION_BASE_REVISION || defaultBaseRevision(snapshot.root)
// CI は Skip 許可が混入しても未検証を拒否する。手元も明示の許可がない限り必須とする。
const requireDB =
  process.env.VERIFICATION_REQUIRE_DB === '1' || process.env.PGTEST_ALLOW_SKIP !== '1'
const command = ['go', 'test', '-json', '-count=1', ...args.slice(separator + 1)]
const input = await verificationInput(snapshot, full ? baseRevision : undefined)
const files = await repositoryGoFiles(snapshot, true)
const modulePath = (await snapshot.read('go.mod'))
  .split('\n')
  .find((line) => line.startsWith('module '))
  ?.slice(7)
  .trim()
if (!modulePath) throw new Error('go.mod declares no module path')
const declaredIds: string[] = []
for (const [path, source] of await readWorkingTree(snapshot.root)) {
  if (!path.endsWith('.md')) continue
  const doc = validateDocument(path, source)
  declaredIds.push(
    ...[...doc.scenarioIds, ...doc.exampleIds, ...doc.standardIds].map((entry) => entry.id),
  )
}
let ids: string[] = []
let changed = new Set<string>()
const checks = full
  ? await runChecks(
      [
        { name: 'contract-drift', groups: [], run: checkContractDrift },
        { name: 'status-drift', groups: [], run: checkStatusDrift },
      ],
      snapshot,
    )
  : []
if (full) {
  const document = JSON.parse(
    await snapshot.read(await snapshot.generatedOpenApi()),
  ) as OpenAPIDocument
  ids = operationIds(document)
  const routes = collectRoutes(files.filter((file) => !file.path.endsWith('_test.go')))
  const handlers = collectGoHandlers(files.filter((file) => !file.path.endsWith('_test.go')))
  const owners = Object.entries(document.paths ?? {}).flatMap(([path, item]) =>
    Object.entries(item).flatMap(([method, operation]) => {
      if (!operation.operationId || !ids.includes(operation.operationId)) return []
      const handler = handlers.get(routes.get(`${method.toUpperCase()} ${path}`) ?? '')
      return [{ operationId: operation.operationId, module: handler?.path.split('/')[1] }]
    }),
  )
  changed = changedOperations(
    changedFiles(snapshot.root, baseRevision),
    owners,
    moduleDependencies(files, modulePath),
  )
}
const references = collectReferences(modulePath, files, [...new Set(declaredIds)], ids)
const dbPackages = [
  ...new Set(
    files
      .filter(
        (file) =>
          file.path.endsWith('_test.go') &&
          // go test ./... は testdata 配下の子プロセス用 fixture を列挙しない。
          !file.path.split('/').includes('testdata') &&
          file.source.includes('/shared/storage/testing_postgres"'),
      )
      .map((file) => `${modulePath}/${dirname(file.path)}`),
  ),
]

await mkdir(dirname(reportPath), { recursive: true })
const child = Bun.spawn(command, { cwd: snapshot.root, stdout: 'pipe', stderr: 'pipe' })
const [output, stderr, code] = await Promise.all([
  new Response(child.stdout).text(),
  new Response(child.stderr).text(),
  child.exited,
])
if (stderr) process.stderr.write(stderr)
const eventLog = `${reportPath}.jsonl`
await Bun.write(eventLog, output)
const { events, failures } = parseGoEvents(output)
if (code !== 0) failures.push(`go test exited ${code}`)
if (events.length === 0) failures.push('go test produced no execution events')
const tests = executionResults(events)
if (tests.some((test) => test.state === 'failed'))
  failures.push('go test reported failed execution')
const facts = referenceResults(references, tests)
const databases = databaseResults(
  full ? dbPackages : dbPackages.filter((pkg) => events.some((event) => event.Package === pkg)),
  events,
  tests,
)
const operations = checks.flatMap((check) => check.operations ?? [])
for (const check of checks) {
  for (const line of check.lines) console.log(line)
  if (!check.ok) failures.push(`${check.name}: static verification failed`)
}
// 代替証拠はこの実行の facts だけから選び、以前の結果ファイルを読まない。
failures.push(...evaluateVerification(operations, changed, facts, databases, requireDB))
const after = await verificationInput(snapshot, full ? baseRevision : undefined)
if (after.fingerprint !== input.fingerprint)
  failures.push('verification inputs changed during execution; rerun on a stable tree')
const result = {
  schemaVersion: 1,
  input,
  command,
  requireDB,
  scope: full ? 'full' : 'selected',
  eventLog,
  checks,
  changedOperations: [...changed],
  tests,
  references: facts,
  databases,
  changeResistance: {
    state: 'not-collected',
    task: 'test-go-mutation',
    reason: 'test success does not measure detection ability',
  },
  failures: [...new Set(failures)],
}
await Bun.write(reportPath, `${JSON.stringify(result, null, 2)}\n`)
for (const db of databases)
  console.log(
    `DB ${db.state} ${db.package}${db.reasons.length ? ` (${db.reasons.join(', ')})` : ''}`,
  )
for (const state of ['passed', 'failed', 'skipped', 'not-run'])
  console.log(`tests ${state}: ${tests.filter((test) => test.name && test.state === state).length}`)
console.log(
  `normative references: ${facts.filter((fact) => fact.normativeIds.length).length} test(s); execution and change-resistance are separate facts`,
)
console.log(`verification report: ${reportPath}`)
if (failures.length) {
  for (const failure of result.failures) console.error(`fail ${failure}`)
  for (const event of events) {
    if (
      event.Output &&
      tests.some((test) => test.package === event.Package && test.state === 'failed')
    )
      process.stderr.write(event.Output)
  }
  process.exit(1)
}
