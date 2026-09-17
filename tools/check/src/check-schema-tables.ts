import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { CheckOutcome } from './runner.ts'
import { compareTables, declaredTables, describedTables } from './schema-tables.ts'

const SCHEMA = 'infra/schema/postgres.sql'
const DESCRIPTION = 'docs/design/data/database.md'

export async function checkSchemaTables(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const declared = declaredTables(await snapshot.read(SCHEMA))
  const findings = compareTables(declared, describedTables(await snapshot.read(DESCRIPTION)))
  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map((finding) => `fail  ${DESCRIPTION}: ${finding.message}`)
        : [`ok  ${declared.length} table(s) in ${SCHEMA} described in ${DESCRIPTION}`],
  }
}
