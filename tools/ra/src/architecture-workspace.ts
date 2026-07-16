import { readFile, readdir } from 'node:fs/promises'
import { dirname, extname, join, normalize, relative, sep } from 'node:path'

export type ArchitectureSourceFile = {
  path: string
  content: string
}

export type ArchitectureWorkspaceSnapshot = {
  files: ArchitectureSourceFile[]
  goModulePath?: string
  tsAliases?: Record<string, string[]>
}

export type ArchitectureWorkspaceFinding = {
  kind: 'import' | 'complexity'
  path: string
  message: string
}

export type ArchitectureWorkspaceOptions = {
  today?: string
}

type ModuleRecord = {
  id: string
  path: string
  dependencies: Set<string>
}

type BudgetRecord = {
  id: string
  metric: string
  includes: string[]
  excludes: string[]
  limit: number
}

type DebtRecord = {
  budget: string
  path: string
  ceiling?: number
  expiresAt?: string
}

const SOURCE_EXTENSIONS = new Set(['.go', '.ts', '.tsx', '.js', '.jsx'])
const ALWAYS_EXCLUDED_DIRECTORIES = new Set(['.git', 'vendor', 'node_modules'])

function slash(path: string): string {
  return path.split(sep).join('/').replace(/^\.\//, '')
}

export function isProductionSource(path: string): boolean {
  const normalized = slash(path)
  const parts = normalized.split('/')
  const name = parts.at(-1) ?? ''
  if (parts.some((part) => ALWAYS_EXCLUDED_DIRECTORIES.has(part))) return false
  if (parts.includes('generated') || parts.includes('sqlc')) return false
  if (parts.includes('pgfixtures')) return false
  if (name === 'routeTree.gen.ts') return false
  if (name.endsWith('_test.go')) return false
  if (/\.(?:test|spec)\.[cm]?[jt]sx?$/.test(name)) return false
  return SOURCE_EXTENSIONS.has(extname(name))
}

async function walkSourceFiles(root: string, dir = root): Promise<ArchitectureSourceFile[]> {
  const files: ArchitectureSourceFile[] = []
  const entries = await readdir(dir, { withFileTypes: true })
  entries.sort((left, right) => left.name.localeCompare(right.name))
  for (const entry of entries) {
    if (entry.isDirectory() && ALWAYS_EXCLUDED_DIRECTORIES.has(entry.name)) continue
    const absolute = join(dir, entry.name)
    if (entry.isDirectory()) {
      files.push(...(await walkSourceFiles(root, absolute)))
      continue
    }
    if (!entry.isFile()) continue
    const path = slash(relative(root, absolute))
    if (!isProductionSource(path)) continue
    files.push({ path, content: await readFile(absolute, 'utf8') })
  }
  return files
}

function parseGoModule(text: string): string | undefined {
  return text.match(/^\s*module\s+([^\s]+)\s*$/m)?.[1]
}

type TsConfig = {
  extends?: unknown
  compilerOptions?: {
    baseUrl?: unknown
    paths?: unknown
  }
}

function stripJsonComments(text: string): string {
  return text
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/(^|[^:])\/\/.*$/gm, '$1')
    .replace(/,\s*([}\]])/g, '$1')
}

async function findTsConfigs(root: string, dir = root): Promise<string[]> {
  const configs: string[] = []
  const entries = await readdir(dir, { withFileTypes: true })
  for (const entry of entries) {
    if (entry.isDirectory() && ALWAYS_EXCLUDED_DIRECTORIES.has(entry.name)) continue
    const absolute = join(dir, entry.name)
    if (entry.isDirectory()) {
      configs.push(...(await findTsConfigs(root, absolute)))
    } else if (entry.isFile() && /^tsconfig(?:\.[^.]+)?\.json$/.test(entry.name)) {
      configs.push(absolute)
    }
  }
  return configs.sort()
}

async function loadTsAliases(
  workspaceRoot: string,
  path: string,
): Promise<Record<string, string[]>> {
  let config: TsConfig
  try {
    config = JSON.parse(stripJsonComments(await readFile(path, 'utf8'))) as TsConfig
  } catch {
    return {}
  }
  const paths = config.compilerOptions?.paths
  if (!paths || typeof paths !== 'object' || Array.isArray(paths)) return {}
  const baseUrl =
    typeof config.compilerOptions?.baseUrl === 'string' ? config.compilerOptions.baseUrl : '.'
  const configRoot = slash(relative(workspaceRoot, dirname(path)))
  const aliases: Record<string, string[]> = {}
  for (const [alias, targets] of Object.entries(paths as Record<string, unknown>)) {
    if (!Array.isArray(targets)) continue
    aliases[alias] = targets
      .filter((target): target is string => typeof target === 'string')
      .map((target) => slash(normalize(join(configRoot, baseUrl, target))))
  }
  return aliases
}

async function loadWorkspaceTsAliases(root: string): Promise<Record<string, string[]>> {
  const aliases: Record<string, string[]> = {}
  for (const path of await findTsConfigs(root)) {
    Object.assign(aliases, await loadTsAliases(root, path))
  }
  return aliases
}

export async function collectArchitectureWorkspace(
  workspaceRoot: string,
): Promise<ArchitectureWorkspaceSnapshot> {
  let goModulePath: string | undefined
  try {
    goModulePath = parseGoModule(await readFile(join(workspaceRoot, 'go.mod'), 'utf8'))
  } catch {
    // A workspace without Go sources does not need a go.mod.
  }
  return {
    files: await walkSourceFiles(workspaceRoot),
    ...(goModulePath ? { goModulePath } : {}),
    tsAliases: await loadWorkspaceTsAliases(workspaceRoot),
  }
}

function object(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : undefined
}

function dependencyTarget(value: unknown): string | undefined {
  if (typeof value === 'string') return value
  const edge = object(value)
  if (!edge) return undefined
  for (const key of ['module', 'target', 'id']) {
    if (typeof edge[key] === 'string') return edge[key]
  }
  return undefined
}

function modulesFrom(doc: unknown): ModuleRecord[] {
  const modules = object(object(doc)?.modules)
  if (!modules) return []
  return Object.entries(modules)
    .flatMap(([id, value]) => {
      const record = object(value)
      if (!record || typeof record.path !== 'string') return []
      const dependencies = Array.isArray(record.depends_on)
        ? record.depends_on.flatMap((dependency) => dependencyTarget(dependency) ?? [])
        : []
      return {
        id,
        path: slash(normalize(record.path)).replace(/\/$/, ''),
        dependencies: new Set(dependencies),
      }
    })
    .sort((left, right) => right.path.length - left.path.length || left.id.localeCompare(right.id))
}

function stringList(value: unknown): string[] {
  if (typeof value === 'string') return [value]
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []
}

function records(value: unknown): Array<[string, Record<string, unknown>]> {
  if (Array.isArray(value)) {
    return value.flatMap((item, index) => {
      const record = object(item)
      if (!record) return []
      const id = typeof record.id === 'string' ? record.id : String(index)
      return [[id, record] as [string, Record<string, unknown>]]
    })
  }
  const map = object(value)
  return map
    ? Object.entries(map).flatMap(([id, item]) => {
        const record = object(item)
        return record ? [[id, record] as [string, Record<string, unknown>]] : []
      })
    : []
}

function complexityFrom(doc: unknown): { budgets: BudgetRecord[]; debts: DebtRecord[] } {
  const complexity = object(object(doc)?.complexity)
  const budgets = records(complexity?.budgets).flatMap(([id, record]) => {
    const metric = record.metric ?? record.measure
    const limit = record.limit
    const includes = stringList(record.include ?? record.includes ?? record.glob)
    if (typeof metric !== 'string' || typeof limit !== 'number' || includes.length === 0) return []
    return [
      { id, metric, includes, excludes: stringList(record.exclude ?? record.excludes), limit },
    ]
  })
  const debts = records(complexity?.debts).flatMap(([, record]) => {
    const budget = record.budget ?? record.budget_id
    if (typeof budget !== 'string' || typeof record.path !== 'string') return []
    const ceiling = typeof record.ceiling === 'number' ? record.ceiling : undefined
    const expiresAt = record.expires_at ?? record.expiry ?? record.deadline
    return [
      {
        budget,
        path: slash(record.path),
        ...(ceiling !== undefined ? { ceiling } : {}),
        ...(typeof expiresAt === 'string' ? { expiresAt } : {}),
      },
    ]
  })
  return { budgets, debts }
}

function moduleForPath(path: string, modules: ModuleRecord[]): ModuleRecord | undefined {
  return modules.find((module) => {
    const extensionless = module.path.slice(0, module.path.length - extname(module.path).length)
    return (
      path === module.path ||
      (extensionless !== module.path && path === extensionless) ||
      path.startsWith(module.path === '.' ? '' : `${module.path}/`)
    )
  })
}

function goImports(content: string): string[] {
  const imports: string[] = []
  const single = /^\s*import\s+(?:[._A-Za-z][\w.]*\s+)?"([^"]+)"/gm
  for (const match of content.matchAll(single)) if (match[1]) imports.push(match[1])
  const blocks = content.matchAll(/^\s*import\s*\(([\s\S]*?)^\s*\)/gm)
  for (const block of blocks) {
    for (const match of (block[1] ?? '').matchAll(
      /(?:^|\n)\s*(?:[._A-Za-z][\w.]*\s+)?"([^"]+)"/g,
    )) {
      if (match[1]) imports.push(match[1])
    }
  }
  return imports
}

function tsImports(content: string): string[] {
  const imports: string[] = []
  const pattern =
    /(?:\b(?:import|export)\s+(?:[\s\S]*?\s+from\s+)?|\b(?:import|require)\s*\()\s*['"]([^'"]+)['"]/g
  for (const match of content.matchAll(pattern)) if (match[1]) imports.push(match[1])
  return imports
}

function applyAlias(specifier: string, aliases: Record<string, string[]>): string | undefined {
  for (const [pattern, targets] of Object.entries(aliases)) {
    const star = pattern.indexOf('*')
    const prefix = star < 0 ? pattern : pattern.slice(0, star)
    const suffix = star < 0 ? '' : pattern.slice(star + 1)
    if (!specifier.startsWith(prefix) || !specifier.endsWith(suffix)) continue
    if (star < 0 && specifier !== pattern) continue
    const capture = specifier.slice(prefix.length, specifier.length - suffix.length)
    const target = targets[0]
    if (target) return slash(target.replace('*', capture))
  }
  return undefined
}

function importTarget(
  file: ArchitectureSourceFile,
  specifier: string,
  snapshot: ArchitectureWorkspaceSnapshot,
): string | undefined {
  if (extname(file.path) === '.go') {
    const module = snapshot.goModulePath
    if (!module || (specifier !== module && !specifier.startsWith(`${module}/`))) return undefined
    return specifier === module ? '.' : specifier.slice(module.length + 1)
  }
  if (specifier.startsWith('.')) {
    const target = slash(normalize(join(dirname(file.path), specifier)))
    if (target.endsWith('/routeTree.gen') || target.endsWith('/routeTree.gen.ts')) return undefined
    return target
  }
  return applyAlias(specifier, snapshot.tsAliases ?? {})
}

function globMatches(pattern: string, path: string): boolean {
  try {
    return new Bun.Glob(pattern).match(path)
  } catch {
    return false
  }
}

function metricValue(metric: string, content: string): number | undefined {
  if (metric === 'source_lines') {
    if (content.length === 0) return 0
    return content.split(/\r?\n/).length - (/\r?\n$/.test(content) ? 1 : 0)
  }
  if (metric === 'react_local_state_hooks') {
    return [...content.matchAll(/\buseState\s*(?:<[^;{}()]*>)?\s*\(/g)].length
  }
  return undefined
}

export function evaluateArchitectureWorkspace(
  doc: unknown,
  snapshot: ArchitectureWorkspaceSnapshot,
  options: ArchitectureWorkspaceOptions = {},
): ArchitectureWorkspaceFinding[] {
  const findings: ArchitectureWorkspaceFinding[] = []
  const modules = modulesFrom(doc)
  const files = snapshot.files.filter((file) => isProductionSource(file.path))
  const undeclaredImports = new Map<
    string,
    { source: ModuleRecord; target: ModuleRecord; file: ArchitectureSourceFile; specifier: string }
  >()
  const unmappedImportTargets = new Map<
    string,
    { file: ArchitectureSourceFile; specifier: string; targetPath: string }
  >()

  for (const file of files) {
    const source = moduleForPath(file.path, modules)
    if (!source) {
      findings.push({
        kind: 'import',
        path: file.path,
        message: `architecture: production source '${file.path}' belongs to no declared module`,
      })
      continue
    }
    const specifiers =
      extname(file.path) === '.go' ? goImports(file.content) : tsImports(file.content)
    for (const specifier of new Set(specifiers)) {
      const targetPath = importTarget(file, specifier, snapshot)
      if (!targetPath) continue
      const target = moduleForPath(targetPath, modules)
      if (!target) {
        const key = `${file.path}\0${targetPath}`
        if (!unmappedImportTargets.has(key)) {
          unmappedImportTargets.set(key, { file, specifier, targetPath })
        }
        continue
      }
      if (target.id === source.id || source.dependencies.has(target.id)) continue
      const key = `${source.id}\0${target.id}`
      if (!undeclaredImports.has(key)) {
        undeclaredImports.set(key, { source, target, file, specifier })
      }
    }
  }
  for (const { source, target, file, specifier } of undeclaredImports.values()) {
    findings.push({
      kind: 'import',
      path: file.path,
      message: `architecture: module '${source.id}' imports '${target.id}' via '${specifier}' without declaring it in depends_on`,
    })
  }
  for (const { file, specifier, targetPath } of unmappedImportTargets.values()) {
    findings.push({
      kind: 'import',
      path: file.path,
      message: `architecture: workspace-local import target '${targetPath}' via '${specifier}' belongs to no declared module`,
    })
  }

  const { budgets, debts } = complexityFrom(doc)
  const today = options.today ?? new Date().toISOString().slice(0, 10)
  const filesByPath = new Map(files.map((file) => [file.path, file]))
  for (const debt of debts) {
    const budget = budgets.find((item) => item.id === debt.budget)
    const file = filesByPath.get(debt.path)
    if (!budget) {
      findings.push({
        kind: 'complexity',
        path: debt.path,
        message: `architecture: complexity debt references unknown budget '${debt.budget}'`,
      })
      continue
    }
    if (!file) {
      findings.push({
        kind: 'complexity',
        path: debt.path,
        message: `architecture: complexity debt for budget '${debt.budget}' references no production source`,
      })
      continue
    }
    const value = metricValue(budget.metric, file.content)
    if (debt.ceiling === undefined) {
      findings.push({
        kind: 'complexity',
        path: file.path,
        message: `architecture: complexity debt for budget '${budget.id}' has no numeric ceiling`,
      })
    } else if (value !== undefined && value > debt.ceiling) {
      findings.push({
        kind: 'complexity',
        path: file.path,
        message: `architecture: complexity budget '${budget.id}' increased to ${value}, above debt ceiling ${debt.ceiling}`,
      })
    }
    if (!debt.expiresAt) {
      findings.push({
        kind: 'complexity',
        path: file.path,
        message: `architecture: complexity debt for budget '${budget.id}' has no expiry`,
      })
    } else if (debt.expiresAt < today) {
      findings.push({
        kind: 'complexity',
        path: file.path,
        message: `architecture: complexity debt for budget '${budget.id}' expired on ${debt.expiresAt}`,
      })
    }
  }
  for (const budget of budgets) {
    for (const file of files) {
      if (!budget.includes.some((pattern) => globMatches(pattern, file.path))) continue
      if (budget.excludes.some((pattern) => globMatches(pattern, file.path))) continue
      const value = metricValue(budget.metric, file.content)
      if (value === undefined || value <= budget.limit) continue
      const debt = debts.find((item) => item.budget === budget.id && item.path === file.path)
      if (!debt) {
        findings.push({
          kind: 'complexity',
          path: file.path,
          message: `architecture: complexity budget '${budget.id}' is ${value}, above limit ${budget.limit}, without declared debt`,
        })
      }
    }
  }

  return findings.sort(
    (left, right) =>
      left.path.localeCompare(right.path) ||
      left.kind.localeCompare(right.kind) ||
      left.message.localeCompare(right.message),
  )
}
