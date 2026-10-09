/**
 * 論理アーキテクチャの責務表を宣言として読み、Go の import がモジュール間の依存規則を守るかを判定する。
 *
 * 規則は公開範囲、モジュール単位の非循環、共有ライブラリと組み立て地点の制約だけであり、
 * モジュールの組ごとの許可リストは持たない。根拠は `docs/domain/structure.md` が定める。
 */

export type PublicationMode = 'legacy' | 'internal'

export type ModuleDefinition = {
  name: string
  packagePrefixes: string[]
  publication: PublicationMode
  /** `internal` のモジュールだけが列挙する。`legacy` は区画の名前で判定するので空にする。 */
  publicPackages: string[]
}

export type ArchitectureModel = {
  modules: ModuleDefinition[]
  compositionPrefixes: string[]
}

export type GoSource = {
  path: string
  source: string
}

export type BoundaryViolationKind =
  | 'private-import'
  | 'module-cycle'
  | 'shared-dependency'
  | 'composition-import'
  | 'domain-effect'
  | 'table-write'

export type BoundaryViolation = {
  id: string
  debtId: string
  kind: BoundaryViolationKind
  sourceModule: string
  targetModule?: string
  path: string
  message: string
}

/** 違反の一覧と、台帳で免除できない入力の誤り。 */
export type BoundaryAnalysis = {
  violations: BoundaryViolation[]
  diagnostics: string[]
}

export type BoundaryDebtEntry = {
  id: string
  sourceModule: string
  targetModule?: string
  violations: string[]
  reason: string
}

export type BoundaryDebtGroup = {
  id: string
  sourceModule: string
  targetModule?: string
  violationIds: string[]
  paths: string[]
}

export type BoundaryDebtFinding = {
  id: string
  kind: 'unrecorded-violation' | 'stale-debt' | 'invalid-debt'
  message: string
}

/** 組み立て地点の名前。違反の出所として表示し、台帳の項目を分ける。 */
export const COMPOSITION_NAME = 'System'

/** 共有ライブラリの名前。テーブルの所有者「共通基盤」とは別に、import の出所として表示する。 */
export const SHARED_LIBRARY_NAME = 'shared'

const SHARED_PREFIX = 'backend/shared'

const RESPONSIBILITY_HEADER = '| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |'

/**
 * 責務表を改めた revision より前の表。基準 revision の比較だけがこの形を読む。
 * 旧表は組み立て地点を列挙しないので、新しい表が System 行に宣言する初期値を使う。
 */
const RETIRED_RESPONSIBILITY_HEADER = '| 仕様上の Context | Subdomain | Go パッケージ | 責務 |'
const RETIRED_COMPOSITION_PREFIXES = [
  'backend/cmd',
  'backend/shared/http/server_http',
  'backend/shared/http/testing_stack',
]

type ImportSpec = {
  path: string
  localName: string
}

type PackageOwner =
  | { kind: 'module'; module: ModuleDefinition; prefix: string }
  | { kind: 'composition' }
  | { kind: 'shared' }

type ModuleDependency = {
  sourceModule: string
  targetModule: string
}

/** 責務表からモジュール、公開方式、公開パッケージ、組み立て地点を読む。 */
export function parseLogicalArchitecture(markdown: string): ArchitectureModel {
  const lines = markdown.split(/\r?\n/)
  const headerIndex = lines.findIndex(
    (line) =>
      line.trim() === RESPONSIBILITY_HEADER || line.trim() === RETIRED_RESPONSIBILITY_HEADER,
  )
  if (headerIndex < 0) {
    throw new Error('logical architecture must contain the module responsibility table')
  }
  const retired = lines[headerIndex]!.trim() === RETIRED_RESPONSIBILITY_HEADER
  const modules: ModuleDefinition[] = []
  let compositionPrefixes: string[] = retired ? [...RETIRED_COMPOSITION_PREFIXES] : []
  for (const line of lines.slice(headerIndex + 2)) {
    if (!line.trim().startsWith('|')) break
    const row = cells(line)
    const name = row[0]?.match(/^\[([^\]]+)\]/)?.[1]?.trim()
    if (!name) continue
    const packages = backendPaths(row[retired ? 2 : 3] ?? '')
    if (name === COMPOSITION_NAME) {
      if (!retired) compositionPrefixes = packages
      continue
    }
    if (packages.length === 0) {
      throw new Error(`${name}: the responsibility table must declare a backend package`)
    }
    if (retired) {
      modules.push({ name, packagePrefixes: packages, publication: 'legacy', publicPackages: [] })
      continue
    }
    const publication = row[1]?.replace(/`/g, '').trim()
    if (publication !== 'legacy' && publication !== 'internal') {
      throw new Error(`${name}: publication mode must be legacy or internal`)
    }
    const publicPackages = backendPaths(row[2] ?? '')
    if (publication === 'legacy' && publicPackages.length > 0) {
      throw new Error(`${name}: a legacy module derives its public packages from domain and ports`)
    }
    for (const declared of publicPackages) {
      if (declared.split('/').includes('internal')) {
        throw new Error(`${name}: ${declared} is below internal/ and cannot be public`)
      }
      if (!packages.some((prefix) => within(declared, prefix))) {
        throw new Error(`${name}: public package ${declared} is outside the module`)
      }
    }
    modules.push({ name, packagePrefixes: packages, publication, publicPackages })
  }
  if (modules.length === 0) {
    throw new Error('the module responsibility table declares no module')
  }
  validateAssignments(modules, compositionPrefixes)
  modules.sort((left, right) => left.name.localeCompare(right.name))
  return { modules, compositionPrefixes: [...compositionPrefixes].sort() }
}

/** 本番の Go ソースから依存規則の違反と、分類できない入力を求める。 */
export function findBoundaryViolations(input: {
  modulePath: string
  architecture: ArchitectureModel
  sources: readonly GoSource[]
}): BoundaryAnalysis {
  const { architecture } = input
  const violations: BoundaryViolation[] = []
  const diagnostics: string[] = []
  const dependencies: ModuleDependency[] = []
  const packages = new Set(input.sources.map((source) => packageForFile(source.path)))

  for (const packagePath of [...packages].sort()) {
    if (!ownerOf(packagePath, architecture)) {
      diagnostics.push(
        `${packagePath}: production package belongs to no module, composition point, or shared library`,
      )
    }
  }
  diagnostics.push(...publicationDiagnostics(architecture, packages))

  for (const source of input.sources) {
    const sourcePackage = packageForFile(source.path)
    const imports = goImports(source.source)
    violations.push(...domainEffectViolations(source, imports))
    const sourceOwner = ownerOf(sourcePackage, architecture)
    if (!sourceOwner) continue
    for (const imported of imports) {
      const targetPackage = repositoryImport(input.modulePath, imported.path)
      if (!targetPackage) continue
      const targetOwner = ownerOf(targetPackage, architecture)
      if (!targetOwner) continue
      const violation = importViolation(sourcePackage, sourceOwner, targetPackage, targetOwner)
      if (violation) violations.push(violation)
      if (
        sourceOwner.kind === 'module' &&
        targetOwner.kind === 'module' &&
        sourceOwner.module.name !== targetOwner.module.name
      ) {
        dependencies.push({
          sourceModule: sourceOwner.module.name,
          targetModule: targetOwner.module.name,
        })
      }
    }
  }

  violations.push(...cycleViolations(dependencies))
  return {
    violations: [
      ...new Map(violations.map((violation) => [violation.id, violation])).values(),
    ].sort(compareById),
    diagnostics,
  }
}

/** パッケージがどのモジュール、組み立て地点、共有ライブラリに属するか。組み立て地点を先に照合する。 */
export function ownerOf(path: string, architecture: ArchitectureModel): PackageOwner | undefined {
  if (architecture.compositionPrefixes.some((prefix) => within(path, prefix))) {
    return { kind: 'composition' }
  }
  const match = architecture.modules
    .flatMap((module) => module.packagePrefixes.map((prefix) => ({ module, prefix })))
    .filter(({ prefix }) => within(path, prefix))
    .sort((left, right) => right.prefix.length - left.prefix.length)[0]
  if (match) return { kind: 'module', ...match }
  if (within(path, SHARED_PREFIX)) return { kind: 'shared' }
  return undefined
}

export function groupBoundaryViolations(
  violations: readonly BoundaryViolation[],
): BoundaryDebtGroup[] {
  const groups = new Map<string, BoundaryDebtGroup>()
  for (const violation of violations) {
    const group = groups.get(violation.debtId) ?? {
      id: violation.debtId,
      sourceModule: violation.sourceModule,
      ...(violation.targetModule ? { targetModule: violation.targetModule } : {}),
      violationIds: [],
      paths: [],
    }
    group.violationIds.push(violation.id)
    group.paths.push(violation.path)
    groups.set(violation.debtId, group)
  }
  return [...groups.values()]
    .map((group) => ({
      ...group,
      violationIds: [...new Set(group.violationIds)].sort(),
      paths: [...new Set(group.paths)].sort(),
    }))
    .sort(compareById)
}

export function parseBoundaryDebt(source: string): BoundaryDebtEntry[] {
  const parsed = JSON.parse(source) as { violations?: unknown }
  if (!Array.isArray(parsed.violations)) {
    throw new Error('tools/check/boundary-debt.json: violations must be an array')
  }
  const entries: BoundaryDebtEntry[] = []
  const ids = new Set<string>()
  for (const value of parsed.violations) {
    if (typeof value !== 'object' || value === null || Array.isArray(value)) {
      throw new Error('tools/check/boundary-debt.json: every violation must be an object')
    }
    const entry = value as Record<string, unknown>
    if (
      typeof entry.id !== 'string' ||
      typeof entry.sourceModule !== 'string' ||
      typeof entry.reason !== 'string' ||
      !Array.isArray(entry.violations) ||
      entry.violations.some((violation) => typeof violation !== 'string') ||
      (entry.targetModule !== undefined && typeof entry.targetModule !== 'string')
    ) {
      throw new Error(
        'tools/check/boundary-debt.json: every entry needs id, sourceModule, optional targetModule, violations, and reason',
      )
    }
    const violationIds = [...new Set(entry.violations as string[])].sort()
    if (violationIds.length === 0 || violationIds.length !== entry.violations.length) {
      throw new Error(`${entry.id}: violations must be a non-empty array of unique ids`)
    }
    if (
      entry.reason.trim().length < 12 ||
      /^(?:existing|legacy|既存)(?:\s+violation|のため)?[.。]?$/i.test(entry.reason.trim())
    ) {
      throw new Error(`${entry.id}: boundary debt reason must explain the concrete coupling`)
    }
    if (ids.has(entry.id)) throw new Error(`${entry.id}: duplicate boundary debt id`)
    ids.add(entry.id)
    entries.push({
      id: entry.id,
      sourceModule: entry.sourceModule,
      targetModule: entry.targetModule as string | undefined,
      violations: violationIds,
      reason: entry.reason,
    })
  }
  return entries.sort(compareById)
}

export function reconcileBoundaryDebt(
  violations: readonly BoundaryViolation[],
  debt: readonly BoundaryDebtEntry[],
): BoundaryDebtFinding[] {
  const groups = groupBoundaryViolations(violations)
  const groupById = new Map(groups.map((group) => [group.id, group]))
  const debtById = new Map(debt.map((entry) => [entry.id, entry]))
  const findings: BoundaryDebtFinding[] = []
  for (const group of groups) {
    const entry = debtById.get(group.id)
    if (!entry) {
      findings.push({
        id: group.id,
        kind: 'unrecorded-violation',
        message: `${group.id}: ${group.violationIds.length} violation(s) are absent from boundary-debt.json`,
      })
      continue
    }
    if (entry.sourceModule !== group.sourceModule || entry.targetModule !== group.targetModule) {
      findings.push({
        id: group.id,
        kind: 'invalid-debt',
        message: `${group.id}: debt module fields do not match the observed dependency`,
      })
    }
    const currentIds = new Set(group.violationIds)
    const debtIds = new Set(entry.violations)
    for (const id of group.violationIds) {
      if (debtIds.has(id)) continue
      findings.push({
        id: `${group.id}#${id}`,
        kind: 'unrecorded-violation',
        message: `${id}: violation is absent from boundary-debt.json entry ${group.id}`,
      })
    }
    for (const id of entry.violations) {
      if (currentIds.has(id)) continue
      findings.push({
        id: `${group.id}#${id}`,
        kind: 'stale-debt',
        message: `${id}: no observed violation has this id; remove it from ${group.id}`,
      })
    }
  }
  for (const entry of debt) {
    if (groupById.has(entry.id)) continue
    findings.push({
      id: entry.id,
      kind: 'stale-debt',
      message: `${entry.id}: no observed violation belongs to this entry; remove it`,
    })
  }
  return findings.sort(compareById)
}

export function compareById(left: { id: string }, right: { id: string }): number {
  return left.id.localeCompare(right.id)
}

/** ディレクトリ区画で照合する。`backend/cmd` は `backend/cmdx` を含めない。 */
function within(path: string, prefix: string): boolean {
  return path === prefix || path.startsWith(`${prefix}/`)
}

function cells(row: string): string[] {
  const parts = row.split('|').map((cell) => cell.trim())
  if (parts[0] === '') parts.shift()
  if (parts.at(-1) === '') parts.pop()
  return parts
}

function backendPaths(cell: string): string[] {
  return [...cell.matchAll(/`([^`]+)`/g)]
    .map((match) => normalizeRepositoryPath(match[1] ?? ''))
    .filter((path) => within(path, 'backend'))
    .sort()
}

function normalizeRepositoryPath(path: string): string {
  return path.replace(/^\.\//, '').replace(/\/+$/, '')
}

function validateAssignments(
  modules: readonly ModuleDefinition[],
  compositionPrefixes: readonly string[],
): void {
  const seenNames = new Set<string>()
  for (const module of modules) {
    if (seenNames.has(module.name)) {
      throw new Error(`${module.name}: the responsibility table repeats a module name`)
    }
    seenNames.add(module.name)
  }
  const seenPrefixes = new Set<string>()
  for (const prefix of [
    ...compositionPrefixes,
    ...modules.flatMap((module) => module.packagePrefixes),
  ]) {
    if (seenPrefixes.has(prefix)) throw new Error(`${prefix} is assigned more than once`)
    seenPrefixes.add(prefix)
  }
  for (const module of modules) {
    for (const prefix of module.packagePrefixes) {
      if (within(prefix, SHARED_PREFIX) || compositionPrefixes.some((c) => within(prefix, c))) {
        throw new Error(
          `${module.name}: ${prefix} belongs to a shared library or composition point`,
        )
      }
    }
  }
}

/** `internal` のモジュールで、宣言と実在するパッケージの食い違いを返す。 */
function publicationDiagnostics(
  architecture: ArchitectureModel,
  packages: ReadonlySet<string>,
): string[] {
  const diagnostics: string[] = []
  for (const module of architecture.modules) {
    if (module.publication !== 'internal') continue
    for (const declared of module.publicPackages) {
      if (!packages.has(declared)) {
        diagnostics.push(
          `${module.name}: ${declared} is declared public but has no production Go file`,
        )
      }
    }
    for (const packagePath of [...packages].sort()) {
      const owner = ownerOf(packagePath, architecture)
      if (owner?.kind !== 'module' || owner.module.name !== module.name) continue
      if (packagePath === owner.prefix || module.publicPackages.includes(packagePath)) continue
      if (packagePath.slice(owner.prefix.length).split('/').includes('internal')) continue
      diagnostics.push(
        `${module.name}: ${packagePath} is outside internal/ but is neither the root nor a declared public package`,
      )
    }
  }
  return diagnostics
}

/** ほかのモジュールが import できるか。ルートパッケージは組み立て地点だけに公開する。 */
function isPublicPackage(path: string, owner: Extract<PackageOwner, { kind: 'module' }>): boolean {
  if (path === owner.prefix) return false
  if (owner.module.publication === 'internal') return owner.module.publicPackages.includes(path)
  const segments = path.slice(owner.prefix.length).split('/').filter(Boolean)
  if (segments.includes('internal')) return false
  return segments.some((segment) => segment === 'domain' || segment === 'ports')
}

function importViolation(
  sourcePackage: string,
  source: PackageOwner,
  targetPackage: string,
  target: PackageOwner,
): BoundaryViolation | undefined {
  const sourceName = ownerName(source)
  if (target.kind === 'composition') {
    if (source.kind === 'composition') return undefined
    return {
      id: `composition-import:${sourcePackage}->${targetPackage}`,
      debtId: `composition-import:${sourceName}`,
      kind: 'composition-import',
      sourceModule: sourceName,
      path: `${sourcePackage} -> ${targetPackage}`,
      message: `${sourcePackage} imports composition point ${targetPackage}`,
    }
  }
  if (target.kind === 'shared') return undefined
  const targetName = target.module.name
  if (source.kind === 'shared') {
    return {
      id: `shared-dependency:${sourcePackage}->${targetName}`,
      debtId: `shared-dependency:${SHARED_LIBRARY_NAME}->${targetName}`,
      kind: 'shared-dependency',
      sourceModule: SHARED_LIBRARY_NAME,
      targetModule: targetName,
      path: `${sourcePackage} -> ${targetName}`,
      message: `shared library ${sourcePackage} depends on module ${targetName}`,
    }
  }
  if (source.kind === 'module' && source.module.name === targetName) return undefined
  if (isPublicPackage(targetPackage, target)) return undefined
  if (source.kind === 'composition') {
    const reachable = target.module.publication === 'legacy' || targetPackage === target.prefix
    if (reachable) return undefined
  }
  return {
    id: `private-import:${sourceName}:${sourcePackage}->${targetPackage}`,
    debtId: `private-import:${sourceName}->${targetName}`,
    kind: 'private-import',
    sourceModule: sourceName,
    targetModule: targetName,
    path: `${sourcePackage} -> ${targetPackage}`,
    message: `${sourceName} imports private package ${targetPackage} from ${targetName}`,
  }
}

function ownerName(owner: PackageOwner): string {
  if (owner.kind === 'module') return owner.module.name
  return owner.kind === 'composition' ? COMPOSITION_NAME : SHARED_LIBRARY_NAME
}

function importSpec(line: string): ImportSpec | undefined {
  const match = line.trim().match(/^(?:([A-Za-z_][A-Za-z0-9_]*|[._])\s+)?"([^"\n]+)"/)
  const path = match?.[2]
  if (!path) return undefined
  const localName = match?.[1] ?? path.split('/').at(-1) ?? ''
  return { path, localName }
}

function goImports(source: string): ImportSpec[] {
  const imports: ImportSpec[] = []
  let inImportBlock = false
  for (const line of source.split('\n')) {
    const trimmed = line.trim()
    if (inImportBlock) {
      if (/^\)/.test(trimmed)) {
        inImportBlock = false
        continue
      }
      const parsed = importSpec(trimmed)
      if (parsed) imports.push(parsed)
      continue
    }
    if (/^import\s*\(/.test(trimmed)) {
      inImportBlock = true
      continue
    }
    const single = trimmed.match(/^import\s+(.+)$/)?.[1]
    if (!single) continue
    const parsed = importSpec(single)
    if (parsed) imports.push(parsed)
  }
  return imports
}

function packageForFile(path: string): string {
  return normalizeRepositoryPath(path).split('/').slice(0, -1).join('/')
}

function repositoryImport(modulePath: string, imported: string): string | undefined {
  const prefix = `${modulePath}/`
  return imported.startsWith(prefix) ? imported.slice(prefix.length) : undefined
}

function domainEffectViolations(
  source: GoSource,
  imports: readonly ImportSpec[],
): BoundaryViolation[] {
  if (!normalizeRepositoryPath(source.path).split('/').includes('domain')) return []
  const sourceDirectory = normalizeRepositoryPath(source.path).split('/')[1] ?? 'unknown'
  const effects = new Set<string>()
  for (const imported of imports) {
    if (imported.path === 'crypto/rand') effects.add('crypto/rand')
    if (imported.path === 'database/sql') effects.add('database/sql')
    if (imported.path === 'math/rand' || imported.path.startsWith('math/rand/')) {
      effects.add('math/rand')
    }
    if (imported.path === 'net' || imported.path.startsWith('net/')) effects.add('net')
    if (imported.path === 'os' || imported.path.startsWith('os/')) effects.add('os')
    if (imported.path === 'time') {
      const escapedName = imported.localName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
      const call =
        imported.localName === '.' ? /\bNow\s*\(/ : new RegExp(`\\b${escapedName}\\.Now\\s*\\(`)
      if (call.test(source.source)) effects.add('time.Now')
    }
  }
  return [...effects].sort().map(
    (effect): BoundaryViolation => ({
      id: `domain-effect:${source.path}:${effect}`,
      debtId: `domain-effect:${sourceDirectory}`,
      kind: 'domain-effect',
      sourceModule: sourceDirectory,
      path: source.path,
      message: `${source.path}: domain packages must receive ${effect} effects through an input or port`,
    }),
  )
}

function cycleViolations(edges: readonly ModuleDependency[]): BoundaryViolation[] {
  const key = (edge: ModuleDependency) => `${edge.sourceModule}\0${edge.targetModule}`
  const uniqueEdges = new Map(edges.map((edge) => [key(edge), edge]))
  const adjacency = new Map<string, Set<string>>()
  for (const edge of uniqueEdges.values()) {
    const targets = adjacency.get(edge.sourceModule) ?? new Set<string>()
    targets.add(edge.targetModule)
    adjacency.set(edge.sourceModule, targets)
  }
  const reaches = (from: string, target: string): boolean => {
    const pending = [from]
    const seen = new Set<string>()
    while (pending.length > 0) {
      const current = pending.pop()!
      if (current === target) return true
      if (seen.has(current)) continue
      seen.add(current)
      pending.push(...(adjacency.get(current) ?? []))
    }
    return false
  }
  return [...uniqueEdges.values()]
    .filter((edge) => reaches(edge.targetModule, edge.sourceModule))
    .map(
      (edge): BoundaryViolation => ({
        id: `module-cycle:${edge.sourceModule}->${edge.targetModule}`,
        debtId: `module-cycle:${edge.sourceModule}->${edge.targetModule}`,
        kind: 'module-cycle',
        sourceModule: edge.sourceModule,
        targetModule: edge.targetModule,
        path: `${edge.sourceModule} -> ${edge.targetModule}`,
        message: `${edge.sourceModule} -> ${edge.targetModule} participates in a module dependency cycle`,
      }),
    )
    .sort(compareById)
}
