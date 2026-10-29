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
  isStarted,
  specImpactDeclaration,
  verifySpecImpact,
  workRangeStart,
} from './spec-impact.ts'
import { parseFrontmatterAndMarkdown } from './work-item-markdown.ts'

const WORK_ITEM = /^work-items\/(?:(?:active|done)\/)?(wi-[^/]+)\.md$/
const GO_TEST_FILE = /^backend\/.*_test\.go$/
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

type Commit = {
  sha: string
  subject: string
  /** `Spec-Impact` トレーラーの値。トレーラーがなければ undefined。 */
  trailer: string | undefined
  paths: string[]
}

/** `from` から HEAD までのマージでないコミットを古い順に、件名、トレーラー、パスとともに一回で読む。 */
function commitsSince(from: string, git: Git): Commit[] {
  const output = git([
    'log',
    '--reverse',
    '--no-merges',
    '--no-renames',
    '--name-only',
    '--format=%x1e%H%x1f%s%x1f%(trailers:key=Spec-Impact,valueonly,unfold)%x1f',
    `${from}..HEAD`,
  ])
  return (output ?? '')
    .split('\x1e')
    .slice(1)
    .map((record) => {
      const [sha = '', subject = '', trailer = '', paths] = record.split('\x1f')
      return { sha, subject, trailer: trailer.trim() || undefined, paths: lines(paths) }
    })
}

/**
 * 範囲の連続した一部。`to` が undefined なら作業ツリーまでを表す。
 * 仕様差分と特性化テストは、区間の両端の比較で一度だけ求める。
 */
type Segment = { from: string; to: string | undefined; paths: string[] }

/**
 * `from` から作業ツリーまでの範囲から、完了した作業の区間を除いた区間。
 * `uncommitted` は、コミットしていない変更が完了した作業に含まれないときの、その変更のパスである。
 */
function segments(
  from: string,
  commits: readonly Commit[],
  completed: ReadonlySet<string>,
  uncommitted: readonly string[] | undefined,
): Segment[] {
  const result: Segment[] = []
  let current: Segment = { from, to: from, paths: [] }
  for (const commit of commits) {
    if (completed.has(commit.sha)) {
      if (current.paths.length > 0) result.push(current)
      current = { from: commit.sha, to: commit.sha, paths: [] }
      continue
    }
    current.to = commit.sha
    current.paths.push(...commit.paths)
  }
  if (uncommitted !== undefined) {
    current.to = undefined
    current.paths.push(...uncommitted)
  }
  if (current.paths.length > 0) result.push(current)
  return result
}

/** `paths` のうち Go のテストファイルについて、`before` にあり `after` で変わったか消えた特性化テスト。 */
function characterizationChanges(
  paths: readonly string[],
  before: (path: string) => string | undefined,
  after: (path: string) => string | undefined,
): string[] {
  const tests = [...new Set(paths.filter((path) => GO_TEST_FILE.test(path)))]
  const addedSources = tests.filter((path) => before(path) === undefined).map(after)
  return tests.flatMap((path) => {
    const original = before(path)
    const current = after(path)
    // 移動先でも本文が一致する場合だけ、削除と追加を一つの移動として扱う。
    // 移動と同時の期待値の変更は、従来どおり拒否する。
    if (
      current === undefined &&
      original !== undefined &&
      addedSources.some(
        (source) => source !== undefined && changedCharacterizations(original, source).length === 0,
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

function workItemIds(paths: readonly string[]): string[] {
  return [...new Set(paths.flatMap((path) => path.match(WORK_ITEM)?.[1] ?? []))]
}

/**
 * 仕様影響の検査が Git と作業ツリーから読む値。
 *
 * 同じリビジョンの仕様と同じ起点からのコミットを、work item ごとに読み直さないよう記憶する。
 */
function repository(
  snapshot: WorkspaceSnapshot,
  git: Git,
  head: string,
  completing?: { path: string; text: string },
) {
  const revisions = new Map<string, Snapshot>()
  const commitLists = new Map<string, Commit[]>()
  const shown = new Map<string, string | undefined>()
  const histories = new Map<string, Array<{ sha: string; status: string | undefined }>>()
  let uncommitted: string[] | undefined
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
  const show = (object: string): string | undefined => {
    if (!shown.has(object)) shown.set(object, git(['show', object]))
    return shown.get(object)
  }
  /** `<rev>:<path>` の内容を一つの `git cat-file --batch` でまとめて読み、`show` の記憶へ入れる。 */
  const prefetch = (objects: readonly string[]): void => {
    const wanted = [...new Set(objects)].filter((object) => !shown.has(object))
    if (wanted.length === 0) return
    const result = Bun.spawnSync(['git', 'cat-file', '--batch'], {
      cwd: snapshot.root,
      stdin: Buffer.from(`${wanted.join('\n')}\n`),
    })
    if (result.exitCode !== 0) return
    const output = result.stdout
    let offset = 0
    for (const object of wanted) {
      const newline = output.indexOf(10, offset)
      if (newline < 0) return
      const header = output.subarray(offset, newline).toString()
      offset = newline + 1
      const size = Number(header.match(/^\S+ blob (\d+)$/)?.[1] ?? Number.NaN)
      if (Number.isNaN(size)) {
        // 存在しないパスは `<object> missing` と一行だけ返る。
        shown.set(object, undefined)
        continue
      }
      shown.set(object, output.subarray(offset, offset + size).toString())
      offset += size + 1
    }
  }
  const read = (path: string, at?: string): string | undefined =>
    at
      ? show(`${at}:${path}`)
      : path === completing?.path
        ? completing.text.replace(/^status:.*$/m, 'status: in_progress')
        : snapshot.exists(path)
          ? snapshot.readSync(path)
          : undefined
  const workItemAt = (
    id: string,
    at?: string,
  ): { path: string; record: WorkItemRecord } | undefined => {
    for (const path of workItemPaths(id)) {
      const record = workItemRecord(path, read(path, at))
      if (record) return { path, record }
    }
    return undefined
  }
  /** work item のファイルを変更したコミットと、その時点の状態を古い順に並べる。 */
  const history = (id: string) => {
    let value = histories.get(id)
    if (!value) {
      value = lines(git(['log', '--reverse', '--format=%H', '--', ...workItemPaths(id)])).map(
        (sha) => ({ sha, status: workItemAt(id, sha)?.record.status }),
      )
      histories.set(id, value)
    }
    return value
  }

  return {
    read,
    prefetch,
    workItemAt,
    /** 読み済みの一覧に `from` が含まれていれば、その後ろを切り出して Git を呼ばない。 */
    commitsSince: (from: string): Commit[] => {
      let value = commitLists.get(from)
      for (const known of value ? [] : commitLists.values()) {
        const index = known.findIndex((commit) => commit.sha === from)
        if (index >= 0) {
          value = known.slice(index + 1)
          break
        }
      }
      value ??= commitsSince(from, git)
      commitLists.set(from, value)
      return value
    },
    /** HEAD から作業ツリーまでに変わったパス。未追跡のファイルと完了中の記録を含む。 */
    uncommitted: (): string[] => {
      uncommitted ??= [
        ...lines(git(['diff', '--name-only', '--no-renames', head, '--'])),
        ...lines(git(['ls-files', '--others', '--exclude-standard'])),
        ...(completing ? [completing.path] : []),
      ]
      return uncommitted
    },
    /** 区間の両端の仕様差分。区間が仕様の原本を変えていなければ読まない。 */
    diffOf: async (segment: Segment): Promise<SpecificationDiff> => {
      if (!segment.paths.some(isSpecificationSource)) return NO_DIFF
      if (segment.to) return diffSpecifications(revision(segment.from), revision(segment.to))
      workingTree ??= readWorkingTree(snapshot.root)
      return diffSpecifications(revision(segment.from), await workingTree)
    },
    /** 区間の両端で比べた特性化テストの変更。 */
    characterizationsOf: (segment: Segment): string[] =>
      characterizationChanges(
        segment.paths,
        (path) => read(path, segment.from),
        (path) => read(path, segment.to),
      ),
    /** 作業範囲の起点のコミット。着手がコミットされていなければ HEAD。求められなければ undefined。 */
    workStart: (id: string): string | undefined => {
      const start = workRangeStart(history(id), workItemAt(id, head)?.record.status, head)
      return (
        (start && git(['rev-parse', '--verify', '--quiet', `${start}^{commit}`])?.trim()) ||
        undefined
      )
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

/** 完了した作業の区間。完了時に宣言を捨てるので、宣言は読み戻さず、どのコミットが含まれるかだけを持つ。 */
type CompletedWork = { commits: ReadonlySet<string>; uncommitted: boolean }

/**
 * `from` から作業ツリーまでの窓の中で、完了した work item の区間を求める。
 *
 * 区間は、状態が初めて pending でなくなったコミットから、初めて completed になったコミットまでである。
 * 窓の起点で既に着手していれば窓の先頭から、既に完了していれば窓の中に区間はない。
 * 窓の中の状態は、その記録を変えたコミットの内容をまとめて読んで求め、記録ごとに履歴を読まない。
 */
function completedWorks(
  ids: readonly string[],
  from: string,
  window: readonly Commit[],
  repo: Repository,
): CompletedWork[] {
  const candidates = ids.filter((id) => repo.workItemAt(id)?.record.status === 'completed')
  const touching = new Map<string, Array<{ sha: string; index: number }>>()
  window.forEach((commit, index) => {
    for (const id of workItemIds(commit.paths)) {
      touching.set(id, [...(touching.get(id) ?? []), { sha: commit.sha, index }])
    }
  })
  repo.prefetch(
    candidates.flatMap((id) =>
      [from, ...(touching.get(id) ?? []).map((event) => event.sha)].flatMap((sha) =>
        workItemPaths(id).map((path) => `${sha}:${path}`),
      ),
    ),
  )
  return candidates.flatMap((id) => {
    const before = repo.workItemAt(id, from)?.record.status
    if (before === 'completed') return []
    const events = (touching.get(id) ?? []).map((event) => ({
      ...event,
      status: repo.workItemAt(id, event.sha)?.record.status,
    }))
    const start = isStarted(before)
      ? 0
      : (events.find((event) => isStarted(event.status))?.index ?? window.length)
    const end = events.find((event) => event.index >= start && event.status === 'completed')
    return [
      {
        commits: new Set(window.slice(start, (end?.index ?? window.length) + 1).map((c) => c.sha)),
        uncommitted: end === undefined,
      },
    ]
  })
}

/** 進行中で宣言を持つ work item と、その作業範囲の起点とコミット。 */
type ActiveWork = {
  id: string
  path: string
  declaration: SpecImpactDeclaration
  start: string | undefined
  commits: readonly Commit[]
}

function activeWork(id: string, repo: Repository): ActiveWork | undefined {
  const current = repo.workItemAt(id)
  if (!current || !isUnderWay(current.record) || !current.record.declaration) return undefined
  const start = repo.workStart(id)
  return {
    id,
    path: current.path,
    declaration: current.record.declaration,
    start,
    commits: start ? repo.commitsSince(start) : [],
  }
}

function affectedBy(declaration: SpecImpactDeclaration): SpecReference[] {
  return declaration.kind === 'affected' ? declaration.references : []
}

/** 区間ごとの仕様差分と、範囲の起点に存在した特性化テストの変更。 */
async function rangeChanges(
  start: string,
  commits: readonly Commit[],
  completed: readonly CompletedWork[],
  repo: Repository,
): Promise<{ diffs: SpecificationDiff[]; changedCharacterizations: string[] }> {
  const parts = segments(
    start,
    commits,
    new Set(completed.flatMap((work) => [...work.commits])),
    completed.some((work) => work.uncommitted) ? undefined : repo.uncommitted(),
  )
  const diffs = await Promise.all(parts.map(repo.diffOf))
  // 範囲の中で足した特性化テストは、範囲の起点にあった名前に含まれないので数えない。
  const changed = [...new Set(parts.flatMap(repo.characterizationsOf))]
  const existing = new Set(
    characterizationChanges(
      parts.flatMap((part) => part.paths),
      (path) => repo.read(path, start),
      () => undefined,
    ).map((change) => change.slice(change.lastIndexOf(' ') + 1)),
  )
  return {
    diffs,
    changedCharacterizations: changed.filter((change) =>
      existing.has(change.slice(change.lastIndexOf(' ') + 1)),
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
  const repo = repository(snapshot, git, head, completing)

  const baseCommits = repo.commitsSince(base)
  const pathsOf = (commits: readonly Commit[]) => [
    ...commits.flatMap((commit) => commit.paths),
    ...repo.uncommitted(),
  ]
  const active = workItemIds(pathsOf(baseCommits)).flatMap((id) => activeWork(id, repo) ?? [])
  // 作業範囲は基準より前から始まることがあるので、最も古い起点からの窓で完了した作業を集める。
  // 線形の履歴では、起点の共通祖先が最も古い起点である。
  const starts = active.flatMap((work) => work.start ?? [])
  const from =
    (starts.length > 0 && git(['merge-base', '--octopus', base, ...starts])?.trim()) || base
  const window = repo.commitsSince(from)
  const completed = completedWorks(workItemIds(pathsOf(window)), from, window, repo)
  const completedIn = (commits: readonly Commit[]): CompletedWork[] =>
    completed.filter(
      (work) => work.uncommitted || commits.some((commit) => work.commits.has(commit.sha)),
    )

  const items: CheckedWorkItem[] = []
  for (const work of active) {
    const item = { path: work.path, declaration: work.declaration }
    if (!work.start) {
      items.push({ ...item, range: undefined })
      continue
    }
    items.push({
      ...item,
      range: {
        ...(await rangeChanges(work.start, work.commits, completedIn(work.commits), repo)),
        claimedByOthers: active
          .filter((other) => other.id !== work.id)
          .flatMap((other) => affectedBy(other.declaration)),
      },
    })
  }

  // 作業範囲と完了した作業の区間のコミットは、その work item の宣言に従う。
  // それ以外のコミットだけを一つずつ読み、トレーラーで判定する。
  const covered = new Set([
    ...active.flatMap((work) => work.commits.map((commit) => commit.sha)),
    ...completed.flatMap((work) => [...work.commits]),
  ])
  const exclusions = await repo.productionCodeExclusions()
  const commits: CheckedCommit[] = []
  for (const commit of baseCommits.filter((commit) => !covered.has(commit.sha))) {
    const segment = { from: `${commit.sha}^`, to: commit.sha, paths: commit.paths }
    commits.push({
      sha: commit.sha,
      subject: commit.subject,
      productionPaths: commit.paths.filter((path) => isProductionCode(path, exclusions)),
      trailer: commit.trailer,
      diff: await repo.diffOf(segment),
      changedCharacterizations: repo.characterizationsOf(segment),
    })
  }

  const diffs = (await rangeChanges(base, baseCommits, completedIn(baseCommits), repo)).diffs
  const findings = verifySpecImpact({ items, commits, diffs })
  if (findings.length > 0)
    return { ok: false, lines: findings.map((finding) => `fail  ${finding}`) }
  return {
    ok: true,
    lines: [
      `ok  spec impact (${items.length} work item(s), ${baseCommits.length} commit(s) against ${options.baseRevision})`,
    ],
  }
}
