import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import {
  isProductionCode,
  productionCodeExclusions,
  type ProductionCodeExclusions,
} from './production-code.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'
import {
  diffSpecifications,
  isSpecificationSource,
  readRevision,
  readWorkingTree,
  type Snapshot,
  type SpecificationDiff,
} from './spec-diff.ts'
import {
  type CheckedCommit,
  type CheckedWorkItem,
  type SpecImpactDeclaration,
  type SpecReference,
  changedCharacterizations,
  specImpactDeclaration,
  verifySpecImpact,
  workRangeStart,
} from './spec-impact.ts'
import { parseFrontmatterAndMarkdown } from './work-item-markdown.ts'

const WORK_ITEM = /^work-items\/(?:(?:active|done)\/)?(wi-[^/]+)\.md$/
const TEST_FILE = /^(?:backend|frontend)\/.*(?:_test\.go|\.(?:test|spec)\.tsx?)$/
const GO_TEST_FILE = /^backend\/.*_test\.go$/
const CHECKPOINT = /^checkpoint\((wi-\d+)[^)]*\):/
const GENERATED_DIRECTORIES = ['node_modules', 'vendor', 'dist', 'build', 'generated']
const NO_DIFF = diffSpecifications(new Map(), new Map())

type Git = (args: string[]) => string | undefined

function gitIn(root: string): Git {
  return (args) => {
    const result = Bun.spawnSync(['git', ...args], { cwd: root })
    return result.exitCode === 0 ? result.stdout.toString() : undefined
  }
}

function lines(output: string | undefined): string[] {
  return (output ?? '').split('\n').filter((line) => line.length > 0)
}

/** `paths` のうち Go のテストファイルについて、`before` にあり `after` で変わったか消えた特性化テスト。 */
function characterizationChanges(
  paths: readonly string[],
  before: (path: string) => string | undefined,
  after: (path: string) => string | undefined,
): string[] {
  const addedSources = paths
    .filter((path) => GO_TEST_FILE.test(path) && before(path) === undefined)
    .map(after)
  return paths
    .filter((path) => GO_TEST_FILE.test(path))
    .flatMap((path) => {
      const original = before(path)
      const current = after(path)
      // 移動先でも本文が一致する場合だけ、削除と追加を一つの移動として扱う。
      // 移動と同時の期待値の変更は、従来どおり拒否する。
      if (
        current === undefined &&
        original !== undefined &&
        addedSources.some(
          (source) =>
            source !== undefined && changedCharacterizations(original, source).length === 0,
        )
      )
        return []
      return changedCharacterizations(original, current).map((name) => `${path} ${name}`)
    })
}

type WorkItemRecord = {
  status: string | undefined
  declaration: SpecImpactDeclaration | undefined
}

function workItemRecord(path: string, source: string | undefined): WorkItemRecord | undefined {
  if (source === undefined) return undefined
  try {
    const record = parseFrontmatterAndMarkdown(path, source)
    return {
      status: typeof record.status === 'string' ? record.status : undefined,
      declaration: specImpactDeclaration(record),
    }
  } catch {
    // 読めない記録は work-items の検査が報告する。ここでは宣言のない記録として扱う。
    return undefined
  }
}

function isUnderWay(record: WorkItemRecord | undefined): record is WorkItemRecord {
  return record?.status === 'in_progress'
}

/** 現在の兄弟配置と、移行前の Git 履歴を同じ記録として読む。 */
function workItemPaths(id: string): string[] {
  return [`work-items/active/${id}.md`, `work-items/done/${id}.md`, `work-items/${id}.md`]
}

/**
 * 仕様影響の検査が Git と作業ツリーから読む値。
 *
 * 同じリビジョンの仕様と同じ範囲の変更一覧を、work item ごとに読み直さないよう記憶する。
 */
function repository(
  snapshot: WorkspaceSnapshot,
  git: Git,
  completing?: { path: string; text: string },
) {
  const revisions = new Map<string, Snapshot>()
  const changes = new Map<string, string[]>()
  let workingTree: Promise<Snapshot> | undefined
  let exclusions: Promise<ProductionCodeExclusions> | undefined

  const revision = (sha: string): Snapshot => {
    let value = revisions.get(sha)
    if (!value) {
      value = readRevision(snapshot.root, sha)
      revisions.set(sha, value)
    }
    return value
  }
  /** `sha` から終端までの変更。終端の省略時だけ作業ツリーと未追跡のファイルを含む。 */
  const changedSince = (sha: string, end?: string): string[] => {
    const key = `${sha}:${end ?? 'working-tree'}`
    let value = changes.get(key)
    if (!value) {
      value = [
        ...lines(git(['diff', '--name-only', '--no-renames', sha, ...(end ? [end] : []), '--'])),
        ...(end ? [] : lines(git(['ls-files', '--others', '--exclude-standard']))),
      ]
      changes.set(key, value)
    }
    return value
  }
  const read = (path: string, at?: string): string | undefined =>
    at
      ? git(['show', `${at}:${path}`])
      : path === completing?.path
        ? completing.text.replace(/^status:.*$/m, 'status: in_progress')
        : snapshot.exists(path)
          ? snapshot.readSync(path)
          : undefined

  return {
    read,
    changedSince,
    /** `sha` から終端までの仕様差分。終端の省略時は作業ツリーを読む。 */
    diffSince: async (sha: string, end?: string): Promise<SpecificationDiff> => {
      if (!changedSince(sha, end).some(isSpecificationSource)) return NO_DIFF
      if (end) return diffSpecifications(revision(sha), revision(end))
      workingTree ??= readWorkingTree(snapshot.root)
      return diffSpecifications(revision(sha), await workingTree)
    },
    /** コミット一つが加えた仕様差分。 */
    diffOf: (sha: string, paths: readonly string[]): SpecificationDiff => {
      if (!paths.some(isSpecificationSource)) return NO_DIFF
      const parent = git(['rev-parse', '--verify', '--quiet', `${sha}^`])?.trim()
      return diffSpecifications(parent ? revision(parent) : new Map(), revision(sha))
    },
    workItem: (id: string, at?: string): { path: string; record: WorkItemRecord } | undefined => {
      for (const path of workItemPaths(id)) {
        const source = read(path, at)
        const record = workItemRecord(path, source)
        if (record) return { path, record }
      }
      return undefined
    },
    /** 作業ツリーにある work item のうち、識別番号が `number`（`wi-<番号>`）のもの。 */
    workItemsNumbered: (number: string): string[] =>
      workItemIds(
        lines(git(['ls-files', '--cached', '--others', '--exclude-standard', '--', 'work-items'])),
      ).filter((id) => id === number || id.startsWith(`${number}-`)),
    workItemAt: (sha: string, id: string): WorkItemRecord | undefined => {
      for (const path of workItemPaths(id)) {
        const record = workItemRecord(path, git(['show', `${sha}:${path}`]))
        if (record) return record
      }
      return undefined
    },
    productionCodeExclusions: (): Promise<ProductionCodeExclusions> => {
      exclusions ??= (async () => {
        const modulePath = read('go.mod')?.match(/^module\s+(\S+)/m)?.[1] ?? ''
        const sources = new Map<string, string>()
        if (snapshot.exists('backend')) {
          for (const path of await snapshot.files('backend', GENERATED_DIRECTORIES)) {
            if (path.endsWith('.go')) sources.set(path, await snapshot.read(path))
          }
        }
        return productionCodeExclusions(modulePath, sources)
      })()
      return exclusions
    },
  }
}

type Repository = ReturnType<typeof repository>

type CompletedRange = {
  commits: readonly string[]
  diff: SpecificationDiff
  end?: string
}

function workHistory(id: string, git: Git, repo: Repository) {
  return lines(git(['log', '--reverse', '--format=%H', '--', ...workItemPaths(id)])).map((sha) => ({
    sha,
    status: repo.workItemAt(sha, id)?.status,
  }))
}

/** 完了範囲は状態の履歴で閉じる。削除済みの宣言は読み戻さない。 */
async function completedRange(
  id: string,
  head: string,
  git: Git,
  repo: Repository,
): Promise<CompletedRange | undefined> {
  if (repo.workItem(id)?.record.status !== 'completed') return undefined
  const history = workHistory(id, git, repo)
  const start = workRangeStart(history, repo.workItemAt(head, id)?.status, head)
  const startSha = start && git(['rev-parse', '--verify', '--quiet', `${start}^{commit}`])?.trim()
  if (!startSha) return undefined
  const end = history.find((commit) => commit.status === 'completed')?.sha
  return {
    commits: lines(git(['rev-list', `${startSha}..${end ?? head}`])),
    diff: await repo.diffSince(startSha, end),
    end,
  }
}

function workItemIds(paths: readonly string[]): string[] {
  return [...new Set(paths.flatMap((path) => path.match(WORK_ITEM)?.[1] ?? []))]
}

function modifiedBy(declaration: SpecImpactDeclaration | undefined): SpecReference[] {
  return declaration?.kind === 'affected'
    ? declaration.references.filter((reference) => reference.impact === 'modifies')
    : []
}

async function checkedWorkItem(
  id: string,
  head: string,
  git: Git,
  repo: Repository,
): Promise<CheckedWorkItem | undefined> {
  const current = repo.workItem(id)
  if (!current || !isUnderWay(current.record) || !current.record.declaration) return undefined
  const history = workHistory(id, git, repo)
  const start = workRangeStart(history, repo.workItemAt(head, id)?.status, head)
  const startSha = start && git(['rev-parse', '--verify', '--quiet', `${start}^{commit}`])?.trim()
  const item = { path: current.path, declaration: current.record.declaration }
  if (!startSha) return { ...item, range: undefined }

  const activeCommits = new Set(lines(git(['rev-list', `${startSha}..${head}`])))
  const changed = repo.changedSince(startSha)
  const completed: CompletedRange[] = []
  for (const other of workItemIds(changed).filter((other) => other !== id)) {
    const range = await completedRange(other, head, git, repo)
    if (range) completed.push(range)
  }
  return {
    ...item,
    range: {
      diff: await repo.diffSince(startSha),
      completedDiffs: completed
        .filter((range) => !range.end || range.commits.some((sha) => activeCommits.has(sha)))
        .map((range) => range.diff),
      claimedByOthers: workItemIds(changed)
        .filter((other) => other !== id)
        .flatMap((other) => {
          const record = repo.workItem(other)?.record
          return isUnderWay(record) ? modifiedBy(record.declaration) : []
        }),
      changedTests: changed
        .filter((path) => TEST_FILE.test(path))
        .flatMap((path) => repo.read(path) ?? []),
      changedCharacterizations: characterizationChanges(
        changed,
        (path) => git(['show', `${startSha}:${path}`]),
        (path) => repo.read(path),
      ),
    },
  }
}

async function checkedCommit(sha: string, git: Git, repo: Repository): Promise<CheckedCommit> {
  const subject = git(['log', '-1', '--format=%s', sha])?.trim() ?? ''
  const trailer = git([
    'log',
    '-1',
    '--format=%(trailers:key=Spec-Impact,valueonly,unfold)',
    sha,
  ])?.trim()
  const paths = lines(
    git(['diff-tree', '--root', '--no-commit-id', '--name-only', '--no-renames', '-r', sha]),
  )
  const exclusions = await repo.productionCodeExclusions()
  const checkpointed = subject.match(CHECKPOINT)?.[1]
  // checkpoint は work item のファイルを毎回変更するとは限らないので、件名が名指す記録を使う。
  const named = checkpointed
    ? repo.workItemsNumbered(checkpointed).map((id) => repo.workItem(id)?.record)
    : []
  const committed = workItemIds(paths).map((id) => repo.workItemAt(sha, id))
  return {
    sha,
    subject,
    productionPaths: paths.filter((path) => isProductionCode(path, exclusions)),
    workItemDeclared: [...committed, ...named].some(
      (record) => isUnderWay(record) && record.declaration !== undefined,
    ),
    trailer: trailer ? trailer : undefined,
    diff: repo.diffOf(sha, paths),
    changedCharacterizations: characterizationChanges(
      paths,
      (path) => git(['show', `${sha}^:${path}`]),
      (path) => git(['show', `${sha}:${path}`]),
    ),
  }
}

export async function checkSpecImpact(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
  completing?: { path: string; text: string },
): Promise<CheckOutcome> {
  if (!options.baseRevision) {
    return { ok: false, lines: ['spec-impact requires --base-revision <git-revision>'] }
  }
  const git = gitIn(snapshot.root)
  const head = git(['rev-parse', '--verify', '--quiet', 'HEAD^{commit}'])?.trim()
  const resolved = git(['rev-parse', '--verify', '--quiet', `${options.baseRevision}^{commit}`])
  if (!head || !resolved) {
    return {
      ok: false,
      lines: [`spec-impact cannot resolve Git revision ${options.baseRevision}`],
    }
  }
  // 基準が先へ進んだブランチでは、基準側の変更が逆向きの差分として現れる。分岐点から比べる。
  const base = git(['merge-base', resolved.trim(), head])?.trim() || resolved.trim()
  const repo = repository(snapshot, git, completing)

  const ids = workItemIds([...repo.changedSince(base), ...(completing ? [completing.path] : [])])
  const completed: CompletedRange[] = []
  for (const id of ids) {
    const range = await completedRange(id, head, git, repo)
    if (range) completed.push(range)
  }
  const completedCommits = new Set(completed.flatMap((range) => range.commits))

  const items: CheckedWorkItem[] = []
  for (const id of ids) {
    const item = await checkedWorkItem(id, head, git, repo)
    if (item) items.push(item)
  }
  const commits: CheckedCommit[] = []
  for (const sha of lines(git(['rev-list', '--reverse', '--no-merges', `${base}..${head}`]))) {
    commits.push({
      ...(await checkedCommit(sha, git, repo)),
      workItemCompleted: completedCommits.has(sha),
    })
  }
  const uncompletedDiffs = commits
    .filter((commit) => !commit.workItemCompleted)
    .map((commit) => commit.diff)
  if (!completed.some((range) => range.end === undefined))
    uncompletedDiffs.push(await repo.diffSince(head))
  const findings = verifySpecImpact({
    items,
    commits,
    diff: await repo.diffSince(base),
    uncompletedDiffs,
  })
  if (findings.length > 0)
    return { ok: false, lines: findings.map((finding) => `fail  ${finding}`) }
  return {
    ok: true,
    lines: [
      `ok  spec impact (${items.length} work item(s), ${commits.length} commit(s) against ${options.baseRevision})`,
    ],
  }
}
