import { collectOperationReferences } from '../../verification/src/evidence.ts'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { OpenAPIDocument } from './contract-drift.ts'
import { operationIds } from './operation-verification.ts'
import { repositoryGoFiles } from './repository-inputs.ts'
import type { CheckOutcome } from './runner.ts'

export async function checkOperationReferences(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const document = JSON.parse(
    await snapshot.read(await snapshot.generatedOpenApi()),
  ) as OpenAPIDocument
  const ids = operationIds(document)
  const lines: string[] = []
  let accepted = 0
  for (const file of await repositoryGoFiles(snapshot, true)) {
    if (!file.path.endsWith('_test.go')) continue
    const result = collectOperationReferences(file.source, ids)
    accepted += result.references.length
    lines.push(
      ...result.findings.map((finding) => `fail  ${file.path}:${finding.line}: ${finding.message}`),
    )
  }
  const ok = lines.length === 0
  lines.push(`operation references: ${accepted} accepted; test execution is separate evidence`)
  return { ok, lines }
}
