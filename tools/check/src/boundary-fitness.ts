export type ContextDefinition = {
  name: string
  packagePrefixes: string[]
}

export type ContextRelation = {
  supplier: string
  customer: string
  label: string
}

export type ArchitectureModel = {
  contexts: ContextDefinition[]
  relations: ContextRelation[]
}

export type GoSource = {
  path: string
  source: string
}

export type BoundaryViolationKind =
  | 'private-import'
  | 'undeclared-context-edge'
  | 'context-cycle'
  | 'domain-effect'
  | 'shared-detour'

export type BoundaryViolation = {
  id: string
  debtId: string
  kind: BoundaryViolationKind
  sourceContext: string
  targetContext?: string
  path: string
  message: string
}

export type BoundaryDebtEntry = {
  id: string
  sourceContext: string
  targetContext?: string
  violations: string[]
  reason: string
}

export type BoundaryDebtGroup = {
  id: string
  sourceContext: string
  targetContext?: string
  violationIds: string[]
  paths: string[]
}

export type BoundaryDebtFinding = {
  id: string
  kind: 'unrecorded-violation' | 'stale-debt' | 'invalid-debt'
  message: string
}

type ImportSpec = {
  path: string
  localName: string
}

type ContextDependency = {
  sourceContext: string
  targetContext: string
}

const REPOSITORY_CONTEXT_PREFIX = /^backend\/(?!cmd(?:\/|$)|shared(?:\/|$))[^/]+/

function normalizeRepositoryPath(path: string): string {
  return path.replace(/^\.\//, '').replace(/\/+$/, '')
}

function compareById(left: { id: string }, right: { id: string }): number {
  return left.id.localeCompare(right.id)
}

/** Derive the Context Map and Go package ownership from the canonical logical architecture. */
export function parseLogicalArchitecture(markdown: string): ArchitectureModel {
  const contexts: ContextDefinition[] = []
  const contextRow = /^\|\s*\[([^\]]+)\]\([^)]+\)\s*\|\s*[^|]*\|\s*([^|]+)\|\s*[^|]*\|\s*$/gm
  for (const match of markdown.matchAll(contextRow)) {
    const name = match[1]?.trim()
    const packageCell = match[2] ?? ''
    if (!name) continue
    const packagePrefixes = [...packageCell.matchAll(/`([^`]+)`/g)]
      .map((pathMatch) => normalizeRepositoryPath(pathMatch[1] ?? ''))
      .filter((path) => REPOSITORY_CONTEXT_PREFIX.test(path))
      .sort()
    if (name !== 'System' && packagePrefixes.length === 0) {
      throw new Error(`${name}: Context responsibility table must declare a backend package`)
    }
    contexts.push({ name, packagePrefixes })
  }
  contexts.sort((left, right) => left.name.localeCompare(right.name))
  if (contexts.length === 0) {
    throw new Error('logical architecture must contain a Context responsibility table')
  }

  const knownContexts = new Set(contexts.map((context) => context.name))
  if (knownContexts.size !== contexts.length) {
    throw new Error('Context responsibility table must not repeat a Context name')
  }
  const contextMapSection = markdown.slice(markdown.search(/^## Context Map\s*$/m))
  const contextMap = contextMapSection.match(/```mermaid\r?\n([\s\S]*?)\r?\n```/)?.[1]
  if (!contextMap) throw new Error('logical architecture must contain a Mermaid Context Map')
  const relations: ContextRelation[] = []
  const mermaidEdge = /^\s*([A-Za-z][A-Za-z0-9]*)\s*-->\|([^|]+)\|\s*([A-Za-z][A-Za-z0-9]*)\s*$/
  for (const line of contextMap.split('\n')) {
    if (!line.includes('-->')) continue
    const match = line.match(mermaidEdge)
    if (!match) {
      throw new Error(
        `Context Map relation must use Supplier -->|relation| Customer: ${line.trim()}`,
      )
    }
    const supplier = match[1] ?? ''
    const label = match[2]?.trim() ?? ''
    const customer = match[3] ?? ''
    if (!knownContexts.has(supplier) || !knownContexts.has(customer)) {
      throw new Error(`Context Map relation names an unknown Context: ${supplier} -> ${customer}`)
    }
    relations.push({ supplier, customer, label })
  }
  relations.sort((left, right) =>
    `${left.supplier}\0${left.customer}\0${left.label}`.localeCompare(
      `${right.supplier}\0${right.customer}\0${right.label}`,
    ),
  )
  return { contexts, relations }
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

function ownerOf(
  path: string,
  contexts: readonly ContextDefinition[],
): ContextDefinition | undefined {
  return contexts
    .flatMap((context) => context.packagePrefixes.map((prefix) => ({ context, prefix })))
    .filter(({ prefix }) => path === prefix || path.startsWith(`${prefix}/`))
    .sort((left, right) => right.prefix.length - left.prefix.length)[0]?.context
}

function publicPackage(path: string, context: ContextDefinition): boolean {
  const prefix = context.packagePrefixes
    .filter((candidate) => path === candidate || path.startsWith(`${candidate}/`))
    .sort((left, right) => right.length - left.length)[0]
  if (!prefix) return false
  return path
    .slice(prefix.length)
    .split('/')
    .filter(Boolean)
    .some((segment) => segment === 'domain' || segment === 'ports')
}

function declaredDependency(
  sourceContext: string,
  targetContext: string,
  architecture: ArchitectureModel,
): boolean {
  return architecture.relations.some(
    (relation) => relation.customer === sourceContext && relation.supplier === targetContext,
  )
}

function directDependencyViolations(
  sourcePackage: string,
  sourceContext: ContextDefinition,
  targetPackage: string,
  targetContext: ContextDefinition,
  architecture: ArchitectureModel,
): BoundaryViolation[] {
  const violations: BoundaryViolation[] = []
  if (!publicPackage(targetPackage, targetContext)) {
    violations.push({
      id: `private-import:${sourceContext.name}:${sourcePackage}->${targetPackage}`,
      debtId: `private-import:${sourceContext.name}->${targetContext.name}`,
      kind: 'private-import',
      sourceContext: sourceContext.name,
      targetContext: targetContext.name,
      path: `${sourcePackage} -> ${targetPackage}`,
      message: `${sourceContext.name} imports private package ${targetPackage} from ${targetContext.name}`,
    })
  }
  if (!declaredDependency(sourceContext.name, targetContext.name, architecture)) {
    violations.push({
      id:
        `undeclared-context-edge:${sourceContext.name}->${targetContext.name}:` +
        `${sourcePackage}->${targetPackage}`,
      debtId: `undeclared-context-edge:${sourceContext.name}->${targetContext.name}`,
      kind: 'undeclared-context-edge',
      sourceContext: sourceContext.name,
      targetContext: targetContext.name,
      path: `${sourcePackage} -> ${targetPackage}`,
      message:
        `${sourceContext.name} depends on ${targetContext.name}, but the Context Map has no ` +
        `${targetContext.name} supplier -> ${sourceContext.name} customer relation`,
    })
  }
  return violations
}

function domainEffectViolations(
  source: GoSource,
  imports: readonly ImportSpec[],
): BoundaryViolation[] {
  if (!normalizeRepositoryPath(source.path).split('/').includes('domain')) return []
  const sourceContext = normalizeRepositoryPath(source.path).split('/')[1] ?? 'unknown'
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
      debtId: `domain-effect:${sourceContext}`,
      kind: 'domain-effect',
      sourceContext,
      path: source.path,
      message: `${source.path}: domain packages must receive ${effect} effects through an input or port`,
    }),
  )
}

function edgeKey(edge: ContextDependency): string {
  return `${edge.sourceContext}\0${edge.targetContext}`
}

function cycleViolations(edges: readonly ContextDependency[]): BoundaryViolation[] {
  const uniqueEdges = new Map(edges.map((edge) => [edgeKey(edge), edge]))
  const adjacency = new Map<string, Set<string>>()
  for (const edge of uniqueEdges.values()) {
    const targets = adjacency.get(edge.sourceContext) ?? new Set<string>()
    targets.add(edge.targetContext)
    adjacency.set(edge.sourceContext, targets)
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
    .filter((edge) => reaches(edge.targetContext, edge.sourceContext))
    .map(
      (edge): BoundaryViolation => ({
        id: `context-cycle:${edge.sourceContext}->${edge.targetContext}`,
        debtId: `context-cycle:${edge.sourceContext}->${edge.targetContext}`,
        kind: 'context-cycle',
        sourceContext: edge.sourceContext,
        targetContext: edge.targetContext,
        path: `${edge.sourceContext} -> ${edge.targetContext}`,
        message: `${edge.sourceContext} -> ${edge.targetContext} participates in a Context dependency cycle`,
      }),
    )
    .sort(compareById)
}

export function findBoundaryViolations(input: {
  modulePath: string
  architecture: ArchitectureModel
  sources: readonly GoSource[]
}): BoundaryViolation[] {
  const importsByPackage = new Map<string, Set<string>>()
  const importsBySource = new Map<string, ImportSpec[]>()
  for (const source of input.sources) {
    const imports = goImports(source.source)
    importsBySource.set(source.path, imports)
    const packagePath = packageForFile(source.path)
    const packageImports = importsByPackage.get(packagePath) ?? new Set<string>()
    for (const imported of imports) {
      const path = repositoryImport(input.modulePath, imported.path)
      if (path) packageImports.add(path)
    }
    importsByPackage.set(packagePath, packageImports)
  }

  const violations: BoundaryViolation[] = []
  const dependencies: ContextDependency[] = []
  const sharedStarts = new Map<string, Set<string>>()
  for (const source of input.sources) {
    const sourcePackage = packageForFile(source.path)
    const sourceContext = ownerOf(sourcePackage, input.architecture.contexts)
    violations.push(...domainEffectViolations(source, importsBySource.get(source.path) ?? []))
    if (!sourceContext) continue
    for (const imported of importsBySource.get(source.path) ?? []) {
      const targetPackage = repositoryImport(input.modulePath, imported.path)
      if (!targetPackage) continue
      if (targetPackage.startsWith('backend/shared/')) {
        const starts = sharedStarts.get(sourceContext.name) ?? new Set<string>()
        starts.add(targetPackage)
        sharedStarts.set(sourceContext.name, starts)
        continue
      }
      const targetContext = ownerOf(targetPackage, input.architecture.contexts)
      if (!targetContext || targetContext.name === sourceContext.name) continue
      dependencies.push({ sourceContext: sourceContext.name, targetContext: targetContext.name })
      violations.push(
        ...directDependencyViolations(
          sourcePackage,
          sourceContext,
          targetPackage,
          targetContext,
          input.architecture,
        ),
      )
    }
  }

  for (const [sourceContextName, starts] of sharedStarts) {
    const sourceContext = input.architecture.contexts.find(
      (context) => context.name === sourceContextName,
    )!
    for (const firstSharedPackage of starts) {
      const pending = [firstSharedPackage]
      const seen = new Set<string>()
      while (pending.length > 0) {
        const sharedPackage = pending.pop()!
        if (seen.has(sharedPackage)) continue
        seen.add(sharedPackage)
        for (const targetPackage of importsByPackage.get(sharedPackage) ?? []) {
          if (targetPackage.startsWith('backend/shared/')) {
            pending.push(targetPackage)
            continue
          }
          const targetContext = ownerOf(targetPackage, input.architecture.contexts)
          if (!targetContext || targetContext.name === sourceContext.name) continue
          dependencies.push({
            sourceContext: sourceContext.name,
            targetContext: targetContext.name,
          })
          const forbidden =
            !publicPackage(targetPackage, targetContext) ||
            !declaredDependency(sourceContext.name, targetContext.name, input.architecture)
          if (!forbidden) continue
          violations.push({
            id:
              `shared-detour:${sourceContext.name}:${firstSharedPackage}->` +
              `${targetContext.name}:${targetPackage}`,
            debtId: `shared-detour:${sourceContext.name}->${targetContext.name}`,
            kind: 'shared-detour',
            sourceContext: sourceContext.name,
            targetContext: targetContext.name,
            path: `${sourceContext.name} -> ${firstSharedPackage} -> ${targetPackage}`,
            message:
              `${sourceContext.name} reaches forbidden ${targetContext.name} package ${targetPackage} ` +
              `through ${firstSharedPackage}`,
          })
        }
      }
    }
  }

  violations.push(...cycleViolations(dependencies))
  return [...new Map(violations.map((violation) => [violation.id, violation])).values()].sort(
    compareById,
  )
}

export function groupBoundaryViolations(
  violations: readonly BoundaryViolation[],
): BoundaryDebtGroup[] {
  const groups = new Map<string, BoundaryDebtGroup>()
  for (const violation of violations) {
    const group = groups.get(violation.debtId) ?? {
      id: violation.debtId,
      sourceContext: violation.sourceContext,
      ...(violation.targetContext ? { targetContext: violation.targetContext } : {}),
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
      typeof entry.sourceContext !== 'string' ||
      typeof entry.reason !== 'string' ||
      !Array.isArray(entry.violations) ||
      entry.violations.some((violation) => typeof violation !== 'string') ||
      (entry.targetContext !== undefined && typeof entry.targetContext !== 'string')
    ) {
      throw new Error(
        'tools/check/boundary-debt.json: every entry needs id, sourceContext, optional targetContext, violations, and reason',
      )
    }
    const violationIds = [...new Set(entry.violations as string[])].sort()
    if (violationIds.length === 0 || violationIds.length !== entry.violations.length) {
      throw new Error(`${entry.id}: violations must be a non-empty array of unique ids`)
    }
    if (
      entry.reason.trim().length < 12 ||
      /^(?:existing|legacy|\u65e2存)(?:\s+violation|のため)?[.\u3002]?$/i.test(entry.reason.trim())
    ) {
      throw new Error(`${entry.id}: boundary debt reason must explain the concrete coupling`)
    }
    if (ids.has(entry.id)) throw new Error(`${entry.id}: duplicate boundary debt id`)
    ids.add(entry.id)
    entries.push({
      id: entry.id,
      sourceContext: entry.sourceContext,
      targetContext: entry.targetContext as string | undefined,
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
    if (
      entry.sourceContext !== group.sourceContext ||
      entry.targetContext !== group.targetContext
    ) {
      findings.push({
        id: group.id,
        kind: 'invalid-debt',
        message: `${group.id}: debt Context fields do not match the observed dependency`,
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
        message: `${id}: boundary debt is resolved; remove it from ${group.id}`,
      })
    }
  }
  for (const entry of debt) {
    if (groupById.has(entry.id)) continue
    findings.push({
      id: entry.id,
      kind: 'stale-debt',
      message: `${entry.id}: boundary debt is resolved; remove the stale entry`,
    })
  }
  return findings.sort(compareById)
}

export function addedBoundaryDebtViolations(
  before: readonly BoundaryDebtEntry[],
  current: readonly BoundaryDebtEntry[],
): string[] {
  const beforeIds = new Set(before.flatMap((entry) => entry.violations))
  return current
    .flatMap((entry) => entry.violations)
    .filter((id) => !beforeIds.has(id))
    .sort()
}
