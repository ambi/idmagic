import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { addedBoundaryDebtViolations, parseBoundaryDebt } from './boundary-fitness.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'

const BOUNDARY_DEBT_PATH = 'tools/check/boundary-debt.json'

export type RevisionReader = {
  read(ref: string, path: string): string | undefined
}

export function gitRevisionReader(root: string): RevisionReader {
  return {
    read(ref, path) {
      const revision = Bun.spawnSync(['git', 'rev-parse', '--verify', `${ref}^{commit}`], {
        cwd: root,
      })
      if (revision.exitCode !== 0) {
        throw new Error(`cannot resolve Git revision ${ref}: ${revision.stderr.toString().trim()}`)
      }
      const result = Bun.spawnSync(['git', 'show', `${ref}:${path}`], { cwd: root })
      return result.exitCode === 0 ? result.stdout.toString() : undefined
    },
  }
}

export async function checkBoundaryDebtRatchet(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
  reader: RevisionReader = gitRevisionReader(snapshot.root),
): Promise<CheckOutcome> {
  if (!options.baseRevision) {
    return { ok: false, lines: ['boundary-debt-ratchet requires --base-revision <git-revision>'] }
  }
  const current = parseBoundaryDebt(await snapshot.read(BOUNDARY_DEBT_PATH))
  const beforeSource = reader.read(options.baseRevision, BOUNDARY_DEBT_PATH)
  if (beforeSource === undefined) {
    return {
      ok: true,
      lines: [`ok  boundary debt ratchet adopted against ${options.baseRevision}`],
    }
  }
  const added = addedBoundaryDebtViolations(parseBoundaryDebt(beforeSource), current)
  if (added.length > 0) {
    return {
      ok: false,
      lines: [
        `fail  ${BOUNDARY_DEBT_PATH}: ${added.join(', ')} absent from ${options.baseRevision}; ` +
          'remove the new dependency instead of growing boundary debt.',
      ],
    }
  }
  return {
    ok: true,
    lines: [`ok  boundary debt ratchet against ${options.baseRevision}`],
  }
}
