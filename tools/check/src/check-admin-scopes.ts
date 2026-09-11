import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import {
  collectAdminOperations,
  type OpenApiDocument,
  parseApiTokenScopes,
  verifyAdminScopes,
} from './admin-scopes.ts'
import type { CheckOutcome } from './runner.ts'

export async function checkAdminScopes(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const document = JSON.parse(
    await snapshot.read(await snapshot.generatedOpenApi()),
  ) as OpenApiDocument
  const vocabulary = parseApiTokenScopes(await snapshot.read('spec/contexts/api-tokens/models.tsp'))
  const operations = collectAdminOperations(document)
  const findings = verifyAdminScopes(operations, vocabulary)
  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map((finding) => `admin API scope declaration: ${finding}`)
        : [`ok  admin API scope declarations (${operations.length} operation(s))`],
  }
}
