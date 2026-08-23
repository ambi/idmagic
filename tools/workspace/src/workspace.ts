import { type Dirent, existsSync } from 'node:fs'
import { readdir } from 'node:fs/promises'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { CONTEXT_DOCUMENTS, ROOT_DOCUMENTS } from '../../check/src/specification-doc.ts'

const HERE = dirname(fileURLToPath(import.meta.url))
export const TOOLS_DIR = resolve(HERE, '../..')
export const WORKSPACE_ROOT =
  process.env.SPEC_WORKSPACE_ROOT ??
  (resolve(process.cwd()) === TOOLS_DIR ? resolve(TOOLS_DIR, '..') : resolve(process.cwd()))

export type WorkspaceConfig = {
  specification?: string
  documents: string[]
  workItems?: string
}

async function discoverSingleFile(
  directory: string,
  predicate: (name: string) => boolean,
  description: string,
): Promise<string> {
  const matches = (await readdir(directory, { withFileTypes: true }))
    .filter((entry) => entry.isFile() && predicate(entry.name))
    .map((entry) => resolve(directory, entry.name))
    .sort()
  if (matches.length !== 1) {
    throw new Error(`expected exactly one ${description} in ${directory}, found ${matches.length}`)
  }
  return matches[0]!
}

export async function discoverGeneratedOpenApi(root = WORKSPACE_ROOT): Promise<string> {
  return discoverSingleFile(
    resolve(root, 'spec/generated/openapi'),
    (name) => name.endsWith('.json'),
    'generated OpenAPI JSON file',
  )
}

export async function discoverOpenApiBaseline(root = WORKSPACE_ROOT): Promise<string> {
  return discoverSingleFile(
    resolve(root, 'spec'),
    (name) => name.endsWith('.openapi.baseline.json'),
    'OpenAPI baseline JSON file',
  )
}

const EXCLUDED = new Set(['.git', 'node_modules', 'vendor', 'dist', 'build', 'generated'])

async function scanNamed(root: string, names: Set<string>, dir = root, found: string[] = []) {
  let entries: Dirent[]
  try {
    entries = await readdir(dir, { withFileTypes: true })
  } catch {
    return found
  }
  for (const entry of entries) {
    const absolute = resolve(dir, entry.name)
    if (entry.isDirectory()) {
      if (EXCLUDED.has(entry.name) || entry.name.startsWith('.')) continue
      await scanNamed(root, names, absolute, found)
    } else if (entry.isFile() && names.has(entry.name)) {
      found.push(relative(root, absolute))
    }
  }
  return found
}

/**
 * The canonical documents of one directory in the split layout. Only the names
 * the layout defines are returned, so an unrelated Markdown file next to them
 * is not mistaken for specification source.
 */
async function scanCanonicalDocuments(root: string, directory: string): Promise<string[]> {
  const names = new Set<string>(directory === 'docs' ? ROOT_DOCUMENTS : CONTEXT_DOCUMENTS)
  let entries: Dirent[]
  try {
    entries = await readdir(resolve(root, directory), { withFileTypes: true })
  } catch {
    return []
  }
  return entries
    .filter((entry) => entry.isFile() && names.has(entry.name))
    .map((entry) => `${directory}/${entry.name}`)
}

async function discoverSpecificationDocuments(root: string): Promise<string[]> {
  const documents = await scanCanonicalDocuments(root, 'docs')
  let contexts: Dirent[] = []
  try {
    contexts = await readdir(resolve(root, 'docs/contexts'), { withFileTypes: true })
  } catch {
    contexts = []
  }
  for (const entry of contexts) {
    if (entry.isDirectory()) {
      documents.push(...(await scanCanonicalDocuments(root, `docs/contexts/${entry.name}`)))
    }
  }
  return documents.sort()
}

export async function discoverWorkspaceConfig(root = WORKSPACE_ROOT): Promise<WorkspaceConfig> {
  const specification = existsSync(resolve(root, 'spec/main.tsp')) ? 'spec/main.tsp' : undefined
  const workItems = existsSync(resolve(root, 'work-items')) ? 'work-items' : undefined
  const documents = await discoverSpecificationDocuments(root)
  const legacyDocuments = (
    await scanNamed(root, new Set(['ARCHITECTURE.md', 'requirements.md', 'SPECIFICATION.md']))
  ).sort()
  if (legacyDocuments.length > 0) {
    throw new Error(`legacy specification documents found: ${legacyDocuments.join(', ')}`)
  }
  if (!specification && !workItems && documents.length === 0) {
    throw new Error(`no specification-first workspace targets found under ${root}`)
  }
  return { specification, documents, workItems }
}

export async function loadWorkspaceConfig(): Promise<WorkspaceConfig> {
  return discoverWorkspaceConfig()
}

export async function runTool(args: string[]): Promise<void> {
  const proc = Bun.spawn(['bun', 'run', ...args], {
    cwd: TOOLS_DIR,
    stdout: 'inherit',
    stderr: 'inherit',
  })
  const code = await proc.exited
  if (code !== 0) throw new Error(`${args.join(' ')} exited with ${code}`)
}

export function rootPath(path: string): string {
  return resolve(WORKSPACE_ROOT, path)
}
