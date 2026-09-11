import { basename } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { compareOpenApi, type JsonSchema } from './api-compat.ts'
import {
  claimsSpecificationAddition,
  diffFeatureMaturities,
  type DocumentationImpactEnvironment,
  verifyDocumentationImpact,
} from './documentation-impact.ts'
import { validateMarkdownRecord } from './work-item-markdown.ts'
import type { PrimaryUseCaseEnvironment } from './primary-use-case-evidence.ts'
import { verifyPrimaryUseCaseEvidence } from './primary-use-case-evidence.ts'
import type { CheckOutcome } from './runner.ts'
import {
  diffSpecifications,
  diffWorkspaceSpecifications,
  type SpecificationDiff,
} from './spec-diff.ts'
import { parseMiseTasks, taskClosure } from './verification-tasks.ts'
import {
  type WorkItemDependencyRecord,
  verifyWorkItemDependencies,
} from './work-item-dependencies.ts'
import type { ReferenceEnvironment } from './work-item-references.ts'
import { verifyWorkItemReferences } from './work-item-references.ts'

export type ParsedWorkItem = {
  id: string
  path: string
  data?: Record<string, unknown>
  formatFindings: Array<{ line: number; column: number; message: string }>
}

type RecordValidator = typeof validateMarkdownRecord

async function workItemPaths(snapshot: WorkspaceSnapshot): Promise<string[]> {
  const paths: string[] = []
  for (const directory of ['work-items', 'work-items/done']) {
    try {
      for (const entry of await snapshot.list(directory)) {
        if (entry.isFile() && entry.name.endsWith('.md')) paths.push(`${directory}/${entry.name}`)
      }
    } catch {
      // done ディレクトリがまだ無い workspace は有効である。
    }
  }
  return paths.sort()
}

/** 一回の検査で共有する work item の一覧、本文、解析結果を作る。 */
export async function loadWorkItems(
  snapshot: WorkspaceSnapshot,
  validate: RecordValidator = validateMarkdownRecord,
): Promise<ParsedWorkItem[]> {
  return Promise.all(
    (await workItemPaths(snapshot)).map(async (path): Promise<ParsedWorkItem> => {
      const result = validate(path, await snapshot.read(path), 'work-item')
      return { id: basename(path, '.md'), path, data: result.data, formatFindings: result.findings }
    }),
  )
}

function collectStrings(value: unknown, strings: string[]): void {
  if (typeof value === 'string') strings.push(value)
  else if (Array.isArray(value)) for (const item of value) collectStrings(item, strings)
  else if (typeof value === 'object' && value !== null) {
    for (const item of Object.values(value)) collectStrings(item, strings)
  }
}

async function requiredVerificationTasks(
  snapshot: WorkspaceSnapshot,
): Promise<ReadonlySet<string>> {
  const required = new Set<string>()
  try {
    for (const task of taskClosure(parseMiseTasks(await snapshot.read('mise.toml')), 'verify')) {
      required.add(task)
    }
  } catch {
    // 最小 fixture に mise.toml が無い場合は空集合でよい。
  }
  try {
    for (const entry of await snapshot.list('.github/workflows')) {
      if (!entry.isFile() || !/\.ya?ml$/.test(entry.name)) continue
      const workflow = Bun.YAML.parse(await snapshot.read(`.github/workflows/${entry.name}`))
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

function revisionFile(snapshot: WorkspaceSnapshot, ref: string, path: string): string {
  const result = Bun.spawnSync(['git', 'show', `${ref}:${path}`], { cwd: snapshot.root })
  return result.exitCode === 0 ? result.stdout.toString() : ''
}

function changedWorkItemRecords(snapshot: WorkspaceSnapshot): ReadonlySet<string> | undefined {
  const changedFromMain = Bun.spawnSync(
    ['git', 'diff', '--name-only', 'main...', '--', 'work-items'],
    { cwd: snapshot.root },
  )
  const status = Bun.spawnSync(['git', 'status', '--porcelain', '--', 'work-items'], {
    cwd: snapshot.root,
  })
  if (changedFromMain.exitCode !== 0 && status.exitCode !== 0) return undefined
  const changed = new Set<string>()
  const add = (path: string) => {
    const name = path.trim().match(/([^/\\]+)\.md$/)?.[1]
    if (name) changed.add(name)
  }
  for (const line of changedFromMain.stdout.toString().split('\n')) add(line)
  for (const line of status.stdout.toString().split('\n')) {
    for (const part of line.slice(3).split(' -> ')) add(part)
  }
  return changed
}

function specificationAdditionsClaimed(
  diff: SpecificationDiff,
  changed: ReadonlySet<string> | undefined,
  records: readonly ParsedWorkItem[],
): boolean {
  const added = [...diff.addedScenarios, ...diff.addedStandards, ...diff.addedDeclarations]
  if (added.length === 0 || changed === undefined) return false
  return records.some(
    (record) =>
      changed.has(record.id) &&
      record.data !== undefined &&
      claimsSpecificationAddition(record.data, added),
  )
}

async function documentationImpactEnvironment(
  snapshot: WorkspaceSnapshot,
  records: readonly ParsedWorkItem[],
): Promise<DocumentationImpactEnvironment> {
  const featureRegistryPath = 'backend/cmd/internal/bootstrap/features.go'
  let specificationDiff = diffSpecifications(new Map(), new Map())
  try {
    specificationDiff = await diffWorkspaceSpecifications(snapshot.root)
  } catch {
    // 最小 fixture は Git 履歴や仕様文書を持たない。
  }
  const changed = changedWorkItemRecords(snapshot)
  let breakingApiChanges: string[] = []
  try {
    breakingApiChanges = compareOpenApi(
      JSON.parse(await snapshot.read(await snapshot.openApiBaseline())) as JsonSchema,
      JSON.parse(await snapshot.read(await snapshot.generatedOpenApi())) as JsonSchema,
    ).map((finding) => `${finding.operation}: ${finding.message}`)
  } catch {
    // 最小 fixture と生成前の checkout には OpenAPI が無い。
  }
  return {
    read: (path) => (snapshot.exists(path) ? snapshot.readSync(path) : undefined),
    specificationDiff,
    changedRecords: changed,
    specificationAdditionsClaimed: specificationAdditionsClaimed(
      specificationDiff,
      changed,
      records,
    ),
    maturityChanges: diffFeatureMaturities(
      revisionFile(snapshot, 'main', featureRegistryPath),
      snapshot.exists(featureRegistryPath) ? snapshot.readSync(featureRegistryPath) : '',
    ),
    breakingApiChanges,
  }
}

export async function checkWorkItems(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  if (!snapshot.exists('work-items')) return { ok: true, lines: ['ok  0 work item(s)'] }
  const parsed = await loadWorkItems(snapshot)
  const lines = parsed.flatMap((record) =>
    record.formatFindings.map(
      (finding) => `${record.path}:${finding.line}:${finding.column}: ${finding.message}`,
    ),
  )
  const repository: ReferenceEnvironment = {
    exists: (path) => snapshot.exists(path),
    read: (path) => (snapshot.exists(path) ? snapshot.readSync(path) : undefined),
  }
  const primaryEnvironment: PrimaryUseCaseEnvironment = {
    read: repository.read,
    requiredTasks: await requiredVerificationTasks(snapshot),
  }
  const documentationEnvironment = await documentationImpactEnvironment(snapshot, parsed)
  const dependencyRecords: WorkItemDependencyRecord[] = []
  for (const record of parsed) {
    if (!record.data) continue
    dependencyRecords.push({
      id: record.id,
      path: record.path,
      depends_on: Array.isArray(record.data.depends_on)
        ? record.data.depends_on.filter((item): item is string => typeof item === 'string')
        : [],
    })
    lines.push(
      ...verifyWorkItemReferences(record.data, repository).map(
        (finding) => `${record.path}: ${finding}`,
      ),
      ...verifyPrimaryUseCaseEvidence(record.data, primaryEnvironment).map(
        (finding) => `${record.path}: ${finding}`,
      ),
      ...verifyDocumentationImpact(record.data, documentationEnvironment).map(
        (finding) => `${record.path}: ${finding}`,
      ),
    )
  }
  lines.push(
    ...verifyWorkItemDependencies(dependencyRecords).map(
      (finding) => `${finding.path}: ${finding.message}`,
    ),
  )
  return {
    ok: lines.length === 0,
    lines:
      lines.length > 0 ? lines : [`ok  ${dependencyRecords.length} work-item dependency record(s)`],
  }
}
