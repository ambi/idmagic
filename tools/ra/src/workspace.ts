import { existsSync } from 'node:fs'
import { readdir } from 'node:fs/promises'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
export const TOOLS_DIR = resolve(HERE, '../..')
export const WORKSPACE_ROOT =
  process.env.RA_WORKSPACE_ROOT ??
  (resolve(process.cwd()) === TOOLS_DIR ? resolve(TOOLS_DIR, '..') : resolve(process.cwd()))

export type WorkspaceConfig = {
  specification?: string
  requirements: string[]
  workItems?: string
  decisions?: string
  architectureDocs: string[]
}

const EXCLUDED = new Set(['.git', 'node_modules', 'vendor', 'dist', 'build', 'generated'])

async function scanNamed(root: string, names: Set<string>, dir = root, found: string[] = []) {
  let entries
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

export async function discoverWorkspaceConfig(root = WORKSPACE_ROOT): Promise<WorkspaceConfig> {
  const specification = existsSync(resolve(root, 'spec/main.tsp')) ? 'spec/main.tsp' : undefined
  const workItems = existsSync(resolve(root, 'work-items')) ? 'work-items' : undefined
  const decisions = existsSync(resolve(root, 'decisions')) ? 'decisions' : undefined
  const architectureDocs = (await scanNamed(root, new Set(['ARCHITECTURE.md']))).sort()
  const requirements = (await scanNamed(root, new Set(['requirements.md']))).sort()
  if (!specification && !workItems && requirements.length === 0) {
    throw new Error(`no Regenerative Architecture workspace targets found under ${root}`)
  }
  return { specification, requirements, workItems, decisions, architectureDocs }
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
