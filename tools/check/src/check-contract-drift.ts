import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { collectRoutes, diffContract, type OpenAPIDocument } from './contract-drift.ts'
import { repositoryGoFiles } from './repository-inputs.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'

export async function checkContractDrift(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
): Promise<CheckOutcome> {
  const goFiles = await repositoryGoFiles(snapshot)
  const document = JSON.parse(
    await snapshot.read(await snapshot.generatedOpenApi()),
  ) as OpenAPIDocument
  const { findings, unresolved } = diffContract(document, collectRoutes(goFiles), goFiles)
  const lines = findings.map((finding) => `fail  ${finding.message}`)
  if (options.listUnresolved) {
    lines.push(
      ...unresolved.map(
        (entry) => `unresolved  ${entry.operationId} (${entry.reason}: ${entry.detail})`,
      ),
    )
  }
  if (findings.length > 0) return { ok: false, lines }

  const byReason = new Map<string, number>()
  for (const entry of unresolved) byReason.set(entry.reason, (byReason.get(entry.reason) ?? 0) + 1)
  const breakdown = [...byReason.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([reason, count]) => `${reason}=${count}`)
    .join(', ')
  const operations = Object.values(document.paths ?? {}).flatMap((pathItem) =>
    Object.values(pathItem).filter((operation) => operation.operationId),
  ).length
  const compared = operations - unresolved.length
  lines.push(
    `ok  contract body drift (0 finding(s); compared ${compared}/${operations} operation(s), ` +
      `${unresolved.length} not followed: ${breakdown || 'none'}. --list-unresolved names them)`,
  )
  return { ok: true, lines }
}
