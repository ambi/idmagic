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
  specImpactDeclaration,
  verifySpecImpact,
  workRangeStart,
} from './spec-impact.ts'
import { parseFrontmatterAndMarkdown } from './work-item-markdown.ts'

const WORK_ITEM = /^work-items\/(?:done\/)?(wi-[^/]+)\.md$/
const TEST_FILE = /^(?:backend|frontend)\/.*(?:_test\.go|\.(?:test|spec)\.tsx?)$/
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
  return record?.status === 'in_progress' || record?.status === 'completed'
}

/** work item は、完了時に `work-items/done/` へ移る。どちらにあっても同じ記録として読む。 */
function workItemPaths(id: string): string[] {
  return [`work-items/${id}.md`, `work-items/done/${id}.md`]
}

/**
 * 仕様影響の検査が Git と作業ツリーから読む値。
 *
 * 同じリビジョンの仕様と同じ範囲の変更一覧を、work item ごとに読み直さないよう記憶する。
 */
function repository(snapshot: WorkspaceSnapshot, git: Git) {
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
  /** `sha` から作業ツリーまでに追加、変更、削除されたパス。未追跡のファイルを含む。 */
  const changedSince = (sha: string): string[] => {
    let value = changes.get(sha)
    if (!value) {
      value = [
        ...lines(git(['diff', '--name-only', '--no-renames', sha, '--'])),
        ...lines(git(['ls-files', '--others', '--exclude-standard'])),
      ]
      changes.set(sha, value)
    }
    return value
  }
  const read = (path: string): string | undefined =>
    snapshot.exists(path) ? snapshot.readSync(path) : undefined

  return {
    read,
    changedSince,
    /** `sha` から作業ツリーまでの仕様差分。仕様のファイルが変わっていなければ読まない。 */
    diffSince: async (sha: string): Promise<SpecificationDiff> => {
      if (!changedSince(sha).some(isSpecificationSource)) return NO_DIFF
      workingTree ??= readWorkingTree(snapshot.root)
      return diffSpecifications(revision(sha), await workingTree)
    },
    /** コミット一つが加えた仕様差分。 */
    diffOf: (sha: string, paths: readonly string[]): SpecificationDiff => {
      if (!paths.some(isSpecificationSource)) return NO_DIFF
      const parent = git(['rev-parse', '--verify', '--quiet', `${sha}^`])?.trim()
      return diffSpecifications(parent ? revision(parent) : new Map(), revision(sha))
    },
    workItem: (id: string): { path: string; record: WorkItemRecord } | undefined => {
      for (const path of workItemPaths(id)) {
        const record = workItemRecord(path, read(path))
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
  base: string,
  head: string,
  git: Git,
  repo: Repository,
): Promise<CheckedWorkItem | undefined> {
  const current = repo.workItem(id)
  if (!current || !isUnderWay(current.record) || !current.record.declaration) return undefined
  // 基準の時点で完了していた記録は、参照先の書き換えで変わっても過去の作業を再検査しない。
  if (current.record.status === 'completed' && repo.workItemAt(base, id)?.status === 'completed') {
    return undefined
  }
  const history = lines(git(['log', '--reverse', '--format=%H', '--', ...workItemPaths(id)])).map(
    (sha) => ({ sha, status: repo.workItemAt(sha, id)?.status }),
  )
  const start = workRangeStart(history, repo.workItemAt(head, id)?.status, head)
  const startSha = start && git(['rev-parse', '--verify', '--quiet', `${start}^{commit}`])?.trim()
  const item = { path: current.path, declaration: current.record.declaration }
  if (!startSha) return { ...item, range: undefined }

  const changed = repo.changedSince(startSha)
  return {
    ...item,
    range: {
      diff: await repo.diffSince(startSha),
      claimedByOthers: workItemIds(changed)
        .filter((other) => other !== id)
        .flatMap((other) => {
          const record = repo.workItem(other)?.record
          return isUnderWay(record) ? modifiedBy(record.declaration) : []
        }),
      changedTests: changed
        .filter((path) => TEST_FILE.test(path))
        .flatMap((path) => repo.read(path) ?? []),
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
    diff: trailer ? repo.diffOf(sha, paths) : NO_DIFF,
  }
}

export async function checkSpecImpact(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
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
  const repo = repository(snapshot, git)

  const items: CheckedWorkItem[] = []
  for (const id of workItemIds(repo.changedSince(base))) {
    const item = await checkedWorkItem(id, base, head, git, repo)
    if (item) items.push(item)
  }
  const commits: CheckedCommit[] = []
  for (const sha of lines(git(['rev-list', '--reverse', '--no-merges', `${base}..${head}`]))) {
    commits.push(await checkedCommit(sha, git, repo))
  }
  const findings = verifySpecImpact({ items, commits, diff: await repo.diffSince(base) })
  if (findings.length > 0)
    return { ok: false, lines: findings.map((finding) => `fail  ${finding}`) }
  return {
    ok: true,
    lines: [
      `ok  spec impact (${items.length} work item(s), ${commits.length} commit(s) against ${options.baseRevision})`,
    ],
  }
}
