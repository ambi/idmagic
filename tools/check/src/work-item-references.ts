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

import { specificationRules } from './feature-specification.ts'
import { parseScenarioDocument } from './gherkin-scenarios.ts'

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
  /** 移した仕様文書の行き先。`tools/check/relocated-spec-paths.json` が定める。 */
  relocated?: (path: string) => string[] | undefined
}

/** 移した仕様文書の旧パスから、規則を宣言する新しいパスへの対応。 */
export const RELOCATED_SPEC_PATHS = 'tools/check/relocated-spec-paths.json'

/**
 * 旧パスの行き先。ファイルを名指す項目を先に引き、なければ `/` で終わる項目を、
 * ディレクトリごと移した接頭辞の対応として使う。ディレクトリの改名で動いたファイルを
 * 一つずつ列挙すると、完了記録が参照するパスの数だけ表が増えるためである。
 */
export function relocatedSpecPaths(
  table: Readonly<Record<string, readonly string[]>>,
  path: string,
): string[] | undefined {
  const named = table[path]
  if (named) return [...named]
  const prefix = Object.keys(table)
    .filter((key) => key.endsWith('/') && path.startsWith(key))
    .sort((left, right) => right.length - left.length)[0]
  if (prefix === undefined) return undefined
  return (table[prefix] ?? []).map((target) => target + path.slice(prefix.length))
}

/** Reading-list keys whose entries are repository paths. */
const PATH_KEYS = ['source', 'tests', 'stop_before_reading'] as const

function declaresScenario(source: string, id: string): boolean {
  if (new RegExp(`^### ${id}: `, 'm').test(source)) return true
  if (specificationRules(source).some((rule) => rule.id === id)) return true
  return parseScenarioDocument(source).rules.some((rule) => rule.id === id)
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
      // 完了した記録は書き換えず、移した文書の行き先で解決する。未完了の記録は現在のパスへ直す。
      const relocated = active ? undefined : environment.relocated?.(reference.path)
      if (relocated === undefined) {
        findings.push(`affected_spec path does not exist: ${reference.path}`)
        continue
      }
      const sources = relocated.flatMap((path) => environment.read(path) ?? [])
      if (
        typeof reference.requirement === 'string' &&
        !sources.some((moved) => resolvesRequirement(moved, reference.requirement as string))
      ) {
        findings.push(
          `requirement does not resolve where ${reference.path} moved: ${reference.requirement}`,
        )
      }
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
