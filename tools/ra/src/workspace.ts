import { type Dirent, existsSync } from 'node:fs'
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
  architectureLedgers?: string[]
  verificationManifest?: string
  verificationEvidence?: string
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

const ARCHITECTURE_SCAN_EXCLUDED = new Set(['.git', 'node_modules', 'vendor', 'dist', 'build'])

/**
 * Walk the workspace for the second-layer Architecture artifacts: the prose
 * design records (`ARCHITECTURE.md`) and the machine-checked ledgers
 * (`architecture.yaml`). Both may sit at the workspace root or next to the code
 * a bounded context owns (ADR-143 / ARCHITECTURE_FORMAT.md §1).
 */
async function scanArchitectureArtifacts(
  workspaceRoot: string,
  dir = workspaceRoot,
  found: { docs: string[]; ledgers: string[] } = { docs: [], ledgers: [] },
): Promise<{ docs: string[]; ledgers: string[] }> {
  let entries: Dirent[]
  try {
    entries = await readdir(dir, { withFileTypes: true })
  } catch {
    return found
  }
  for (const entry of entries) {
    const absolute = resolve(dir, entry.name)
    if (entry.isDirectory()) {
      if (ARCHITECTURE_SCAN_EXCLUDED.has(entry.name) || entry.name.startsWith('.')) continue
      await scanArchitectureArtifacts(workspaceRoot, absolute, found)
      continue
    }
    if (!entry.isFile()) continue
    if (entry.name === 'ARCHITECTURE.md') found.docs.push(workspacePath(workspaceRoot, absolute))
    if (entry.name === 'architecture.yaml')
      found.ledgers.push(workspacePath(workspaceRoot, absolute))
  }
  return found
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
  const scanned = await scanArchitectureArtifacts(root)
  const architectureDocs = [...scanned.docs].sort()
  const architectureLedgers = [...scanned.ledgers].sort()
  const verificationManifest = exists(resolve(root, 'verification/manifest.yaml'))
    ? 'verification/manifest.yaml'
    : undefined
  const verificationEvidence = exists(resolve(root, 'verification/evidence.yaml'))
    ? 'verification/evidence.yaml'
    : undefined
  const config = {
    apps,
    repositoryWorkItems,
    toolSpecs,
    architectureDocs,
    architectureLedgers,
    verificationManifest,
    verificationEvidence,
  }
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
