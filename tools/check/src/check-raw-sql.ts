import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { isRawSqlTarget, rawSqlCalls } from './raw-sql.ts'
import type { CheckOutcome } from './runner.ts'

const MESSAGES = {
  unmarked:
    'declare the SQL as a sqlc query, or state why sqlc cannot express it with //sql:raw <reason>',
  'missing-reason': '//sql:raw needs a reason',
} as const

export async function checkRawSql(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const files = (await snapshot.files('backend', ['node_modules', 'vendor'])).filter(isRawSqlTarget)
  const lines: string[] = []
  let exceptions = 0
  for (const path of files) {
    const source = await snapshot.read(path)
    exceptions += source.match(/^\s*\/\/sql:raw\b/gm)?.length ?? 0
    for (const call of rawSqlCalls(source)) {
      lines.push(`fail  ${path}:${call.line}: ${MESSAGES[call.problem]}`)
    }
  }
  return {
    ok: lines.length === 0,
    lines:
      lines.length > 0
        ? lines
        : [`ok  ${files.length} Go file(s), ${exceptions} //sql:raw exception(s)`],
  }
}
