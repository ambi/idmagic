/**
 * コミットの `Agent-Friction:` トレーラーを対象ごとに集計する。
 *
 * エージェントのセッションは前のセッションを覚えていないので、同じ文書やタスクで
 * 繰り返し迷っているかは、コミットに残した記録を数えて初めて分かる。
 * Git とファイルシステムは読まず、入口が渡す値だけから区分を決める。
 */

const SEPARATOR = ' — '
const TASK_PREFIX = 'mise run '

export type FrictionTarget =
  | { kind: 'path'; path: string; anchor?: string }
  | { kind: 'task'; task: string }

export type FrictionEntry = { sha: string; date: string; target: FrictionTarget; detail: string }

/** 対象と起きたことに分けられなかった値。捨てると書いた記録が黙って消えるので残す。 */
export type MalformedEntry = { sha: string; date: string; value: string }

export type TargetSummary = { label: string; entries: FrictionEntry[] }

export type FrictionReport = {
  recurring: TargetSummary[]
  once: TargetSummary[]
  missing: TargetSummary[]
  malformed: MalformedEntry[]
}

export function parseFrictionTrailer(
  sha: string,
  date: string,
  value: string,
): FrictionEntry | MalformedEntry {
  const index = value.indexOf(SEPARATOR)
  const malformed = { sha, date, value }
  if (index < 0) return malformed
  const target = parseTarget(value.slice(0, index).trim())
  const detail = value.slice(index + SEPARATOR.length).trim()
  if (!target || detail === '') return malformed
  return { sha, date, target, detail }
}

function parseTarget(text: string): FrictionTarget | undefined {
  // 呼び出し元が前後の空白を削るので、タスク名のない `mise run` は接頭辞の空白を失って届く。
  if (text === TASK_PREFIX.trim() || text.startsWith(TASK_PREFIX)) {
    const task = text.slice(TASK_PREFIX.length).trim()
    return task === '' ? undefined : { kind: 'task', task }
  }
  if (text === '') return undefined
  const hash = text.indexOf('#')
  if (hash < 0) return { kind: 'path', path: text }
  const path = text.slice(0, hash)
  const anchor = text.slice(hash + 1)
  if (path === '') return undefined
  return anchor === '' ? { kind: 'path', path } : { kind: 'path', path, anchor }
}

/** 集計で同じ対象として数える表記。アンカーが違えば別の対象とする。 */
export function targetLabel(target: FrictionTarget): string {
  if (target.kind === 'task') return `${TASK_PREFIX}${target.task}`
  return target.anchor === undefined ? target.path : `${target.path}#${target.anchor}`
}

export function summarizeFriction(
  entries: (FrictionEntry | MalformedEntry)[],
  exists: (target: FrictionTarget) => boolean,
): FrictionReport {
  const byLabel = new Map<string, TargetSummary>()
  const malformed: MalformedEntry[] = []
  for (const entry of entries) {
    if (!('target' in entry)) {
      malformed.push(entry)
      continue
    }
    const label = targetLabel(entry.target)
    const summary = byLabel.get(label) ?? { label, entries: [] }
    summary.entries.push(entry)
    byLabel.set(label, summary)
  }

  const report: FrictionReport = { recurring: [], once: [], missing: [], malformed }
  for (const summary of byLabel.values()) {
    const target = summary.entries[0]?.target
    if (target && !exists(target)) report.missing.push(summary)
    else if (summary.entries.length >= 2) report.recurring.push(summary)
    else report.once.push(summary)
  }
  report.recurring.sort(
    (a, b) => b.entries.length - a.entries.length || a.label.localeCompare(b.label),
  )
  return report
}

/**
 * `git log` の一コミット分。`frictions` は `Agent-Friction:`、`reviewed` は
 * `Agent-Friction-Reviewed:` トレーラーの値で、後者は振り返りで判断した対象を一件ずつ示す。
 */
export type LogRecord = { sha: string; date: string; frictions: string[]; reviewed: string[] }

/**
 * 新しい順の履歴から、まだ判断していない記録を古い順に取り出す。
 *
 * 判断した対象は、その振り返りより前の記録を数え直すと、直した後も繰り返しとして残り続ける。
 * 判断しなかった対象は、1 件だけの記録が次の記録と合わさって初めて繰り返しになるので、全履歴から数える。
 * 振り返りのコミット自身の記録はその振り返りの作業中に起きたもので、まだ判断していないので含める。
 */
export function unreviewedEntries(newestFirst: LogRecord[]): (FrictionEntry | MalformedEntry)[] {
  const reviewed = new Set<string>()
  const entries: (FrictionEntry | MalformedEntry)[] = []
  for (const record of newestFirst) {
    for (const value of record.frictions) {
      const entry = parseFrictionTrailer(record.sha, record.date, value)
      if (!('target' in entry) || !reviewed.has(targetLabel(entry.target))) entries.push(entry)
    }
    for (const value of record.reviewed) {
      const target = parseTarget(value)
      // 対象として読めない目印は、どの記録を外すかを決められない。黙って捨てずに形式外として示す。
      if (target) reviewed.add(targetLabel(target))
      else
        entries.push({
          sha: record.sha,
          date: record.date,
          value: `Agent-Friction-Reviewed: ${value}`,
        })
    }
  }
  return entries.reverse()
}

export function formatFrictionReport(report: FrictionReport): string {
  const lines: string[] = []
  const section = (title: string, body: string[]) => {
    lines.push(`## ${title}`, '', ...(body.length === 0 ? ['none'] : body), '')
  }
  const summaries = (items: TargetSummary[]) =>
    items.flatMap((summary) => [
      `- ${summary.label}: ${summary.entries.length}`,
      ...summary.entries.map(
        (entry) => `  - ${entry.date} ${entry.sha.slice(0, 7)} ${entry.detail}`,
      ),
    ])

  section('Recurring (2 or more)', summaries(report.recurring))
  section('Once', summaries(report.once))
  section('Missing from the current tree', summaries(report.missing))
  section(
    'Malformed',
    report.malformed.map((entry) => `- ${entry.date} ${entry.sha.slice(0, 7)} ${entry.value}`),
  )
  return lines.join('\n')
}
