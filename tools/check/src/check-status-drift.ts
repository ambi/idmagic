import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { repositoryGoFiles } from './repository-inputs.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'
import { diffStatusCodes, type OpenAPIDocument } from './status-drift.ts'
import {
  operationIds,
  operationVerification,
  verificationSummary,
} from './operation-verification.ts'

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
  const operations = operationVerification(operationIds(document), findings, [
    ...unresolved.map((entry) => ({
      operationId: entry.operationId,
      reason: `${entry.reason}: ${entry.detail}`,
    })),
    ...unread.map((entry) => ({
      operationId: entry.operationId,
      reason: `unread writers: ${entry.writers.join(', ')}`,
    })),
  ])
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
  lines.push(verificationSummary('declared status codes', operations))
  return { ok: findings.length === 0, lines, operations }
}
