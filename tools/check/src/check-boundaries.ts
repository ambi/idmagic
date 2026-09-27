import { extname } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import {
  findBoundaryViolations,
  groupBoundaryViolations,
  parseBoundaryDebt,
  parseLogicalArchitecture,
  reconcileBoundaryDebt,
  type BoundaryDebtGroup,
  type GoSource,
} from './boundary-fitness.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'

const DEFAULT_OPTIONS: CheckOptions = { verbose: false, listUnresolved: false }

function proposedDebtEntry(group: BoundaryDebtGroup): string {
  return JSON.stringify({
    id: group.id,
    sourceContext: group.sourceContext,
    ...(group.targetContext ? { targetContext: group.targetContext } : {}),
    violations: group.violationIds,
    reason:
      `${group.paths[0]} remains coupled across ${group.violationIds.length} observed path(s) ` +
      'until its owning Context exposes or consumes the required published language.',
  })
}

export async function checkBoundaries(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions = DEFAULT_OPTIONS,
): Promise<CheckOutcome> {
  const goModule = (await snapshot.read('go.mod')).match(/^module\s+(\S+)$/m)?.[1]
  if (!goModule) throw new Error('go.mod must declare a module path')
  const backendImportPrefix = `${goModule}/backend/`
  const lines: string[] = []
  const goSources: GoSource[] = []
  const files = await snapshot.files('', [
    '.git',
    'node_modules',
    '.worktrees',
    'vendor',
    'dist',
    'build',
    'generated',
  ])
  for (const rel of files) {
    if (rel === 'architecture.yaml' || rel.endsWith('/architecture.yaml')) {
      lines.push(
        `${rel}: exhaustive architecture ledgers are retired; enforce only forbidden imports`,
      )
      continue
    }
    const extension = extname(rel)
    if (extension === '.go' && rel.startsWith('backend/') && !rel.endsWith('_test.go')) {
      const source = await snapshot.read(rel)
      goSources.push({ path: rel, source })
      // 起動設定は `backend/cmd/internal/bootstrap` だけが環境から読み、検証する。
      // `*_env` adapter は起動設定ではなく、実行時 locator を解決するので対象外とする。
      const ownsEnvAccess =
        rel.startsWith('backend/cmd/internal/bootstrap/') || /\/[a-z0-9]+_env\//.test(rel)
      if (!ownsEnvAccess) {
        const envReads = [...source.matchAll(/os\.(?:Getenv|LookupEnv|Environ)\b/g)].filter(
          (match) =>
            !source
              .slice(Math.max(0, (match.index ?? 0) - 30), match.index)
              .includes('bootstrap.NewConfigLoader('),
        )
        if (envReads.length > 0) {
          lines.push(
            `${rel}: startup configuration must be read through bootstrap's ConfigLoader, not os.Getenv`,
          )
        }
      }
      const imports = [...source.matchAll(/"([^"\n]+)"/g)]
        .map((match) => match[1] ?? '')
        .filter((imported) => imported.startsWith(backendImportPrefix))
      const isDomain = rel.split('/').includes('domain')
      const isUseCase = rel.split('/').includes('usecases')
      for (const imported of imports) {
        const outward = /\/(?:handlers?_[^/]+|db_[^/]+|delivery|cmd)(?:\/|$)/.test(imported)
        const useCase = /\/usecases(?:\/|$)/.test(imported)
        if ((isDomain && (outward || useCase)) || (isUseCase && outward)) {
          lines.push(`${rel}: forbidden outward dependency ${imported}`)
        }
      }
    }
    if (['.ts', '.tsx'].includes(extension) && rel.startsWith('frontend/src/')) {
      const source = await snapshot.read(rel)
      if (/from\s+['"](?:\.\.\/)+(?:backend|tools)\//.test(source)) {
        lines.push(`${rel}: frontend source must not import backend or repository tooling`)
      }
    }
  }
  if (goSources.length > 0) {
    const logicalArchitecturePath = 'docs/design/architecture/logical.md'
    const debtPath = 'tools/check/boundary-debt.json'
    if (!snapshot.exists(logicalArchitecturePath)) {
      throw new Error(`${logicalArchitecturePath} is required to check Context boundaries`)
    }
    if (!snapshot.exists(debtPath))
      throw new Error(`${debtPath} is required to ratchet boundary debt`)
    const violations = findBoundaryViolations({
      modulePath: goModule,
      architecture: parseLogicalArchitecture(await snapshot.read(logicalArchitecturePath)),
      sources: goSources,
    })
    const debtFindings = reconcileBoundaryDebt(
      violations,
      parseBoundaryDebt(await snapshot.read(debtPath)),
    )
    lines.push(...debtFindings.map((finding) => `${debtPath}: ${finding.message}`))
    if (options.listUnresolved) {
      const unrecordedGroupIds = new Set(
        debtFindings
          .filter((finding) => finding.kind === 'unrecorded-violation')
          .map((finding) => finding.id.split('#')[0] ?? finding.id),
      )
      lines.push(
        ...groupBoundaryViolations(violations)
          .filter((group) => unrecordedGroupIds.has(group.id))
          .map((group) => `boundary-debt-entry ${proposedDebtEntry(group)}`),
      )
    }
  }
  return {
    ok: lines.length === 0,
    lines: lines.length > 0 ? lines : ['ok  forbidden dependency boundaries'],
  }
}
