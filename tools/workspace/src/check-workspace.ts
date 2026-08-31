#!/usr/bin/env bun

import { existsSync, readFileSync } from 'node:fs'
import { readFile, readdir } from 'node:fs/promises'
import { basename, extname, join } from 'node:path'
import { compareOpenApi, type JsonSchema } from '../../check-api-compat/src/compat.ts'
import { verifyCanonicalDocumentSet } from '../../check/src/canonical-document-set.ts'
import {
  diffFeatureMaturities,
  type DocumentationImpactEnvironment,
  verifyDocumentationImpact,
} from '../../check/src/documentation-impact.ts'
import { verifySubdomainClassification } from '../../check/src/subdomain-classification.ts'
import {
  type WorkItemDependencyRecord,
  verifyWorkItemDependencies,
} from '../../check/src/work-item-dependencies.ts'
import {
  type ReferenceEnvironment,
  verifyWorkItemReferences,
} from '../../check/src/work-item-references.ts'
import { parseFrontmatterAndMarkdown } from '../../check/src/main.ts'
import {
  type PrimaryUseCaseEnvironment,
  verifyPrimaryUseCaseEvidence,
} from '../../check/src/primary-use-case-evidence.ts'
import { diffSpecifications, diffWorkspaceSpecifications } from '../../check/src/spec-diff.ts'
import {
  discoverGeneratedOpenApi,
  discoverOpenApiBaseline,
  listCanonicalDirectories,
  loadWorkspaceConfig,
  rootPath,
  runTool,
} from './workspace.ts'

const args = new Set(process.argv.slice(2))
if (args.has('--help') || args.has('-h')) {
  process.stdout.write('Usage: check-workspace [--work-items] [--ids] [--documents]\n')
  process.exit(0)
}
const valid = new Set(['--work-items', '--ids', '--documents'])
for (const arg of args) {
  if (!valid.has(arg)) throw new Error(`unknown option ${arg}`)
}
const all = args.size === 0
const config = await loadWorkspaceConfig()

const repository: ReferenceEnvironment = {
  exists: (path) => existsSync(rootPath(path)),
  read: (path) => {
    try {
      return readFileSync(rootPath(path), 'utf8')
    } catch {
      return undefined
    }
  },
}

async function workItemFiles(root: string): Promise<string[]> {
  const result: string[] = []
  for (const dir of [root, join(root, 'done')]) {
    try {
      for (const entry of await readdir(dir, { withFileTypes: true })) {
        if (entry.isFile() && extname(entry.name) === '.md') result.push(join(dir, entry.name))
      }
    } catch {
      // An absent done directory is valid.
    }
  }
  return result.sort()
}

function collectStrings(value: unknown, strings: string[]): void {
  if (typeof value === 'string') {
    strings.push(value)
  } else if (Array.isArray(value)) {
    for (const item of value) collectStrings(item, strings)
  } else if (typeof value === 'object' && value !== null) {
    for (const item of Object.values(value)) collectStrings(item, strings)
  }
}

/** 標準 verify の依存グラフと CI が直接呼ぶ mise タスクを返す。 */
async function requiredVerificationTasks(): Promise<ReadonlySet<string>> {
  const required = new Set<string>()
  try {
    const source = await readFile(rootPath('mise.toml'), 'utf8')
    const config = Bun.TOML.parse(source) as {
      tasks?: Record<string, { depends?: unknown }>
    }
    const tasks = config.tasks ?? {}
    const visit = (name: string): void => {
      if (required.has(name)) return
      required.add(name)
      const dependencies = tasks[name]?.depends
      if (!Array.isArray(dependencies)) return
      for (const dependency of dependencies) {
        if (typeof dependency === 'string') visit(dependency)
      }
    }
    visit('verify')
  } catch {
    // 最小 fixture に mise.toml が無い場合、標準タスクは空集合でよい。
  }

  try {
    const directory = rootPath('.github/workflows')
    for (const entry of await readdir(directory, { withFileTypes: true })) {
      if (!entry.isFile() || !/\.ya?ml$/.test(entry.name)) continue
      const workflow = Bun.YAML.parse(await readFile(join(directory, entry.name), 'utf8'))
      const strings: string[] = []
      collectStrings(workflow, strings)
      for (const source of strings) {
        for (const match of source.matchAll(/\bmise run ([a-z0-9][a-z0-9-]*)/g)) {
          if (match[1]) required.add(match[1])
        }
      }
    }
  } catch {
    // 最小 fixture に CI 定義が無い場合も空集合でよい。
  }
  return required
}

function revisionFile(ref: string, path: string): string {
  const result = Bun.spawnSync(['git', 'show', `${ref}:${path}`], { cwd: rootPath('.') })
  return result.exitCode === 0 ? result.stdout.toString() : ''
}

async function breakingApiChanges(): Promise<string[]> {
  try {
    const [baselinePath, currentPath] = await Promise.all([
      discoverOpenApiBaseline(rootPath('.')),
      discoverGeneratedOpenApi(rootPath('.')),
    ])
    const [baseline, current] = await Promise.all([
      readFile(baselinePath, 'utf8'),
      readFile(currentPath, 'utf8'),
    ])
    return compareOpenApi(
      JSON.parse(baseline) as JsonSchema,
      JSON.parse(current) as JsonSchema,
    ).map((finding) => `${finding.operation}: ${finding.message}`)
  } catch {
    // Minimal fixtures and a fresh checkout may not have generated OpenAPI yet.
    return []
  }
}

async function documentationImpactEnvironment(): Promise<DocumentationImpactEnvironment> {
  const featureRegistryPath = 'backend/cmd/internal/bootstrap/features.go'
  let specificationDiff = diffSpecifications(new Map(), new Map())
  try {
    specificationDiff = await diffWorkspaceSpecifications(rootPath('.'))
  } catch {
    // Minimal fixtures are not Git repositories and have no baseline revision.
  }
  return {
    read: repository.read,
    specificationDiff,
    maturityChanges: diffFeatureMaturities(
      revisionFile('main', featureRegistryPath),
      repository.read(featureRegistryPath) ?? '',
    ),
    breakingApiChanges: await breakingApiChanges(),
  }
}

if ((all || args.has('--work-items')) && config.workItems) {
  const files = await workItemFiles(rootPath(config.workItems))
  await runTool(['check/src/main.ts', '--schema=work-item', ...files])
  const records: WorkItemDependencyRecord[] = []
  const primaryUseCaseEnvironment: PrimaryUseCaseEnvironment = {
    read: repository.read,
    requiredTasks: await requiredVerificationTasks(),
  }
  const documentationEnvironment = await documentationImpactEnvironment()
  for (const path of files) {
    const source = await readFile(path, 'utf8')
    const data = parseFrontmatterAndMarkdown(path, source) as {
      status?: unknown
      depends_on?: unknown
      affected_spec?: unknown
      initial_context?: unknown
    }
    records.push({
      id: basename(path, '.md'),
      path,
      depends_on: Array.isArray(data.depends_on)
        ? data.depends_on.filter((item): item is string => typeof item === 'string')
        : [],
    })

    for (const finding of verifyWorkItemReferences(data, repository)) {
      console.error(`${path}: ${finding}`)
      process.exit(1)
    }
    for (const finding of verifyPrimaryUseCaseEvidence(data, primaryUseCaseEnvironment)) {
      console.error(`${path}: ${finding}`)
      process.exit(1)
    }
    for (const finding of verifyDocumentationImpact(data, documentationEnvironment)) {
      console.error(`${path}: ${finding}`)
      process.exit(1)
    }
  }
  const findings = verifyWorkItemDependencies(records)
  if (findings.length) {
    for (const finding of findings) console.error(`${finding.path}: ${finding.message}`)
    process.exit(1)
  }
  console.log(`ok  ${records.length} work-item dependency record(s)`)
}

if (all || args.has('--documents')) {
  // 閉じた集合の強制は、検証すべき文書が 1 件も見つからないときこそ働かなければ
  // ならない。名前を全部打ち間違えた作業ツリーは、集めた結果が空になるという形で
  // 現れる。ここを文書の件数で条件付けると、その最悪の場合だけ検査が飛ぶ。
  const listings = await listCanonicalDirectories()
  const findings = verifyCanonicalDocumentSet(listings)
  if (findings.length) {
    for (const finding of findings) console.error(`fail  ${finding.path}: ${finding.message}`)
    process.exit(1)
  }
  const contextDirectories = listings
    .filter((listing) => listing.directory.startsWith('docs/contexts/'))
    .map((listing) => listing.directory.slice('docs/contexts/'.length))
  const classifications = verifySubdomainClassification(
    repository.read('docs/README.md') ?? '',
    contextDirectories,
  )
  if (classifications.length) {
    for (const finding of classifications) {
      console.error(`fail  docs/README.md:${finding.line}: ${finding.message}`)
    }
    process.exit(1)
  }
  if (config.documents.length > 0) {
    await runTool(['check/src/check-specifications.ts', ...config.documents.map(rootPath)])
  }
}

if ((all || args.has('--ids')) && config.workItems) {
  await runTool(['check/src/check-ids.ts', '--work-items', rootPath(config.workItems)])
}
