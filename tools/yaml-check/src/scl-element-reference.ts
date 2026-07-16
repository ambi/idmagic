/**
 * Stable, context-qualified references to normative SCL 3.0 elements.
 *
 * References are authored outside SCL and resolved through the workspace
 * context map. This module is intentionally pure so traceability tooling and
 * renderers can share the same identity rules.
 */

type Dict = Record<string, unknown>

export const SCL_ELEMENT_KINDS = [
  'standard_requirement',
  'model',
  'interface',
  'state',
  'authorization_resource',
  'authorization_principal',
  'authorization_policy',
  'objective',
  'scenario',
  'flow',
] as const

export type SclElementKind = (typeof SCL_ELEMENT_KINDS)[number]
export type SclNamedElementKind = Exclude<SclElementKind, 'standard_requirement'>

export type SclNamedElementReference = {
  context: string
  kind: SclNamedElementKind
  element: string
}

export type SclStandardRequirementReference = {
  context: string
  kind: 'standard_requirement'
  standard: string
  requirement: string
}

export type SclElementReference =
  | SclNamedElementReference
  | SclStandardRequirementReference

export type SclReferenceErrorCode =
  | 'invalid_reference'
  | 'unknown_kind'
  | 'unknown_context'
  | 'unavailable_context_spec'
  | 'context_mismatch'
  | 'unknown_element'
  | 'unknown_standard'
  | 'unknown_requirement'
  | 'duplicate_requirement'

export type SclReferenceError = {
  code: SclReferenceErrorCode
  message: string
}

export type NormalizeSclElementReferenceResult =
  | { ok: true; reference: SclElementReference }
  | { ok: false; error: SclReferenceError }

export type SclWorkspaceIndex = {
  root: Dict
  contexts: ReadonlyMap<string, Dict>
}

export type BuildSclWorkspaceIndexResult =
  | { ok: true; index: SclWorkspaceIndex }
  | { ok: false; errors: SclReferenceError[] }

export type ResolveSclElementReferenceResult =
  | {
      ok: true
      reference: SclElementReference
      canonical: string
      target: unknown
    }
  | { ok: false; error: SclReferenceError }

const NAMED_KIND_PATHS: Record<SclNamedElementKind, readonly string[]> = {
  model: ['models'],
  interface: ['interfaces'],
  state: ['states'],
  authorization_resource: ['authorization', 'resources'],
  authorization_principal: ['authorization', 'principals'],
  authorization_policy: ['authorization', 'policies'],
  objective: ['objectives'],
  scenario: ['scenarios'],
  flow: ['flows'],
}

function dict(value: unknown): Dict {
  return value !== null && typeof value === 'object' && !Array.isArray(value)
    ? (value as Dict)
    : {}
}

function nonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0
}

function error(code: SclReferenceErrorCode, message: string): SclReferenceError {
  return { code, message }
}

export function normalizeSclElementReference(
  value: unknown,
): NormalizeSclElementReferenceResult {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    return {
      ok: false,
      error: error('invalid_reference', 'SCL element reference must be an object'),
    }
  }

  const input = value as Dict
  if (!nonEmptyString(input.context)) {
    return {
      ok: false,
      error: error('invalid_reference', "SCL element reference requires non-empty 'context'"),
    }
  }
  if (!nonEmptyString(input.kind)) {
    return {
      ok: false,
      error: error('invalid_reference', "SCL element reference requires non-empty 'kind'"),
    }
  }
  if (!(SCL_ELEMENT_KINDS as readonly string[]).includes(input.kind)) {
    return {
      ok: false,
      error: error('unknown_kind', `unknown SCL element kind '${input.kind}'`),
    }
  }

  if (input.kind === 'standard_requirement') {
    const allowed = new Set(['context', 'kind', 'standard', 'requirement'])
    const extra = Object.keys(input).find((key) => !allowed.has(key))
    if (extra) {
      return {
        ok: false,
        error: error(
          'invalid_reference',
          `standard requirement reference contains unknown field '${extra}'`,
        ),
      }
    }
    if (!nonEmptyString(input.standard) || !nonEmptyString(input.requirement)) {
      return {
        ok: false,
        error: error(
          'invalid_reference',
          "standard requirement reference requires non-empty 'standard' and 'requirement'",
        ),
      }
    }
    return {
      ok: true,
      reference: {
        context: input.context,
        kind: 'standard_requirement',
        standard: input.standard,
        requirement: input.requirement,
      },
    }
  }

  const allowed = new Set(['context', 'kind', 'element'])
  const extra = Object.keys(input).find((key) => !allowed.has(key))
  if (extra) {
    return {
      ok: false,
      error: error('invalid_reference', `SCL element reference contains unknown field '${extra}'`),
    }
  }
  if (!nonEmptyString(input.element)) {
    return {
      ok: false,
      error: error('invalid_reference', "SCL element reference requires non-empty 'element'"),
    }
  }
  return {
    ok: true,
    reference: {
      context: input.context,
      kind: input.kind as SclNamedElementKind,
      element: input.element,
    },
  }
}

export function buildSclWorkspaceIndex(
  root: unknown,
  contextDocuments: ReadonlyMap<string, unknown> | Readonly<Record<string, unknown>>,
): BuildSclWorkspaceIndexResult {
  const rootDoc = dict(root)
  const contextMap = dict(rootDoc.context_map)
  const supplied =
    contextDocuments instanceof Map
      ? contextDocuments
      : new Map(Object.entries(contextDocuments as Readonly<Record<string, unknown>>))
  const contexts = new Map<string, Dict>()
  const errors: SclReferenceError[] = []

  for (const context of Object.keys(contextMap)) {
    if (!supplied.has(context)) {
      errors.push(
        error(
          'unavailable_context_spec',
          `context '${context}' is declared by context_map but its SCL document is unavailable`,
        ),
      )
      continue
    }
    const document = dict(supplied.get(context))
    if (document.context !== context) {
      errors.push(
        error(
          'context_mismatch',
          `context_map key '${context}' does not match document context '${String(document.context ?? '')}'`,
        ),
      )
      continue
    }
    contexts.set(context, document)

    for (const [standardName, standardValue] of Object.entries(dict(document.standards))) {
      const requirements = Array.isArray(dict(standardValue).requirements)
        ? (dict(standardValue).requirements as unknown[])
        : []
      const ids = new Set<string>()
      for (const requirementValue of requirements) {
        const id = dict(requirementValue).id
        if (!nonEmptyString(id)) continue
        if (ids.has(id)) {
          errors.push(
            error(
              'duplicate_requirement',
              `context '${context}' standard '${standardName}' duplicates requirement id '${id}'`,
            ),
          )
        }
        ids.add(id)
      }
    }
  }

  for (const context of supplied.keys()) {
    if (!(context in contextMap)) {
      errors.push(error('unknown_context', `context document '${context}' is absent from context_map`))
    }
  }

  return errors.length > 0
    ? { ok: false, errors }
    : { ok: true, index: { root: rootDoc, contexts } }
}

function nestedValue(document: Dict, path: readonly string[]): unknown {
  let value: unknown = document
  for (const segment of path) value = dict(value)[segment]
  return value
}

export function resolveSclElementReference(
  index: SclWorkspaceIndex,
  value: unknown,
): ResolveSclElementReferenceResult {
  const normalized = normalizeSclElementReference(value)
  if (!normalized.ok) return normalized
  const reference = normalized.reference
  const document = index.contexts.get(reference.context)
  if (!document) {
    return {
      ok: false,
      error: error('unknown_context', `unknown SCL context '${reference.context}'`),
    }
  }

  if (reference.kind === 'standard_requirement') {
    const standard = dict(document.standards)[reference.standard]
    if (standard === undefined) {
      return {
        ok: false,
        error: error(
          'unknown_standard',
          `context '${reference.context}' has no standard '${reference.standard}'`,
        ),
      }
    }
    const requirements = Array.isArray(dict(standard).requirements)
      ? (dict(standard).requirements as unknown[])
      : []
    const matches = requirements.filter(
      (requirement) => dict(requirement).id === reference.requirement,
    )
    if (matches.length === 0) {
      return {
        ok: false,
        error: error(
          'unknown_requirement',
          `standard '${reference.standard}' has no requirement '${reference.requirement}'`,
        ),
      }
    }
    if (matches.length > 1) {
      return {
        ok: false,
        error: error(
          'duplicate_requirement',
          `standard '${reference.standard}' has duplicate requirement '${reference.requirement}'`,
        ),
      }
    }
    return {
      ok: true,
      reference,
      canonical: canonicalSclElementReference(reference),
      target: matches[0],
    }
  }

  const members = dict(nestedValue(document, NAMED_KIND_PATHS[reference.kind]))
  if (!(reference.element in members)) {
    return {
      ok: false,
      error: error(
        'unknown_element',
        `context '${reference.context}' has no ${reference.kind} '${reference.element}'`,
      ),
    }
  }
  return {
    ok: true,
    reference,
    canonical: canonicalSclElementReference(reference),
    target: members[reference.element],
  }
}

function referenceSegments(reference: SclElementReference): string[] {
  return reference.kind === 'standard_requirement'
    ? [reference.context, reference.kind, reference.standard, reference.requirement]
    : [reference.context, reference.kind, reference.element]
}

function escapeCanonicalSegment(value: string): string {
  return value.replaceAll('~', '~0').replaceAll('/', '~1')
}

export function canonicalSclElementReference(reference: SclElementReference): string {
  return referenceSegments(reference).map(escapeCanonicalSegment).join('/')
}

function escapeAnchorSegment(value: string): string {
  let escaped = ''
  for (const byte of new TextEncoder().encode(value)) {
    const character = String.fromCharCode(byte)
    escaped += /[A-Za-z0-9_.-]/.test(character)
      ? character
      : `~${byte.toString(16).toUpperCase().padStart(2, '0')}`
  }
  return escaped
}

export function sclElementAnchor(reference: SclElementReference): string {
  return `scl-${referenceSegments(reference).map(escapeAnchorSegment).join('/')}`
}

export function sclDocumentContext(document: { context?: string; system: string }): string {
  return document.context ?? document.system
}
