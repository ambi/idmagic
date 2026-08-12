/**
 * Resolve the references a work item makes into the specification and the
 * repository. A record that names a scenario, a TypeSpec symbol, or a reading
 * list is only useful while those targets still exist.
 *
 * `affected_spec` is checked for every record. `initial_context` is checked
 * only once the item is in progress: a reading list written for a backlog item
 * rots before the work begins, and the format asks for it to be rewritten at
 * that moment.
 */

export type WorkItemRecord = {
  status?: unknown
  affected_spec?: unknown
  initial_context?: unknown
}

export type ReferenceEnvironment = {
  /** Whether a repository-relative file or directory exists. */
  exists: (path: string) => boolean
  /** Contents of a repository-relative file, or undefined when unreadable. */
  read: (path: string) => string | undefined
}

/** Reading-list keys whose entries are repository paths. */
const PATH_KEYS = ['source', 'tests', 'stop_before_reading'] as const

function declaresScenario(source: string, id: string): boolean {
  return new RegExp(`^### ${id}: `, 'm').test(source)
}

function resolvesRequirement(source: string, requirement: string): boolean {
  // Standards keep their own identifiers (RFC7644-PATCH, SAML2B); only
  // normative scenarios are declared as headings.
  return requirement.startsWith('REQ-')
    ? declaresScenario(source, requirement)
    : source.includes(requirement)
}

function stringList(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === 'string')
    : []
}

function verifyAffectedSpec(record: WorkItemRecord, environment: ReferenceEnvironment): string[] {
  const findings: string[] = []
  const active = record.status === 'pending' || record.status === 'in_progress'
  for (const reference of stringOrObjectList(record.affected_spec)) {
    if (typeof reference.path !== 'string') {
      if (active) findings.push('active work item contains a legacy specification reference')
      continue
    }
    const source = environment.exists(reference.path) ? environment.read(reference.path) : undefined
    if (source === undefined) {
      findings.push(`affected_spec path does not exist: ${reference.path}`)
      continue
    }
    if (
      typeof reference.requirement === 'string' &&
      !resolvesRequirement(source, reference.requirement)
    ) {
      findings.push(`requirement does not resolve in ${reference.path}: ${reference.requirement}`)
    }
    if (typeof reference.symbol === 'string') {
      const name = reference.symbol.split('.').at(-1) ?? ''
      const declaration = new RegExp(`\\b(?:alias|enum|model|op|scalar|union)\\s+${name}\\b`)
      if (!declaration.test(source)) {
        findings.push(`TypeSpec symbol does not resolve in ${reference.path}: ${reference.symbol}`)
      }
    }
  }
  return findings
}

function stringOrObjectList(value: unknown): Array<Record<string, unknown>> {
  if (!Array.isArray(value)) return []
  return value.filter(
    (item): item is Record<string, unknown> =>
      typeof item === 'object' && item !== null && !Array.isArray(item),
  )
}

function verifyInitialContext(record: WorkItemRecord, environment: ReferenceEnvironment): string[] {
  const context = record.initial_context
  if (typeof context !== 'object' || context === null || Array.isArray(context)) return []
  const entries = context as Record<string, unknown>
  const findings: string[] = []

  for (const reference of stringList(entries.specification)) {
    const [path, requirement] = reference.split('#')
    const source = path && environment.exists(path) ? environment.read(path) : undefined
    if (!path || source === undefined) {
      findings.push(`initial_context specification path does not exist: ${reference}`)
      continue
    }
    if (requirement && !resolvesRequirement(source, requirement)) {
      findings.push(`initial_context specification does not resolve: ${reference}`)
    }
  }

  for (const key of PATH_KEYS) {
    for (const path of stringList(entries[key])) {
      if (!environment.exists(path)) findings.push(`initial_context ${key} does not exist: ${path}`)
    }
  }
  return findings
}

export function verifyWorkItemReferences(
  record: WorkItemRecord,
  environment: ReferenceEnvironment,
): string[] {
  const findings = verifyAffectedSpec(record, environment)
  if (record.status === 'in_progress') findings.push(...verifyInitialContext(record, environment))
  return findings
}
