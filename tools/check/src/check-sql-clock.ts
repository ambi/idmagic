import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { CheckOutcome } from './runner.ts'
import { clockReadings } from './sql-clock.ts'
import { sqlcQueryInputs } from './sqlc-inputs.ts'
import { parseSqlFiles } from './table-writes.ts'

/** sqlc のクエリが、記録の値の外でデータベースの時計を読むことを拒否する。 */
export async function checkSqlClock(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const paths = [...new Set((await sqlcQueryInputs(snapshot)).flatMap((input) => input.files))]
  const files = await Promise.all(
    paths.map(async (path) => ({ path, sql: await snapshot.read(path) })),
  )
  const parsed = files.length === 0 ? new Map() : await parseSqlFiles(files)
  const lines: string[] = []
  for (const { path } of files) {
    for (const reading of clockReadings(parsed.get(path) ?? [])) {
      lines.push(
        `fail  ${path}: ${reading.query} reads the database clock with ${reading.expression} outside an assigned value; pass the time as a query parameter`,
      )
    }
  }
  return {
    ok: lines.length === 0,
    lines: lines.length > 0 ? lines : [`ok  ${files.length} sqlc query file(s)`],
  }
}
