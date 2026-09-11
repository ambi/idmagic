import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { DebtEntry } from './normative-coverage.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'

/**
 * 台帳は 1 つだけである。標準の側の台帳は wi-495 が空にして消したので、ratchet が
 * 守る対象も消えた。宣言した標準の行は、その id を名指すテストを持つか検査に落ちる
 * かのどちらかであり、流入を測る基準そのものが要らない。
 */
export const COVERAGE_DEBT_PATHS = ['tools/check/example-coverage-debt.json'] as const

type DebtFile = { untested: DebtEntry[] }

export type RevisionReader = {
  read(ref: string, path: string): string
}

/** Return ledger IDs that the current revision adds beyond the named base revision. */
export function addedDebtIds(
  before: readonly DebtEntry[],
  current: readonly DebtEntry[],
): string[] {
  const beforeIds = new Set(before.map((entry) => entry.id))
  return current
    .map((entry) => entry.id)
    .filter((id) => !beforeIds.has(id))
    .sort()
}

function readDebt(source: string, path: string): DebtEntry[] {
  const parsed = JSON.parse(source) as DebtFile
  if (!Array.isArray(parsed.untested)) throw new Error(`${path}: untested must be an array`)
  return parsed.untested.map((entry) => {
    if (typeof entry?.id !== 'string' || typeof entry?.reason !== 'string') {
      throw new Error(`${path}: every untested entry needs an id and a reason`)
    }
    return entry
  })
}

export function gitRevisionReader(root: string): RevisionReader {
  return {
    read(ref, path) {
      const result = Bun.spawnSync(['git', 'show', `${ref}:${path}`], { cwd: root })
      if (result.exitCode !== 0) {
        throw new Error(
          `cannot read ${path} from Git revision ${ref}: ${result.stderr.toString().trim()}`,
        )
      }
      return result.stdout.toString()
    },
  }
}

export async function checkCoverageDebtRatchet(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
  reader: RevisionReader = gitRevisionReader(snapshot.root),
): Promise<CheckOutcome> {
  if (!options.baseRevision) {
    return { ok: false, lines: ['coverage-debt-ratchet requires --base-revision <git-revision>'] }
  }
  const lines: string[] = []
  for (const path of COVERAGE_DEBT_PATHS) {
    try {
      const added = addedDebtIds(
        readDebt(reader.read(options.baseRevision, path), path),
        readDebt(await snapshot.read(path), path),
      )
      if (added.length > 0) {
        lines.push(
          `fail  ${path}: ${added.join(', ')} ${added.length === 1 ? 'is' : 'are'} absent from ${options.baseRevision}; add a test instead of growing the debt list.`,
        )
      }
    } catch (error) {
      lines.push(`fail  ${error instanceof Error ? error.message : String(error)}`)
    }
  }
  if (lines.length > 0) return { ok: false, lines }
  return { ok: true, lines: [`ok  coverage debt ratchet against ${options.baseRevision}`] }
}
