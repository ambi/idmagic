import { dirname } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import {
  type ConcurrencyFinding,
  concurrencyFindings,
  isConcurrencyGoTarget,
  isConcurrencySqlTarget,
  mergeSqlQueries,
  parseSqlQueries,
  type SqlQueries,
} from './sql-concurrency.ts'
import type { CheckOutcome } from './runner.ts'

function message(finding: ConcurrencyFinding): string {
  switch (finding.problem) {
    case 'row-lock-gap':
      return `${finding.query} locks nothing when no row exists, and the function then inserts into ${finding.table}; insert the row first with ON CONFLICT DO NOTHING or take an advisory lock`
    case 'unguarded-write':
      return `${finding.query} writes ${finding.table} unconditionally after an unlocked read of it; lock first, or check the read value in WHERE`
    case 'lock-outside-tx':
      return `${finding.query} takes FOR UPDATE outside a transaction, so the lock ends with the statement`
    case 'missing-reason':
      return '//sql:concurrency-ok needs a reason'
  }
}

export async function checkSqlConcurrency(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const files = await snapshot.files('backend', ['node_modules', 'vendor'])
  const byDirectory = new Map<string, SqlQueries>()
  for (const path of files.filter(isConcurrencySqlTarget)) {
    const directory = dirname(path)
    const queries = byDirectory.get(directory) ?? { byName: new Map(), advisoryLocks: new Set() }
    mergeSqlQueries(queries, parseSqlQueries(await snapshot.read(path)))
    byDirectory.set(directory, queries)
  }
  const lines: string[] = []
  let checked = 0
  for (const path of files.filter(isConcurrencyGoTarget)) {
    const queries = byDirectory.get(dirname(path))
    if (!queries) continue
    checked++
    for (const finding of concurrencyFindings(queries, await snapshot.read(path))) {
      lines.push(`fail  ${path}:${finding.line}: ${message(finding)}`)
    }
  }
  return {
    ok: lines.length === 0,
    lines: lines.length > 0 ? lines : [`ok  ${checked} Go file(s) with sqlc queries`],
  }
}
