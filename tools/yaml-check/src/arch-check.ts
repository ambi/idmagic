/**
 * Parsing and semantic cross-checks for ARCHITECTURE.md.
 *
 * The JSON Schema owns the authored shape. `verifyArchitecture` checks the
 * relationships which require workspace knowledge and deliberately receives
 * every filesystem/SCL dependency through options so it remains pure.
 */

import { type Finding, locatePointer } from './lib.ts'
import {
  normalizeSclElementReference,
  resolveSclElementReference,
  type SclWorkspaceIndex,
} from './scl-element-reference.ts'

/** SCL sections retained for callers which still collect legacy string refs. */
const SCL_ELEMENT_SECTIONS = [
  'interfaces',
  'models',
  'states',
  'authorization',
  'scenarios',
  'objectives',
  'flows',
  'glossary',
] as const

const BODY_HEADINGS: Record<string, string> = {
  overview: 'overview',
  structure: 'structure',
  stack: 'stack',
  'structural decisions': 'structural_decisions',
  'cross-cutting concerns': 'cross_cutting_concerns',
  diagrams: 'diagrams',
}

const LAYERS = [
  'specification_core',
  'decision_record',
  'domain',
  'use_cases',
  'adapters',
  'infrastructure',
  'deploy_pipeline',
] as const

const ROLES = [
  'implementation',
  'published_interface',
  'binding',
  'technical_shared',
  'composition_root',
] as const

const CROSS_CONTEXT_VIA = [
  'published_interface',
  'binding',
  'technical_shared',
  'composition_root',
] as const

type Dict = Record<string, unknown>
type Layer = (typeof LAYERS)[number]
type Role = (typeof ROLES)[number]
type CrossContextVia = (typeof CROSS_CONTEXT_VIA)[number]

function dict(value: unknown): Dict {
  return value !== null && typeof value === 'object' && !Array.isArray(value) ? (value as Dict) : {}
}

function nonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0
}

/** Parse nested YAML frontmatter and the known Markdown body sections. */
export function parseArchitectureDoc(text: string): Record<string, unknown> {
  const match = text.match(/^---\s*\r?\n([\s\S]*?)\r?\n---\s*\r?\n([\s\S]*)$/)
  let frontmatter: Record<string, unknown> = {}
  let body = text
  if (match?.[1] !== undefined && match[2] !== undefined) {
    const parsed = Bun.YAML.parse(match[1])
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      frontmatter = parsed as Record<string, unknown>
    }
    body = match[2]
  }

  const sections: Record<string, unknown> = {}
  for (const sec of body.split(/(?=^#{1,2}\s+)/m)) {
    const lines = sec.split('\n')
    const header = (lines[0] ?? '')
      .match(/^#{1,2}\s+(.+)$/)?.[1]
      ?.trim()
      .toLowerCase()
    if (!header) continue
    const key = BODY_HEADINGS[header]
    if (!key) continue
    const content = lines.slice(1).join('\n').trim()
    if (content) sections[key] = content
  }

  return { ...sections, ...frontmatter }
}

/** Collect legacy `section.Name` refs from one parsed SCL document. */
export function collectSclElements(sclDoc: unknown): string[] {
  const doc = sclDoc as Record<string, unknown> | null | undefined
  if (!doc || typeof doc !== 'object') return []
  const refs: string[] = []
  for (const section of SCL_ELEMENT_SECTIONS) {
    const members = doc[section]
    if (members && typeof members === 'object' && !Array.isArray(members)) {
      if (section === 'authorization') {
        for (const group of Object.values(members as Record<string, unknown>)) {
          if (group && typeof group === 'object' && !Array.isArray(group)) {
            for (const name of Object.keys(group as Record<string, unknown>)) {
              refs.push(`authorization.${name}`)
            }
          }
        }
        continue
      }
      for (const name of Object.keys(members as Record<string, unknown>)) {
        refs.push(`${section}.${name}`)
      }
    }
  }
  return refs
}

export type ArchReport = { errors: Finding[]; warnings: Finding[] }

export type ArchCheckOptions = {
  text?: string
  archDir: string
  workspaceRoot: string
  /** Direct-reference resolver index for `modules[].realizes`. */
  sclIndex?: SclWorkspaceIndex
  /** Exact SCL context -> workspace-relative spec path expected in the map. */
  expectedContexts?: ReadonlyMap<string, string> | Readonly<Record<string, string>>
  /** Temporary compatibility for pre-direct-reference Architecture maps. */
  sclElements?: ReadonlySet<string>
  pathExists: (absPath: string) => boolean
  join: (...parts: string[]) => string
}

function expectedContextMap(
  value: ArchCheckOptions['expectedContexts'],
): ReadonlyMap<string, string> | undefined {
  if (!value) return undefined
  return value instanceof Map ? value : new Map(Object.entries(value))
}

function crossContextEdgeAllowed(sourceRole: Role, targetRole: Role, via: CrossContextVia) {
  if (via === 'published_interface' || via === 'technical_shared') return targetRole === via
  return sourceRole === via
}

/** Cross-check an Architecture map against its SCL index and workspace. */
export function verifyArchitecture(doc: unknown, opts: ArchCheckOptions): ArchReport {
  const errors: Finding[] = []
  const warnings: Finding[] = []
  const text = opts.text ?? ''
  const data = dict(doc)
  const contexts = dict(data.contexts)
  const modules = dict(data.modules)
  const runtimes = dict(data.runtime_units)

  const addError = (pointer: string, message: string) => {
    errors.push({
      line: locatePointer(text, pointer),
      column: 1,
      message: `architecture: ${message}`,
    })
  }

  // The context ledger is an exact projection of the SCL workspace context map.
  const expected = expectedContextMap(opts.expectedContexts)
  if (expected) {
    for (const [context, spec] of expected) {
      const actual = contexts[context]
      if (actual === undefined) {
        addError(`/contexts/${context}`, `SCL context '${context}' is missing from contexts`)
        continue
      }
      const actualSpec = dict(actual).spec
      if (actualSpec !== spec) {
        addError(
          `/contexts/${context}/spec`,
          `context '${context}' spec must be '${spec}', got '${String(actualSpec ?? '')}'`,
        )
      }
    }
    for (const context of Object.keys(contexts)) {
      if (!expected.has(context)) {
        addError(
          `/contexts/${context}`,
          `context '${context}' is not declared by the SCL context map`,
        )
      }
    }
  }
  for (const [context, value] of Object.entries(contexts)) {
    const spec = dict(value).spec
    if (!nonEmptyString(spec)) {
      addError(`/contexts/${context}/spec`, `context '${context}' requires a non-empty spec`)
    } else if (!opts.pathExists(opts.join(opts.archDir, spec))) {
      addError(`/contexts/${context}/spec`, `context '${context}' spec '${spec}' does not exist`)
    }
  }

  type ModuleInfo = { context: string; layer: Layer; role: Role }
  const moduleInfo = new Map<string, ModuleInfo>()
  for (const [id, value] of Object.entries(modules)) {
    const mod = dict(value)
    const path = mod.path
    const context = mod.context
    const layer = mod.layer
    const role = mod.role

    if (!nonEmptyString(path)) {
      addError(`/modules/${id}/path`, `module '${id}' requires a non-empty path`)
    } else if (!opts.pathExists(opts.join(opts.archDir, path))) {
      addError(`/modules/${id}/path`, `module '${id}' path '${path}' does not exist`)
    }
    if (!nonEmptyString(context) || !(context in contexts)) {
      addError(
        `/modules/${id}/context`,
        `module '${id}' context '${String(context ?? '')}' is not declared`,
      )
    }
    if (!nonEmptyString(layer) || !(LAYERS as readonly string[]).includes(layer)) {
      addError(`/modules/${id}/layer`, `module '${id}' has invalid layer '${String(layer ?? '')}'`)
    }
    if (!nonEmptyString(role) || !(ROLES as readonly string[]).includes(role)) {
      addError(`/modules/${id}/role`, `module '${id}' has invalid role '${String(role ?? '')}'`)
    }
    if (
      nonEmptyString(context) &&
      context in contexts &&
      nonEmptyString(layer) &&
      (LAYERS as readonly string[]).includes(layer) &&
      nonEmptyString(role) &&
      (ROLES as readonly string[]).includes(role)
    ) {
      moduleInfo.set(id, { context, layer: layer as Layer, role: role as Role })
    }

    if (Array.isArray(mod.realizes)) {
      mod.realizes.forEach((reference, index) => {
        const pointer = `/modules/${id}/realizes/${index}`
        if (typeof reference === 'string' && opts.sclElements) {
          if (!opts.sclElements.has(reference)) {
            addError(pointer, `module '${id}' realizes '${reference}' which no SCL element defines`)
          }
          return
        }
        const normalized = normalizeSclElementReference(reference)
        if (!normalized.ok) {
          addError(
            pointer,
            `module '${id}' has invalid realizes reference: ${normalized.error.message}`,
          )
          return
        }
        if (normalized.reference.context !== context) {
          addError(
            pointer,
            `module '${id}' in context '${String(context ?? '')}' cannot realize context '${normalized.reference.context}'`,
          )
        }
        if (!opts.sclIndex) {
          addError(
            pointer,
            `module '${id}' realizes reference cannot be resolved without an SCL index`,
          )
          return
        }
        const resolved = resolveSclElementReference(opts.sclIndex, reference)
        if (!resolved.ok) {
          addError(
            pointer,
            `module '${id}' realizes reference does not resolve: ${resolved.error.message}`,
          )
        }
      })
    }
  }

  // Declared module dependency graph, layer direction and context boundaries.
  const graph = new Map<string, string[]>()
  for (const id of Object.keys(modules)) graph.set(id, [])
  for (const [id, value] of Object.entries(modules)) {
    const source = moduleInfo.get(id)
    const dependencies = dict(value).depends_on
    if (!Array.isArray(dependencies)) continue
    dependencies.forEach((dependency, index) => {
      const edge = dict(dependency)
      const targetId = edge.module
      const via = edge.via
      const pointer = `/modules/${id}/depends_on/${index}`
      if (!nonEmptyString(targetId) || !(targetId in modules)) {
        addError(pointer, `module '${id}' depends on unknown module '${String(targetId ?? '')}'`)
        return
      }
      graph.get(id)?.push(targetId)
      const target = moduleInfo.get(targetId)
      if (!source || !target) return
      if (LAYERS.indexOf(source.layer) < LAYERS.indexOf(target.layer)) {
        addError(
          pointer,
          `module '${id}' layer '${source.layer}' cannot depend on outer layer '${target.layer}' of '${targetId}'`,
        )
      }
      if (source.context === target.context) return
      if (!nonEmptyString(via) || !(CROSS_CONTEXT_VIA as readonly string[]).includes(via)) {
        addError(pointer, `cross-context dependency '${id}' -> '${targetId}' requires a valid via`)
        return
      }
      if (!crossContextEdgeAllowed(source.role, target.role, via as CrossContextVia)) {
        addError(
          pointer,
          `cross-context dependency '${id}' -> '${targetId}' via '${via}' is incompatible with source role '${source.role}' and target role '${target.role}'`,
        )
      }
    })
  }

  const visiting = new Set<string>()
  const visited = new Set<string>()
  const stack: string[] = []
  const findCycle = (id: string): string[] | undefined => {
    if (visiting.has(id)) {
      const start = stack.indexOf(id)
      return [...stack.slice(start), id]
    }
    if (visited.has(id)) return undefined
    visiting.add(id)
    stack.push(id)
    for (const target of graph.get(id) ?? []) {
      const cycle = findCycle(target)
      if (cycle) return cycle
    }
    stack.pop()
    visiting.delete(id)
    visited.add(id)
    return undefined
  }
  for (const id of graph.keys()) {
    const cycle = findCycle(id)
    if (cycle) {
      addError(`/modules/${id}/depends_on`, `module dependency cycle: ${cycle.join(' -> ')}`)
      break
    }
  }

  for (const [id, value] of Object.entries(runtimes)) {
    const runtime = dict(value)
    const entrypoint = runtime.entrypoint
    if (!nonEmptyString(entrypoint)) {
      addError(`/runtime_units/${id}/entrypoint`, `runtime unit '${id}' requires an entrypoint`)
    } else if (!opts.pathExists(opts.join(opts.archDir, entrypoint))) {
      addError(
        `/runtime_units/${id}/entrypoint`,
        `runtime unit '${id}' entrypoint '${entrypoint}' does not exist`,
      )
    }
    if (Array.isArray(runtime.modules)) {
      runtime.modules.forEach((module, index) => {
        if (!nonEmptyString(module) || !(module in modules)) {
          addError(
            `/runtime_units/${id}/modules/${index}`,
            `runtime unit '${id}' references unknown module '${String(module ?? '')}'`,
          )
        }
      })
    }
  }

  return { errors, warnings }
}
