import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { repositoryGoFiles } from './repository-inputs.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'
import { diffStatusCodes, type OpenAPIDocument } from './status-drift.ts'

export async function checkStatusDrift(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
): Promise<CheckOutcome> {
  const document = JSON.parse(
    await snapshot.read(await snapshot.generatedOpenApi()),
  ) as OpenAPIDocument
  const { findings, unresolved, unread } = diffStatusCodes(
    document,
    await repositoryGoFiles(snapshot),
  )
  const lines = findings.map((finding) => `fail  ${finding.message}`)
  if (options.listUnresolved) {
    lines.push(
      ...unresolved.map(
        (entry) => `unresolved  ${entry.operationId} (${entry.reason}: ${entry.detail})`,
      ),
      ...unread.map(
        (entry) => `partial     ${entry.operationId} (not read: ${entry.writers.join(', ')})`,
      ),
    )
  }
  if (findings.length > 0) return { ok: false, lines }

  const operations = Object.values(document.paths ?? {}).flatMap((pathItem) =>
    Object.values(pathItem).filter((operation) => operation.operationId),
  ).length
  const byReason = new Map<string, number>()
  for (const entry of unresolved) byReason.set(entry.reason, (byReason.get(entry.reason) ?? 0) + 1)
  const breakdown = [...byReason.entries()]
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([reason, count]) => `${reason}=${count}`)
    .join(', ')
  const fully = operations - unresolved.length - unread.length
  lines.push(
    `ok  declared status codes (0 finding(s); ${fully}/${operations} operation(s) read in full, ` +
      `${unread.length} read in part, ${unresolved.length} not reached${breakdown ? `: ${breakdown}` : ''}. ` +
      '--list-unresolved names them)',
  )
  return { ok: true, lines }
}
