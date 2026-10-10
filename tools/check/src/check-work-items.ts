import { basename } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { readReleasePhase } from './release-state.ts'
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
import { changedWorkItemRecords, verifiedNow } from './work-item-changes.ts'
import {
  type WorkItemDependencyRecord,
  verifyWorkItemDependencies,
  verifyWorkItemIdentifiers,
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
  for (const directory of ['work-items/active', 'work-items/done']) {
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
  changed: ReadonlySet<string> | undefined,
): Promise<DocumentationImpactEnvironment> {
  const releasePhase = await readReleasePhase(snapshot)
  const featureRegistryPath = 'backend/cmd/internal/bootstrap/features.go'
  let specificationDiff = diffSpecifications(new Map(), new Map())
  try {
    specificationDiff = await diffWorkspaceSpecifications(snapshot.root)
  } catch {
    // 最小 fixture は Git 履歴や仕様文書を持たない。
  }
  let breakingApiChanges: string[] = []
  if (releasePhase === 'published') {
    breakingApiChanges = compareOpenApi(
      JSON.parse(await snapshot.read(await snapshot.openApiBaseline())) as JsonSchema,
      JSON.parse(await snapshot.read(await snapshot.generatedOpenApi())) as JsonSchema,
    ).map((finding) => `${finding.operation}: ${finding.message}`)
  }
  return {
    releasePhase,
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
  const changed = changedWorkItemRecords(snapshot)
  const verified = parsed.filter((record) => verifiedNow(record.path, changed))
  const lines = verified.flatMap((record) =>
    record.formatFindings.map(
      (finding) => `${record.path}:${finding.line}:${finding.column}: ${finding.message}`,
    ),
  )
  for (const entry of await snapshot.list('work-items')) {
    if (entry.isFile() && entry.name.endsWith('.md')) {
      lines.push(
        `work-items/${entry.name}: legacy work item location; move to work-items/active or work-items/done`,
      )
    }
  }
  const repository: ReferenceEnvironment = {
    exists: (path) => snapshot.exists(path),
    read: (path) => (snapshot.exists(path) ? snapshot.readSync(path) : undefined),
  }
  const primaryEnvironment: PrimaryUseCaseEnvironment = {
    read: repository.read,
    requiredTasks: await requiredVerificationTasks(snapshot),
  }
  const documentationEnvironment = await documentationImpactEnvironment(snapshot, parsed, changed)
  // 依存と識別番号は記録の間の現在の関係なので、変わっていない完了済みの記録も含めて見る。
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
  }
  for (const record of verified) {
    if (!record.data) continue
    const status = record.data.status
    const directory =
      status === 'completed' || status === 'cancelled' ? 'work-items/done' : 'work-items/active'
    if (!record.path.startsWith(`${directory}/`)) {
      lines.push(`${record.path}: status ${status} belongs in ${directory}`)
    }
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
  const identifiers = verifyWorkItemIdentifiers(dependencyRecords)
  lines.push(
    ...[...verifyWorkItemDependencies(dependencyRecords), ...identifiers.findings].map(
      (finding) => `${finding.path}: ${finding.message}`,
    ),
  )
  // 空き枠の警告は検査を落とさない。桁を増やす判断に要る観測であって、
  // 誰かが直せる欠陥ではないため、`ok` の根拠は所見の件数だけにする。
  const ok = lines.length === 0
  if (ok) lines.push(`ok  ${dependencyRecords.length} work-item dependency record(s)`)
  return { ok, lines: [...lines, ...identifiers.warnings] }
}
