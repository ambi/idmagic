import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { collectRoutes, diffContract, type OpenAPIDocument } from './contract-drift.ts'
import { repositoryGoFiles } from './repository-inputs.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'
import {
  operationIds,
  operationVerification,
  verificationSummary,
} from './operation-verification.ts'

export async function checkContractDrift(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
): Promise<CheckOutcome> {
  const goFiles = await repositoryGoFiles(snapshot)
  const document = JSON.parse(
    await snapshot.read(await snapshot.generatedOpenApi()),
  ) as OpenAPIDocument
  const { findings, unresolved } = diffContract(document, collectRoutes(goFiles), goFiles)
  const operations = operationVerification(
    operationIds(document),
    findings,
    unresolved.map((entry) => ({
      operationId: entry.operationId,
      reason: `${entry.reason}: ${entry.detail}`,
    })),
  )
  const lines = findings.map((finding) => `fail  ${finding.message}`)
  if (options.listUnresolved) {
    lines.push(
      ...unresolved.map(
        (entry) => `unresolved  ${entry.operationId} (${entry.reason}: ${entry.detail})`,
      ),
    )
  }
  lines.push(verificationSummary('contract body drift', operations))
  return { ok: findings.length === 0, lines, operations }
}
