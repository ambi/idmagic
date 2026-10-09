import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createWorkspaceSnapshot, type WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { BoundaryAnalysis, BoundaryViolationKind } from './boundary-fitness.ts'
import { analyzeModuleBoundaries, BOUNDARY_INPUT_PATHS } from './check-boundaries.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'

/** 基準 revision の境界検査の入力を、読み取り専用の作業ツリーとして取り出す。 */
export type RevisionCheckout = (
  revision: string,
) => Promise<{ snapshot: WorkspaceSnapshot; release(): Promise<void> }>

export function gitRevisionCheckout(root: string): RevisionCheckout {
  return async (revision) => {
    const resolved = Bun.spawnSync(['git', 'rev-parse', '--verify', `${revision}^{commit}`], {
      cwd: root,
    })
    if (resolved.exitCode !== 0) {
      throw new Error(
        `cannot resolve Git revision ${revision}: ${resolved.stderr.toString().trim()}`,
      )
    }
    const archive = Bun.spawnSync(
      [
        'git',
        'archive',
        '--format=tar',
        resolved.stdout.toString().trim(),
        ...BOUNDARY_INPUT_PATHS,
      ],
      { cwd: root },
    )
    if (archive.exitCode !== 0) {
      throw new Error(
        `cannot read boundary inputs at ${revision}: ${archive.stderr.toString().trim()}`,
      )
    }
    const directory = await mkdtemp(join(tmpdir(), 'boundary-base-'))
    const extract = Bun.spawnSync(['tar', '-x', '-C', directory], { stdin: archive.stdout })
    if (extract.exitCode !== 0) {
      await rm(directory, { recursive: true, force: true })
      throw new Error(
        `cannot extract boundary inputs at ${revision}: ${extract.stderr.toString().trim()}`,
      )
    }
    return {
      snapshot: createWorkspaceSnapshot(directory),
      release: () => rm(directory, { recursive: true, force: true }),
    }
  }
}

/**
 * 基準 revision のソースにも現在と同じ規則を当て、基準になかった違反 ID を拒否する。
 * 台帳ではなく観測した違反を比べるので、台帳への書き足しでは通らない。
 */
export async function checkBoundaryDebtRatchet(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
  checkout: RevisionCheckout = gitRevisionCheckout(snapshot.root),
): Promise<CheckOutcome> {
  if (!options.baseRevision) {
    return { ok: false, lines: ['boundary-debt-ratchet requires --base-revision <git-revision>'] }
  }
  const current = await analyzeModuleBoundaries(snapshot)
  const base = await checkout(options.baseRevision)
  let before: BoundaryAnalysis
  try {
    before = await analyzeModuleBoundaries(base.snapshot)
  } finally {
    await base.release()
  }
  if (before.diagnostics.length > 0) {
    return {
      ok: false,
      lines: before.diagnostics.map(
        (diagnostic) => `fail  ${options.baseRevision} cannot be analyzed: ${diagnostic}`,
      ),
    }
  }
  const beforeIds = new Set(before.violations.map((violation) => violation.id))
  const added = current.violations.filter((violation) => !beforeIds.has(violation.id))
  const counts = `${kindCounts(before)} -> ${kindCounts(current)}`
  if (added.length > 0) {
    return {
      ok: false,
      lines: [
        ...added.map(
          (violation) =>
            `fail  ${violation.id}: absent from ${options.baseRevision}; remove the new dependency or write instead of growing boundary debt.`,
        ),
        `      ${counts}`,
      ],
    }
  }
  return {
    ok: true,
    lines: [`ok  boundary debt ratchet against ${options.baseRevision} (${counts})`],
  }
}

function kindCounts(analysis: BoundaryAnalysis): string {
  const counts = new Map<BoundaryViolationKind, number>()
  for (const violation of analysis.violations) {
    counts.set(violation.kind, (counts.get(violation.kind) ?? 0) + 1)
  }
  return (
    [...counts]
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([kind, count]) => `${kind} ${count}`)
      .join(', ') || 'no violation'
  )
}
