import { mkdir } from 'node:fs/promises'
import { dirname } from 'node:path'
import { parseArgs } from 'node:util'
import {
  collectGoHandlers,
  collectRoutes,
  type GoFile,
  type OpenAPIDocument,
} from '../../check/src/contract-drift.ts'
import { operationIds, type OperationVerification } from '../../check/src/operation-verification.ts'
import { checkContractDrift } from '../../check/src/check-contract-drift.ts'
import { checkStatusDrift } from '../../check/src/check-status-drift.ts'
import { repositoryGoFiles } from '../../check/src/repository-inputs.ts'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { defaultBaseRevision } from '../../check/src/runner.ts'
import { changedOperations, collectReferences, moduleDependencies } from './evidence.ts'
import { changedFiles } from './input.ts'

/** 一致は読む候補であり、契約の観測やテストの成功を示さない。 */
export function contractEvidenceInventory(
  document: OpenAPIDocument,
  files: GoFile[],
  verification: OperationVerification[],
) {
  const production = files.filter((file) => !file.path.endsWith('_test.go'))
  const routes = collectRoutes(production)
  const handlers = collectGoHandlers(production)
  const tests = files.filter((file) => file.path.endsWith('_test.go'))
  const shape = (key: string) => key.replace(/\{[^}]*\}/g, '{}')
  return Object.entries(document.paths ?? {}).flatMap(([path, item]) =>
    Object.entries(item).flatMap(([method, operation]) => {
      const operationId = operation.operationId
      const unresolved = verification.filter(
        (entry) => entry.operationId === operationId && entry.state === 'unverified',
      )
      if (!operationId || unresolved.length === 0) return []
      const key = `${method.toUpperCase()} ${path}`
      const matching = [...routes].filter(([route]) => shape(route) === shape(key))
      const name = routes.get(key) ?? (matching.length === 1 ? matching[0]?.[1] : undefined)
      const handler = name ? handlers.get(name) : undefined
      const prefix = path.slice(0, path.indexOf('{') < 0 ? path.length : path.indexOf('{'))
      const direct = tests.filter((file) => prefix.length > 1 && file.source.includes(prefix))
      const candidates = direct.length
        ? direct
        : tests.filter((file) => handler && dirname(file.path) === dirname(handler.path))
      return [
        {
          operationId,
          method: method.toUpperCase(),
          path,
          reasons: [...new Set(unresolved.flatMap((entry) => entry.reasons))],
          handler: name && handler ? { name, path: handler.path } : undefined,
          candidates: candidates.map((file) => file.path),
        },
      ]
    }),
  )
}

export function contractEvidencePreflight(
  document: OpenAPIDocument,
  files: GoFile[],
  verification: OperationVerification[],
  modulePath: string,
  changedPaths: readonly string[],
): {
  operations: (ReturnType<typeof contractEvidenceInventory>[number] & {
    references: { path: string; name: string }[]
  })[]
  missingReferences: string[]
  execution: 'not-collected'
} {
  const inventory = contractEvidenceInventory(document, files, verification)
  const routes = collectRoutes(files.filter((file) => !file.path.endsWith('_test.go')))
  const affected = changedOperations(
    changedPaths,
    inventory.map((entry) => ({
      operationId: entry.operationId,
      // 棚卸しの正規化による担当の推定で、全体検証の未解決経路を選択から落とさない。
      module: routes.has(`${entry.method} ${entry.path}`)
        ? entry.handler?.path.split('/')[1]
        : undefined,
    })),
    moduleDependencies(files, modulePath),
  )
  const references = collectReferences(modulePath, files, [], operationIds(document))
  const operations = inventory
    .filter((entry) => affected.has(entry.operationId))
    .map((entry) => ({
      ...entry,
      references: references
        .filter((reference) => reference.operationIds.includes(entry.operationId))
        .map(({ path, name }) => ({ path, name })),
    }))
  return {
    operations,
    missingReferences: operations
      .filter((entry) => entry.references.length === 0)
      .map((entry) => entry.operationId),
    execution: 'not-collected',
  }
}

if (import.meta.main) {
  const { values } = parseArgs({
    args: process.argv.slice(2),
    options: {
      changed: { type: 'boolean', default: false },
      'base-revision': { type: 'string' },
      target: { type: 'string', multiple: true },
    },
    strict: true,
    allowPositionals: false,
  })
  if (!values.changed && (values['base-revision'] || values.target))
    throw new Error('--base-revision and --target require --changed')
  const snapshot = createWorkspaceSnapshot(process.cwd())
  const document = JSON.parse(
    await snapshot.read(await snapshot.generatedOpenApi()),
  ) as OpenAPIDocument
  const results = await Promise.all([
    checkContractDrift(snapshot, { verbose: false, listUnresolved: true }),
    checkStatusDrift(snapshot, { verbose: false, listUnresolved: true }),
  ])
  const files = await repositoryGoFiles(snapshot, true)
  const verification = results.flatMap((result) => result.operations ?? [])
  let report: unknown
  if (values.changed) {
    const baseRevision =
      values['base-revision'] ||
      process.env.VERIFICATION_BASE_REVISION ||
      defaultBaseRevision(snapshot.root)
    const changed = changedFiles(snapshot.root, baseRevision)
    const targets = values.target ?? []
    const modulePath = (await snapshot.read('go.mod'))
      .split('\n')
      .find((line) => line.startsWith('module '))
      ?.slice(7)
      .trim()
    if (!modulePath) throw new Error('go.mod declares no module path')
    const preflight = contractEvidencePreflight(document, files, verification, modulePath, [
      ...new Set([...changed, ...targets]),
    ])
    report = { baseRevision, changedFiles: changed, targets, ...preflight }
    console.log(
      `変更の影響する未解析操作: ${preflight.operations.length} 件。参照なし: ${preflight.missingReferences.length} 件。実行の証拠は収集していません。`,
    )
    for (const operationId of preflight.missingReferences) console.log(`参照なし: ${operationId}`)
  } else {
    const inventory = contractEvidenceInventory(document, files, verification)
    report = inventory
    console.log(`未解析の操作: ${inventory.length} 件。候補は観測や成功の証拠ではありません。`)
  }
  const output = snapshot.path(
    values.changed
      ? '.verification/contract-evidence-preflight.json'
      : '.verification/contract-evidence-inventory.json',
  )
  await mkdir(dirname(output), { recursive: true })
  await Bun.write(output, `${JSON.stringify(report, null, 2)}\n`)
  console.log(`棚卸し結果: ${output}`)
}
