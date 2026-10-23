/**
 * 論理アーキテクチャの責務表を宣言として読み、Go の import がモジュール間の依存規則を守るかを判定する。
 *
 * 規則は公開範囲、モジュール単位の非循環、共有ライブラリと組み立て地点の制約だけであり、
 * A から B への依存を個別に許可する一覧は持たず、すべてのモジュールへ共通規則を適用する。根拠は
 * `docs/design/application/backend.md` が定める。
 */

export type ModuleDefinition = {
  name: string
  packagePrefixes: string[]
  /** 責務表の公開パッケージの列に挙げたパッケージ。ルートと `internal/` の外にあるのはこれだけである。 */
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
  /** モジュール間の import の辺。重複を除き、名前の順に並べる。 */
  dependencies: ModuleDependency[]
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

export const RESPONSIBILITY_HEADER =
  '| モジュール | 公開方式 | 公開パッケージ | Go パッケージ | 責務 |'

type ImportSpec = {
  path: string
  localName: string
}

type PackageOwner =
  | { kind: 'module'; module: ModuleDefinition; prefix: string }
  | { kind: 'composition' }
  | { kind: 'shared' }

export type ModuleDependency = {
  sourceModule: string
  targetModule: string
}

/** 責務表の行を、表と行の現れる順に返す。責務表がなければ `undefined` を返す。 */
function responsibilityRows(markdown: string): string[] | undefined {
  const lines = markdown.split(/\r?\n/)
  const headerIndexes = lines.flatMap((line, index) =>
    line.trim() === RESPONSIBILITY_HEADER ? [index] : [],
  )
  if (headerIndexes.length === 0) return undefined
  return headerIndexes.flatMap((headerIndex) => {
    const end = lines.findIndex(
      (line, index) => index > headerIndex + 1 && !line.trim().startsWith('|'),
    )
    return lines.slice(headerIndex + 2, end < 0 ? undefined : end)
  })
}

/**
 * 責務表の行が入口として張ったモジュール設計文書のディレクトリを、表の順に返す。
 * 表の順は区分ごとに読む順序でもあるので、生成サイトはこの順にモジュールを並べる。
 */
export function responsibilityTableDirectories(markdown: string): string[] {
  return (responsibilityRows(markdown) ?? []).flatMap((line) => {
    const directory = cells(line)[0]?.match(/^\[[^\]]+\]\(([^)/]+)\/README\.md\)/)?.[1]
    return directory ? [directory] : []
  })
}

/**
 * 責務表からモジュール、公開パッケージ、組み立て地点を読む。公開方式は `internal` だけを受け付ける。
 * 責務表は区分ごとに同じ見出しの表へ分けてよく、見出しが一致する表をすべて読む。
 */
export function parseLogicalArchitecture(markdown: string): ArchitectureModel {
  const rows = responsibilityRows(markdown)
  if (rows === undefined) {
    throw new Error('logical architecture must contain the module responsibility table')
  }
  const modules: ModuleDefinition[] = []
  let compositionPrefixes: string[] = []
  for (const line of rows) {
    const row = cells(line)
    const name = row[0]?.match(/^\[([^\]]+)\]/)?.[1]?.trim()
    if (!name) continue
    const packages = backendPaths(row[3] ?? '')
    if (name === COMPOSITION_NAME) {
      compositionPrefixes = packages
      continue
    }
    if (packages.length === 0) {
      throw new Error(`${name}: the responsibility table must declare a backend package`)
    }
    if (row[1]?.replace(/`/g, '').trim() !== 'internal') {
      throw new Error(`${name}: publication mode must be internal`)
    }
    const publicPackages = backendPaths(row[2] ?? '')
    for (const declared of publicPackages) {
      if (declared.split('/').includes('internal')) {
        throw new Error(`${name}: ${declared} is below internal/ and cannot be public`)
      }
      if (!packages.some((prefix) => within(declared, prefix))) {
        throw new Error(`${name}: public package ${declared} is outside the module`)
      }
    }
    modules.push({ name, packagePrefixes: packages, publicPackages })
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
    dependencies: [
      ...new Map(
        dependencies.map((edge) => [`${edge.sourceModule}\n${edge.targetModule}`, edge]),
      ).values(),
    ].sort(
      (left, right) =>
        left.sourceModule.localeCompare(right.sourceModule) ||
        left.targetModule.localeCompare(right.targetModule),
    ),
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

/** 公開パッケージの宣言と、実在するパッケージの食い違いを返す。 */
function publicationDiagnostics(
  architecture: ArchitectureModel,
  packages: ReadonlySet<string>,
): string[] {
  const diagnostics: string[] = []
  for (const module of architecture.modules) {
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
  return path !== owner.prefix && owner.module.publicPackages.includes(path)
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
  if (source.kind === 'composition' && targetPackage === target.prefix) return undefined
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

/** 値の構文だけを扱い、名前解決も接続も行わない `net` 配下のパッケージ。 */
const NET_SYNTAX_PACKAGES = new Set(['net/mail', 'net/netip', 'net/url'])

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
    if (
      (imported.path === 'net' || imported.path.startsWith('net/')) &&
      !NET_SYNTAX_PACKAGES.has(imported.path)
    ) {
      effects.add('net')
    }
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
