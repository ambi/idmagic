import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'

export type CheckOutcome = {
  ok: boolean
  lines: string[]
}

export type CheckOptions = {
  verbose: boolean
  listUnresolved: boolean
  baseRevision?: string
}

export type RepositoryCheck = {
  name: string
  groups: string[]
  run(snapshot: WorkspaceSnapshot, options: CheckOptions): Promise<CheckOutcome>
}

export type CheckResult = CheckOutcome & { name: string }

const DEFAULT_OPTIONS: CheckOptions = { verbose: false, listUnresolved: false }

/** 独立した検査を並行実行し、途中で打ち切らずに全結果を保持する。 */
export async function runChecks(
  checks: readonly RepositoryCheck[],
  snapshot: WorkspaceSnapshot,
  options: CheckOptions = DEFAULT_OPTIONS,
): Promise<CheckResult[]> {
  return Promise.all(
    checks.map(async (check): Promise<CheckResult> => {
      try {
        return { name: check.name, ...(await check.run(snapshot, options)) }
      } catch (error) {
        const message = error instanceof Error ? error.message : String(error)
        return { name: check.name, ok: false, lines: [`${check.name}: ${message}`] }
      }
    }),
  )
}

if (import.meta.main) {
  const args = process.argv.slice(2)
  const baseRevisionIndex = args.indexOf('--base-revision')
  const baseRevision = baseRevisionIndex === -1 ? undefined : args[baseRevisionIndex + 1]
  const options: CheckOptions = {
    verbose: args.includes('--verbose'),
    listUnresolved: args.includes('--list-unresolved'),
    baseRevision,
  }
  const selectors = args.filter(
    (arg, index) =>
      !arg.startsWith('--') && (baseRevisionIndex === -1 || index !== baseRevisionIndex + 1),
  )
  const [{ repositoryChecks, selectChecks }, { createWorkspaceSnapshot }] = await Promise.all([
    import('./registry.ts'),
    import('../../workspace/src/workspace.ts'),
  ])
  const checks = selectChecks(selectors)
  const results = await runChecks(checks, createWorkspaceSnapshot(), options)
  for (const result of results) {
    for (const line of result.lines) {
      if (result.ok) console.log(line)
      else console.error(line)
    }
  }
  if (selectors.includes('all') && checks.length !== repositoryChecks.length) {
    console.error('check registry is incomplete')
    process.exit(1)
  }
  if (results.some((result) => !result.ok)) process.exit(1)
}
