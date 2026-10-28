/**
 * Resolve the references a work item makes into the specification and the
 * repository. A record that names a scenario or a TypeSpec symbol is only
 * useful while those targets still exist.
 *
 * `affected_spec` is checked for every record the caller verifies, and for a
 * completing record by `move-work-item` before it is collapsed.
 */

import { specificationRules } from './feature-specification.ts'
import { parseScenarioDocument } from './gherkin-scenarios.ts'
import type { SpecificationDiff } from './spec-diff.ts'

/** 基準からの仕様差分が取り除いた要素を、差分の綴りで返す。 */
export function removedSpecificationElements(diff: SpecificationDiff): string[] {
  return [...diff.removedScenarios, ...diff.removedStandards, ...diff.removedDeclarations]
}

function nonEmpty(value: unknown): value is string {
  return typeof value === 'string' && value.trim().length > 0
}

/**
 * Whether one reference names one diff element. A scenario row is spelled
 * `<path>#<id>` and a TypeSpec declaration `<path>:<name>`; both halves are
 * required, so a record cannot claim a same-named element of another document.
 */
function referenceNames(reference: Record<string, unknown>, element: string): boolean {
  for (const field of ['requirement', 'symbol']) {
    if (nonEmpty(reference[field]) && reference[field].trim() === element) return true
  }
  const row = element.lastIndexOf('#')
  if (row >= 0) {
    if (!nonEmpty(reference.path) || !nonEmpty(reference.requirement)) return false
    return (
      reference.path.trim() === element.slice(0, row) &&
      reference.requirement.trim() === element.slice(row + 1)
    )
  }
  const separator = element.lastIndexOf(':')
  if (separator < 0) return false
  if (!nonEmpty(reference.path) || !nonEmpty(reference.symbol)) return false
  return (
    reference.path.trim() === element.slice(0, separator) &&
    reference.symbol.trim().split('.').at(-1) === element.slice(separator + 1)
  )
}

function namesSpecificationElement(
  reference: Record<string, unknown>,
  elements: Iterable<string>,
): boolean {
  for (const element of elements) {
    if (referenceNames(reference, element)) return true
  }
  return false
}

export type WorkItemRecord = {
  status?: unknown
  affected_spec?: unknown
}

export type ReferenceEnvironment = {
  /** Whether a repository-relative file or directory exists. */
  exists: (path: string) => boolean
  /** Contents of a repository-relative file, or undefined when unreadable. */
  read: (path: string) => string | undefined
  /**
   * Elements the specification diff against the baseline removed, spelled as
   * the diff spells them. A record that removes or renames an element names
   * the old one, which no longer exists in the working tree; such a reference
   * is accepted while the diff still carries the removal.
   */
  removedSpecification?: readonly string[]
}

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

function verifyAffectedSpec(record: WorkItemRecord, environment: ReferenceEnvironment): string[] {
  const findings: string[] = []
  for (const reference of stringOrObjectList(record.affected_spec)) {
    if (typeof reference.path !== 'string') continue
    if (namesSpecificationElement(reference, environment.removedSpecification ?? [])) continue
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

export function verifyWorkItemReferences(
  record: WorkItemRecord,
  environment: ReferenceEnvironment,
): string[] {
  return verifyAffectedSpec(record, environment)
}
