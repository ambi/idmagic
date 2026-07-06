/**
 * Filesystem loaders for the four input kinds:
 *
 *   - SCL document   — single YAML file (Bun's native YAML importer)
 *   - Decisions      — directory of *.md (CONCEPTION*.md + ADR-*.md)
 *   - Work items     — directory of *.yaml
 *
 * Pure-ish: file IO only. No network, no clock.
 */

import { readFile, readdir } from 'node:fs/promises'
import { basename, dirname, extname, join, resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { splitTitle } from './markdown.ts'
import type {
  ArchitectureDoc,
  ChangeEntry,
  DecisionDoc,
  SclBundle,
  SclDocument,
  WorkItem,
} from './types.ts'

export async function loadScl(path: string): Promise<SclDocument> {
  const mod = await import(pathToFileURL(path).href)
  const data = (mod as { default?: unknown }).default ?? mod
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    throw new Error(`SCL document ${path} did not parse to an object`)
  }
  return data as SclDocument
}

export async function loadSclBundle(path: string): Promise<SclBundle> {
  const root = await loadScl(path)
  const baseDir = dirname(path)
  const contexts = []
  for (const [name, entry] of Object.entries(root.context_map ?? {})) {
    if (!entry.path) continue
    const contextPath = resolve(baseDir, entry.path)
    const document = await loadScl(contextPath)
    contexts.push({ name, path: entry.path, document })
  }
  return { root, contexts }
}

// ADR filenames may carry a leading context prefix (e.g. `idp-ADR-024-...`,
// `repo-ADR-001-...`); the number is the context-local sequence after `ADR-`.
// The prefix is optional so pre-migration `ADR-NNN-...` names still parse.
const ADR_FILENAME_RE = /^(?:[a-z0-9]+-)?ADR-(\d{1,4})-.+\.md$/i
const CONCEPTION_FILENAME_RE = /^CONCEPTION(?:_[A-Z]+)?\.md$/

export async function loadDecisions(dir: string): Promise<DecisionDoc[]> {
  const names = await readdir(dir)
  const wanted = names
    .filter((n) => CONCEPTION_FILENAME_RE.test(n) || ADR_FILENAME_RE.test(n))
    .sort()
  const out: DecisionDoc[] = []
  for (const name of wanted) {
    const path = join(dir, name)
    const source = await readFile(path, 'utf8')
    const isConception = CONCEPTION_FILENAME_RE.test(name)
    const adrMatch = name.match(ADR_FILENAME_RE)
    const id = isConception
      ? name.toLowerCase().replace(/\.md$/, '').replace(/_/g, '-')
      : name.toLowerCase().replace(/\.md$/, '')
    const { title, body } = splitTitle(source, basename(name, '.md'))
    out.push({
      id,
      title,
      kind: isConception ? 'conception' : 'adr',
      filename: name,
      body,
      number: adrMatch ? Number.parseInt(adrMatch[1] ?? '0', 10) : undefined,
    })
  }
  return out
}

const ARCH_BODY_HEADINGS: Record<string, keyof ArchitectureDoc> = {
  overview: 'overview',
  structure: 'structure',
  stack: 'stack',
  'structural decisions': 'structural_decisions',
  'cross-cutting concerns': 'cross_cutting_concerns',
  diagrams: 'diagrams',
}

/**
 * Load an ARCHITECTURE.md map: nested YAML frontmatter (parsed with the real
 * YAML engine) merged with the prose body sections. Returns null when the file
 * is absent or empty.
 */
export async function loadArchitecture(path: string): Promise<ArchitectureDoc | null> {
  let text: string
  try {
    text = await readFile(path, 'utf8')
  } catch {
    return null
  }
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
  const sections: Record<string, string> = {}
  for (const sec of body.split(/(?=^#{1,2}\s+)/m)) {
    const lines = sec.split('\n')
    const header = (lines[0] ?? '')
      .match(/^#{1,2}\s+(.+)$/)?.[1]
      ?.trim()
      .toLowerCase()
    if (!header) continue
    const key = ARCH_BODY_HEADINGS[header]
    if (!key) continue
    const content = lines.slice(1).join('\n').trim()
    if (content) sections[key] = content
  }
  return { ...frontmatter, ...sections } as ArchitectureDoc
}

/** Closed (completed / cancelled) work items live in this subdirectory. */
const DONE_SUBDIR = 'done'

async function listMdFiles(dir: string): Promise<string[]> {
  try {
    const entries = await readdir(dir, { withFileTypes: true })
    return entries
      .filter((e) => e.isFile() && extname(e.name) === '.md')
      .map((e) => join(dir, e.name))
  } catch {
    // A missing directory (e.g. no done/ yet) just yields no files.
    return []
  }
}

function parseMarkdownWorkItem(id: string, text: string): WorkItem | null {
  const match = text.match(/^---\s*\r?\n([\s\S]*?)\r?\n---\s*\r?\n([\s\S]*)$/)
  if (!match || match[1] === undefined || match[2] === undefined) return null
  const wi: Record<string, unknown> = { id }
  let bodyText = text

  if (match) {
    const yamlText = match[1]
    bodyText = match[2]

    // Simple key-value parser for frontmatter
    const fmLines = yamlText.split('\n')
    for (const line of fmLines) {
      const clean = line.trim()
      if (!clean || clean.startsWith('#')) continue
      const colonIdx = clean.indexOf(':')
      if (colonIdx > 0) {
        const key = clean.slice(0, colonIdx).trim()
        let val = clean.slice(colonIdx + 1).trim()
        if (
          (val.startsWith('"') && val.endsWith('"')) ||
          (val.startsWith("'") && val.endsWith("'"))
        ) {
          val = val.slice(1, -1)
        }
        if (val.startsWith('[') && val.endsWith(']')) {
          wi[key] = val
            .slice(1, -1)
            .split(',')
            .map((s) => s.trim().replace(/^['"]|['"]$/g, ''))
        } else if (val === 'true') {
          wi[key] = true
        } else if (val === 'false') {
          wi[key] = false
        } else {
          wi[key] = val
        }
      }
    }
  }

  // Parse markdown headers. Accept a single H1 title plus H2 sections (current
  // format) as well as the older all-H1 sections, so both parse the same.
  const sections = bodyText.split(/(?=^#{1,2}\s+)/m)
  for (const sec of sections) {
    const lines = sec.split('\n')
    const headerLine = lines[0] ?? ''
    const headerMatch = headerLine.match(/^#{1,2}\s+(.+)$/)
    if (headerMatch?.[1]) {
      const headerTitle = headerMatch[1].trim().toLowerCase()
      const content = lines.slice(1).join('\n').trim()
      if (!content) continue

      if (headerTitle === 'motivation') {
        wi.motivation = content
      } else if (headerTitle === 'scope') {
        wi.scope = content
      } else if (headerTitle === 'out of scope') {
        wi.out_of_scope = content
      } else if (headerTitle === 'plan') {
        wi.plan = content
      } else if (headerTitle === 'tasks') {
        wi.tasks = content
      } else if (headerTitle === 'verification') {
        wi.verification = content
      } else if (headerTitle === 'risk notes') {
        wi.risk_notes = content
      } else if (headerTitle === 'completion') {
        const completion: Record<string, unknown> = {}
        const compLines = content.split('\n')
        let currentField = ''
        let currentText: string[] = []

        const flushField = () => {
          if (currentField) {
            if (currentField === 'verification') {
              // Convert bullet points to list or string
              completion.verification = currentText.join('\n').trim()
            } else {
              completion[currentField] = currentText.join('\n').trim()
            }
            currentField = ''
            currentText = []
          }
        }

        for (const line of compLines) {
          const m = line.match(/^-\s+\*\*([^*]+)\*\*:\s*(.*)$/)
          if (m?.[1]) {
            flushField()
            const label = m[1].trim().toLowerCase()
            const value = m[2] ? m[2].trim() : ''
            if (label === 'completed at') {
              completion.completed_at = value
            } else if (label === 'summary') {
              currentField = 'summary'
              if (value) currentText.push(value)
            } else if (label === 'verification results') {
              currentField = 'verification'
              if (value) currentText.push(value)
            }
          } else if (currentField) {
            currentText.push(line.replace(/^\s{2}/, ''))
          }
        }
        flushField()
        wi.completion = completion
      }
    }
  }

  return wi as WorkItem
}

export async function loadChanges(dir: string): Promise<ChangeEntry[]> {
  const open = await listMdFiles(dir)
  const done = await listMdFiles(join(dir, DONE_SUBDIR))
  const files = [...open, ...done].sort((a, b) => basename(a).localeCompare(basename(b)))
  const out: ChangeEntry[] = []
  for (const wiPath of files) {
    const id = basename(wiPath, '.md')
    try {
      const text = await readFile(wiPath, 'utf8')
      const work_item = parseMarkdownWorkItem(id, text)
      if (work_item) {
        out.push({ id, work_item })
      }
    } catch {
      // Failed to parse — skip this file.
    }
  }
  return out
}
