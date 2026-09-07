import { type Dirent, existsSync } from 'node:fs'
import { readdir } from 'node:fs/promises'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { DirectoryListing } from '../../check/src/canonical-document-set.ts'
import { canonicalDocumentNames, CONTEXT_DOCUMENTS } from '../../check/src/specification-doc.ts'

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

async function listFiles(root: string, directory: string): Promise<string[]> {
  let entries: Dirent[]
  try {
    entries = await readdir(resolve(root, directory), { withFileTypes: true })
  } catch {
    return []
  }
  return entries.filter((entry) => entry.isFile()).map((entry) => entry.name)
}

/** `docs/` 以下で名前を自由に決められる段。閉じた集合の検査は届かない。 */
const FREELY_NAMED_DOCUMENT_DIRECTORIES = new Set([
  'docs/contexts',
  'docs/development',
  'docs/runbooks',
  'docs/releases',
])

/** その段の直下にあるディレクトリ名を、並びを決めて返す。 */
async function listDirectories(root: string, directory: string): Promise<string[]> {
  let entries: Dirent[]
  try {
    entries = await readdir(resolve(root, directory), { withFileTypes: true })
  } catch {
    return []
  }
  return entries
    .filter((entry) => entry.isDirectory())
    .map((entry) => entry.name)
    .sort((a, b) => a.localeCompare(b))
}

/**
 * 分割配置がファイル集合を閉じている段。`docs/` 以下を、名前を自由に決められる段を
 * 除いて全部たどる。固定の一覧に無い段もここで挙げるので、`SYSTEM_DOCUMENT_DIRECTORIES`
 * から段を落としても、そこにある文書が検査から静かに消えることはない。文書を集める側と、
 * 文書でないファイルを拒否する側が同じ一覧を読むので、両者が食い違うこともない。
 */
export async function listCanonicalDirectories(root = WORKSPACE_ROOT): Promise<DirectoryListing[]> {
  const listings: DirectoryListing[] = []
  const pending = ['docs']
  while (pending.length) {
    const directory = pending.shift() as string
    listings.push({ directory, files: await listFiles(root, directory) })
    for (const name of await listDirectories(root, directory)) {
      const child = `${directory}/${name}`
      if (!FREELY_NAMED_DOCUMENT_DIRECTORIES.has(child)) pending.push(child)
    }
  }
  for (const name of await listDirectories(root, 'docs/contexts')) {
    listings.push({
      directory: `docs/contexts/${name}`,
      files: await listFiles(root, `docs/contexts/${name}`),
    })
  }
  return listings
}

/**
 * 分割配置の正本文書。配置が定める名前だけを返すので、隣にある無関係な Markdown を
 * 仕様の原稿と取り違えることはない。そのファイルが存在してよいかどうかを問うのは
 * `verifyCanonicalDocumentSet` であってここではない。対象を集める操作が落ちると、
 * 差分を読むだけの操作まで未登録のファイルを理由に落ちることになる。
 */
async function discoverSpecificationDocuments(root: string): Promise<string[]> {
  const documents: string[] = []
  for (const listing of await listCanonicalDirectories(root)) {
    const names = new Set<string>(canonicalDocumentNames(listing.directory) ?? CONTEXT_DOCUMENTS)
    for (const name of listing.files) {
      if (names.has(name)) documents.push(`${listing.directory}/${name}`)
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
