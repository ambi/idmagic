import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { OperationVerification } from './operation-verification.ts'
import { mkdir } from 'node:fs/promises'
import { verificationInput } from '../../verification/src/input.ts'

export type CheckOutcome = {
  ok: boolean
  lines: string[]
  operations?: OperationVerification[]
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

/**
 * 比較基準を渡されなかったときの既定値。上流ブランチがあればそれを、なければ main を返す。
 * main の上で直接コミットしても、まだ push していないコミットが検査の範囲に入る。
 */
export function defaultBaseRevision(root: string): string {
  const result = Bun.spawnSync(
    ['git', 'rev-parse', '--abbrev-ref', '--symbolic-full-name', '@{upstream}'],
    { cwd: root, stderr: 'ignore' },
  )
  const upstream = result.exitCode === 0 ? result.stdout.toString().trim() : ''
  return upstream || 'main'
}

if (import.meta.main) {
  const args = process.argv.slice(2)
  const baseRevisionIndex = args.indexOf('--base-revision')
  const [{ selectChecks }, { createWorkspaceSnapshot }] = await Promise.all([
    import('./registry.ts'),
    import('../../workspace/src/workspace.ts'),
  ])
  const snapshot = createWorkspaceSnapshot()
  const options: CheckOptions = {
    verbose: args.includes('--verbose'),
    listUnresolved: args.includes('--list-unresolved'),
    baseRevision:
      baseRevisionIndex === -1 ? defaultBaseRevision(snapshot.root) : args[baseRevisionIndex + 1],
  }
  const selectors = args
    .slice(0, baseRevisionIndex === -1 ? undefined : baseRevisionIndex)
    .filter((arg) => !arg.startsWith('--'))
  const checks = selectChecks(selectors)
  const versioned =
    Bun.spawnSync(['git', 'rev-parse', '--verify', 'HEAD'], {
      cwd: snapshot.root,
      stderr: 'ignore',
    }).exitCode === 0
  const input = versioned
    ? await verificationInput(snapshot, options.baseRevision)
    : { identity: 'unverified: no Git revision' }
  const results = await runChecks(checks, snapshot, options)
  await mkdir(snapshot.path('.verification'), { recursive: true })
  const reportName = (selectors.join('-') || 'all').replace(/[^a-zA-Z0-9-]/g, '-').slice(0, 100)
  const reportPath = snapshot.path(`.verification/check-${reportName}.json`)
  await Bun.write(
    reportPath,
    `${JSON.stringify(
      {
        schemaVersion: 1,
        input,
        command: process.argv.slice(1),
        checks: results.map((result) => ({
          ...result,
          state: !result.ok
            ? 'failed'
            : result.operations?.some((entry) => entry.state === 'unverified')
              ? 'unverified'
              : 'verified',
        })),
      },
      null,
      2,
    )}\n`,
  )
  for (const result of results) {
    for (const line of result.lines) {
      if (result.ok) console.log(line)
      else console.error(line)
    }
  }
  console.log(`static verification report: ${reportPath}`)
  if (results.some((result) => !result.ok)) process.exit(1)
}
