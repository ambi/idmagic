import { mkdir } from 'node:fs/promises'
import { dirname } from 'node:path'
import {
  collectGoHandlers,
  collectRoutes,
  type GoFile,
  type OpenAPIDocument,
} from '../../check/src/contract-drift.ts'
import type { OperationVerification } from '../../check/src/operation-verification.ts'
import { checkContractDrift } from '../../check/src/check-contract-drift.ts'
import { checkStatusDrift } from '../../check/src/check-status-drift.ts'
import { repositoryGoFiles } from '../../check/src/repository-inputs.ts'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'

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

if (import.meta.main) {
  const snapshot = createWorkspaceSnapshot(process.cwd())
  const document = JSON.parse(
    await snapshot.read(await snapshot.generatedOpenApi()),
  ) as OpenAPIDocument
  const results = await Promise.all([
    checkContractDrift(snapshot, { verbose: false, listUnresolved: true }),
    checkStatusDrift(snapshot, { verbose: false, listUnresolved: true }),
  ])
  const inventory = contractEvidenceInventory(
    document,
    await repositoryGoFiles(snapshot, true),
    results.flatMap((result) => result.operations ?? []),
  )
  const output = snapshot.path('.verification/contract-evidence-inventory.json')
  await mkdir(dirname(output), { recursive: true })
  await Bun.write(output, `${JSON.stringify(inventory, null, 2)}\n`)
  console.log(`未解析の操作: ${inventory.length} 件。候補は観測や成功の証拠ではありません。`)
  console.log(`棚卸し結果: ${output}`)
}
