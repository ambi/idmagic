/**
 * Parsing and cross-checks for the Architecture document (ARCHITECTURE.md).
 *
 * ARCHITECTURE.md is the current-state projection of the second layer
 * (`REGENERATIVE_ARCHITECTURE.md` §3.2.1); its format is normative in
 * `ARCHITECTURE_FORMAT.md`. It is a hybrid record: a structured YAML
 * frontmatter (contexts / modules — the machine-checked parts) plus prose
 * body sections (overview, structure, stack, decisions, ...).
 *
 * `parseArchitectureDoc` reads the hybrid file into one object so the JSON
 * Schema (`schemas/architecture.schema.json`) can validate its shape. Unlike
 * the work-item frontmatter — which is flat — the architecture frontmatter is
 * nested, so it is parsed with the real YAML engine (`Bun.YAML.parse`) rather
 * than the line-based reader.
 *
 * `verifyArchitecture` adds the cross-checks the schema cannot express: that
 * the map has not drifted from the workspace it describes (§4 of the format).
 * Everything here is pure except for the injected `pathExists`.
 */

import { type Finding, locatePointer } from './lib.ts'

/** SCL sections whose named members can be `realizes` targets. */
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

/**
 * Parse ARCHITECTURE.md into a single object: nested frontmatter merged with
 * body sections keyed by their heading. Frontmatter wins on key collisions.
 */
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

/** Collect `section.Name` element refs from a parsed SCL document. */
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
  /** Raw source text, for line-number resolution. */
  text?: string
  /** Absolute path of the directory containing the ARCHITECTURE.md. */
  archDir: string
  /** Workspace root, for resolving context roots. */
  workspaceRoot: string
  /** Every `section.Name` element present across the workspace's SCL. */
  sclElements: Set<string>
  /** Existence test for a path, injected so the checker stays pure. */
  pathExists: (absPath: string) => boolean
  /** Path join, injected (node:path.join). */
  join: (...parts: string[]) => string
}

/**
 * Cross-check an ARCHITECTURE.md against the workspace it describes:
 *   1. every `modules[].path` exists on disk;
 *   2. every `modules[].realizes` resolves to a real SCL element;
 *   3. `context` is one of the declared `contexts`, and each `contexts[].root`
 *      exists on disk.
 */
export function verifyArchitecture(doc: unknown, opts: ArchCheckOptions): ArchReport {
  const errors: Finding[] = []
  const warnings: Finding[] = []
  const text = opts.text ?? ''
  const data = doc as Record<string, unknown> | null | undefined
  if (!data || typeof data !== 'object') return { errors, warnings }

  const modules = data.modules
  if (modules && typeof modules === 'object' && !Array.isArray(modules)) {
    for (const [id, modU] of Object.entries(modules as Record<string, unknown>)) {
      const mod = modU as { path?: unknown; realizes?: unknown }
      if (typeof mod?.path === 'string' && mod.path.length > 0) {
        if (!opts.pathExists(opts.join(opts.archDir, mod.path))) {
          errors.push({
            line: locatePointer(text, `/modules/${id}/path`),
            column: 1,
            message: `architecture: module '${id}' path '${mod.path}' does not exist`,
          })
        }
      }
      if (Array.isArray(mod?.realizes)) {
        mod.realizes.forEach((ref, i) => {
          if (typeof ref === 'string' && !opts.sclElements.has(ref)) {
            errors.push({
              line: locatePointer(text, `/modules/${id}/realizes/${i}`),
              column: 1,
              message: `architecture: module '${id}' realizes '${ref}' which no SCL element defines`,
            })
          }
        })
      }
    }
  }

  const contexts = data.contexts
  if (contexts && typeof contexts === 'object' && !Array.isArray(contexts)) {
    const entries = contexts as Record<string, unknown>
    const keys = Object.keys(entries)
    const context = data.context
    if (typeof context === 'string' && keys.length > 0 && !keys.includes(context)) {
      errors.push({
        line: locatePointer(text, '/context'),
        column: 1,
        message: `architecture: context '${context}' is not among declared contexts (${keys.join(', ')})`,
      })
    }
    for (const [prefix, ctxU] of Object.entries(entries)) {
      const ctx = ctxU as { root?: unknown }
      if (
        typeof ctx?.root === 'string' &&
        !opts.pathExists(opts.join(opts.workspaceRoot, ctx.root))
      ) {
        errors.push({
          line: locatePointer(text, `/contexts/${prefix}/root`),
          column: 1,
          message: `architecture: context '${prefix}' root '${ctx.root}' does not exist`,
        })
      }
    }
  }

  return { errors, warnings }
}
