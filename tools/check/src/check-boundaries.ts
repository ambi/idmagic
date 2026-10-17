import { extname } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import {
  findBoundaryViolations,
  groupBoundaryViolations,
  ownerOf,
  parseBoundaryDebt,
  parseLogicalArchitecture,
  reconcileBoundaryDebt,
  type BoundaryAnalysis,
  type BoundaryDebtGroup,
  type GoSource,
} from './boundary-fitness.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'
import { describedTables, tableOwners } from './schema-tables.ts'
import {
  extractTableWrites,
  parseSqlFiles,
  schemaColumns,
  tableWriteViolations,
  type QueryInput,
  type QueryWriter,
} from './table-writes.ts'

const DEFAULT_OPTIONS: CheckOptions = { verbose: false, listUnresolved: false }

const LOGICAL_ARCHITECTURE_PATH = 'docs/design/architecture/logical.md'
const DATABASE_DESIGN_PATH = 'docs/design/data/database.md'
const SQLC_CONFIG_PATH = 'sqlc.yaml'
export const BOUNDARY_DEBT_PATH = 'tools/check/boundary-debt.json'

/** 境界検査が読む入力。基準 revision の比較は、この範囲だけを取り出して同じ判定を行う。 */
export const BOUNDARY_INPUT_PATHS = [
  'go.mod',
  'backend',
  'infra/schema',
  SQLC_CONFIG_PATH,
  LOGICAL_ARCHITECTURE_PATH,
  DATABASE_DESIGN_PATH,
]

const EXCLUDED_DIRECTORIES = [
  '.git',
  'node_modules',
  '.worktrees',
  'vendor',
  'dist',
  'build',
  'generated',
]

export async function checkBoundaries(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions = DEFAULT_OPTIONS,
): Promise<CheckOutcome> {
  const lines = await layerAndConfigurationFindings(snapshot)
  // `backend/` を持たない作業ツリーには、モジュールの宣言と台帳を求めない。
  if (!snapshot.exists('backend')) {
    return { ok: lines.length === 0, lines: lines.length > 0 ? lines : ['ok  module boundaries'] }
  }
  if (!snapshot.exists(BOUNDARY_DEBT_PATH)) {
    throw new Error(`${BOUNDARY_DEBT_PATH} is required to ratchet boundary debt`)
  }
  const analysis = await analyzeModuleBoundaries(snapshot)
  lines.push(...analysis.diagnostics.map((diagnostic) => `input: ${diagnostic}`))
  const debtFindings = reconcileBoundaryDebt(
    analysis.violations,
    parseBoundaryDebt(await snapshot.read(BOUNDARY_DEBT_PATH)),
  )
  lines.push(...debtFindings.map((finding) => `${BOUNDARY_DEBT_PATH}: ${finding.message}`))
  if (options.listUnresolved) {
    const unrecordedGroupIds = new Set(
      debtFindings
        .filter((finding) => finding.kind === 'unrecorded-violation')
        .map((finding) => finding.id.split('#')[0] ?? finding.id),
    )
    lines.push(
      ...groupBoundaryViolations(analysis.violations)
        .filter((group) => unrecordedGroupIds.has(group.id))
        .map((group) => `boundary-debt-entry ${proposedDebtEntry(group)}`),
    )
  }
  return {
    ok: lines.length === 0,
    lines: lines.length > 0 ? lines : ['ok  module boundaries'],
  }
}

/**
 * 責務表、テーブルの所有、sqlc の入力を宣言として読み、import とテーブルへの書き込みを判定する。
 * 宣言や SQL を読めないことは違反ではなく診断として返し、台帳で免除させない。
 */
export async function analyzeModuleBoundaries(
  snapshot: WorkspaceSnapshot,
): Promise<BoundaryAnalysis> {
  const modulePath = (await snapshot.read('go.mod')).match(/^module\s+(\S+)$/m)?.[1]
  if (!modulePath) throw new Error('go.mod must declare a module path')
  if (!snapshot.exists(LOGICAL_ARCHITECTURE_PATH)) {
    throw new Error(`${LOGICAL_ARCHITECTURE_PATH} is required to check module boundaries`)
  }
  const architecture = parseLogicalArchitecture(await snapshot.read(LOGICAL_ARCHITECTURE_PATH))
  const sources: GoSource[] = []
  for (const path of await productionGoFiles(snapshot)) {
    sources.push({ path, source: await snapshot.read(path) })
  }
  const imports = findBoundaryViolations({ modulePath, architecture, sources })

  const queryConfig = await sqlcQueryInputs(snapshot)
  const schemaFiles = [...new Set(queryConfig.map((entry) => entry.schema))]
  const queryFiles: { path: string; sql: string; writer: QueryWriter }[] = []
  const diagnostics: string[] = []
  for (const entry of queryConfig) {
    const owner = ownerOf(entry.packagePath, architecture)
    if (!owner) {
      diagnostics.push(
        `${entry.packagePath}: query input belongs to no module, composition point, or shared library`,
      )
      continue
    }
    const writer: QueryWriter =
      owner.kind === 'module' ? { kind: 'module', name: owner.module.name } : { kind: owner.kind }
    for (const path of entry.files) {
      queryFiles.push({ path, sql: await snapshot.read(path), writer })
    }
  }
  const sqlFiles = [
    ...(await Promise.all(
      schemaFiles.map(async (path) => ({ path, sql: await snapshot.read(path) })),
    )),
    ...queryFiles.map(({ path, sql }) => ({ path, sql })),
  ]
  const parsed = sqlFiles.length === 0 ? new Map() : await parseSqlFiles(sqlFiles)
  const columns = schemaColumns(schemaFiles.flatMap((path) => parsed.get(path) ?? []))
  const inputs: QueryInput[] = []
  for (const file of queryFiles) {
    const extracted = extractTableWrites(parsed.get(file.path) ?? [], columns)
    diagnostics.push(...extracted.diagnostics.map((diagnostic) => `${file.path}: ${diagnostic}`))
    inputs.push({ path: file.path, writer: file.writer, writes: extracted.writes })
  }
  const ownership = tableOwners(
    describedTables(await snapshot.read(DATABASE_DESIGN_PATH)),
    new Set(architecture.modules.map((module) => module.name)),
  )
  diagnostics.push(
    ...ownership.diagnostics.map((diagnostic) => `${DATABASE_DESIGN_PATH} ${diagnostic}`),
  )
  const writes = tableWriteViolations(inputs, ownership.owners)
  diagnostics.push(...writes.diagnostics)

  return {
    violations: [...imports.violations, ...writes.violations].sort((left, right) =>
      left.id.localeCompare(right.id),
    ),
    diagnostics: [...imports.diagnostics, ...diagnostics],
  }
}

type SqlcQueryInput = {
  schema: string
  packagePath: string
  files: string[]
}

/** `sqlc.yaml` の各入力を、クエリのファイルと、それを置いたパッケージにする。 */
async function sqlcQueryInputs(snapshot: WorkspaceSnapshot): Promise<SqlcQueryInput[]> {
  const config = Bun.YAML.parse(await snapshot.read(SQLC_CONFIG_PATH)) as {
    sql?: { schema?: unknown; queries?: unknown }[]
  }
  const inputs: SqlcQueryInput[] = []
  for (const entry of config.sql ?? []) {
    if (typeof entry.schema !== 'string' || typeof entry.queries !== 'string') {
      throw new Error(
        `${SQLC_CONFIG_PATH}: every sql entry needs one schema path and one queries path`,
      )
    }
    const queries = entry.queries.replace(/\/+$/, '')
    const files = queries.endsWith('.sql')
      ? [queries]
      : (await snapshot.files(queries)).filter((path) => path.endsWith('.sql'))
    inputs.push({
      schema: entry.schema,
      packagePath: queries.endsWith('.sql') ? queries.split('/').slice(0, -1).join('/') : queries,
      files,
    })
  }
  return inputs
}

async function productionGoFiles(snapshot: WorkspaceSnapshot): Promise<string[]> {
  if (!snapshot.exists('backend')) return []
  return (await snapshot.files('backend', EXCLUDED_DIRECTORIES)).filter(
    (path) => path.endsWith('.go') && !path.endsWith('_test.go'),
  )
}

/** 層の外向き依存、起動設定の読み取り場所、フロントエンドからの import を判定する。 */
async function layerAndConfigurationFindings(snapshot: WorkspaceSnapshot): Promise<string[]> {
  const goModule = (await snapshot.read('go.mod')).match(/^module\s+(\S+)$/m)?.[1]
  if (!goModule) throw new Error('go.mod must declare a module path')
  const backendImportPrefix = `${goModule}/backend/`
  const lines: string[] = []
  for (const rel of await snapshot.files('', EXCLUDED_DIRECTORIES)) {
    if (rel === 'architecture.yaml' || rel.endsWith('/architecture.yaml')) {
      lines.push(
        `${rel}: exhaustive architecture ledgers are retired; enforce only forbidden imports`,
      )
      continue
    }
    const extension = extname(rel)
    if (extension === '.go' && rel.startsWith('backend/') && !rel.endsWith('_test.go')) {
      const source = await snapshot.read(rel)
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
  return lines
}

function proposedDebtEntry(group: BoundaryDebtGroup): string {
  return JSON.stringify({
    id: group.id,
    sourceModule: group.sourceModule,
    ...(group.targetModule ? { targetModule: group.targetModule } : {}),
    violations: group.violationIds,
    reason: `${group.paths[0]} remains across ${group.violationIds.length} observed path(s); replace this text with the concrete coupling and the reference that requires it.`,
  })
}
