import { existsSync } from 'node:fs'
import { readdir, readFile } from 'node:fs/promises'
import { basename, dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const HERE = dirname(fileURLToPath(import.meta.url))
export const TOOLS_DIR = resolve(HERE, '../..')
export const WORKSPACE_ROOT =
  process.env.RA_WORKSPACE_ROOT ??
  (resolve(process.cwd()) === TOOLS_DIR ? resolve(TOOLS_DIR, '..') : resolve(process.cwd()))

type AppArtifacts = {
  html?: string
  fullHtml?: string
  jsonSchema?: string
  openApi?: string
}

export type WorkspaceApp = {
  name: string
  root: string
  scl: string
  contextGlob?: string
  decisions?: string
  workItems?: string
  architecture?: string
  artifacts?: AppArtifacts
}

export type WorkspaceConfig = {
  apps: WorkspaceApp[]
  repositoryWorkItems?: string
  toolSpecs?: string[]
  architectureDocs?: string[]
}

function exists(path: string): boolean {
  return existsSync(path)
}

function workspacePath(workspaceRoot: string, path: string): string {
  return relative(workspaceRoot, path) || '.'
}

async function readSystemName(sclPath: string, fallback: string): Promise<string> {
  try {
    const text = await readFile(sclPath, 'utf8')
    const match = text.match(/^system:\s*["']?([^"'\n#]+)["']?\s*(?:#.*)?$/m)
    return match?.[1]?.trim() || fallback
  } catch {
    return fallback
  }
}

function defaultArtifacts(name: string): AppArtifacts {
  return {
    html: `spec/${name}.html`,
    fullHtml: `spec/${name}.full.html`,
    jsonSchema: `spec/${name}.models.schema.json`,
    openApi: `spec/${name}.openapi.json`,
  }
}

async function discoverToolSpecs(workspaceRoot: string): Promise<string[]> {
  const toolsDir = resolve(workspaceRoot, 'tools')
  if (!exists(toolsDir)) return []
  const entries = await readdir(toolsDir, { withFileTypes: true })
  const specs = entries
    .filter((entry) => entry.isDirectory())
    .map((entry) => resolve(toolsDir, entry.name, 'spec/scl.yaml'))
    .filter(exists)
    .map((path) => workspacePath(workspaceRoot, path))
  return specs.sort()
}

/**
 * Discover ARCHITECTURE.md maps: the repository-wide one at the root, plus any
 * per-context one at an app root. Placement mirrors decisions/ and work-items/
 * (REGENERATIVE_ARCHITECTURE.md §3.2.1 / ARCHITECTURE_FORMAT.md §1).
 */
function discoverArchitectureDocs(workspaceRoot: string, apps: WorkspaceApp[]): string[] {
  const docs = new Set<string>()
  const rootDoc = resolve(workspaceRoot, 'ARCHITECTURE.md')
  if (exists(rootDoc)) docs.add(workspacePath(workspaceRoot, rootDoc))
  for (const app of apps) {
    const appDoc = resolve(workspaceRoot, app.root, 'ARCHITECTURE.md')
    if (exists(appDoc)) docs.add(workspacePath(workspaceRoot, appDoc))
  }
  return [...docs].sort()
}

export async function discoverWorkspaceConfig(
  workspaceRoot = WORKSPACE_ROOT,
): Promise<WorkspaceConfig> {
  const root = resolve(workspaceRoot)
  const apps: WorkspaceApp[] = []
  const appScl = resolve(root, 'spec/scl.yaml')
  if (exists(appScl)) {
    const fallbackName = basename(root)
    const name = await readSystemName(appScl, fallbackName)
    apps.push({
      name,
      root: '.',
      scl: 'spec/scl.yaml',
      contextGlob: exists(resolve(root, 'spec/contexts')) ? 'spec/contexts/*.yaml' : undefined,
      decisions: exists(resolve(root, 'decisions')) ? 'decisions' : undefined,
      workItems: exists(resolve(root, 'work-items')) ? 'work-items' : undefined,
      architecture: exists(resolve(root, 'ARCHITECTURE.md')) ? 'ARCHITECTURE.md' : undefined,
      artifacts: defaultArtifacts(name),
    })
  }

  const repositoryWorkItems =
    apps.length === 0 && exists(resolve(root, 'work-items')) ? 'work-items' : undefined
  const toolSpecs = await discoverToolSpecs(root)
  const architectureDocs = discoverArchitectureDocs(root, apps)
  const config = { apps, repositoryWorkItems, toolSpecs, architectureDocs }
  if (apps.length === 0 && repositoryWorkItems === undefined && toolSpecs.length === 0) {
    throw new Error(`no RA workspace targets found under ${root}`)
  }
  return config
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
