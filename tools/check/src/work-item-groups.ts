/** 未完了の作業項目の `group` と、グループを宣言する索引の照合。 */

/** グループを宣言する索引の位置。索引のない workspace はグループを使わない。 */
export const WORK_ITEM_GROUP_INDEX = 'docs/development/work-item-groups.md'

export type WorkItemGroupIndex = { groups: string[]; findings: string[] }

/** 表の行のうち、最初のセルがコードで書いたケバブケースの名前の行をグループとして読む。 */
const GROUP_ROW = /^\|\s*`([a-z][a-z0-9-]*)`\s*\|/

export function parseWorkItemGroups(markdown: string): WorkItemGroupIndex {
  const groups: string[] = []
  const findings: string[] = []
  for (const line of markdown.split('\n')) {
    const name = GROUP_ROW.exec(line)?.[1]
    if (name === undefined) continue
    if (groups.includes(name)) findings.push(`duplicate work item group '${name}'`)
    else groups.push(name)
  }
  if (groups.length === 0) findings.push('the index declares no work item group')
  return { groups, findings }
}

type GroupedRecord = { path: string; data?: Record<string, unknown> }

const UNFINISHED = new Set(['pending', 'in_progress'])

/**
 * 未完了の記録がそれぞれ索引にあるグループを一つ持つかを確かめる。
 *
 * 完了と中止の記録は状態とタイトルと要約へ縮めるので、グループを持たない。
 * `groups` が `undefined` なのは索引のない workspace であり、何も求めない。
 */
export function verifyWorkItemGroups(
  records: GroupedRecord[],
  groups: string[] | undefined,
  indexPath: string,
): string[] {
  if (groups === undefined) return []
  const findings: string[] = []
  for (const record of records) {
    if (!UNFINISHED.has(String(record.data?.status))) continue
    const group = record.data?.group
    if (group === undefined) {
      findings.push(`${record.path}: missing group; choose one listed in ${indexPath}`)
    } else if (typeof group !== 'string' || !groups.includes(group)) {
      findings.push(
        `${record.path}: unknown group '${String(group)}'; choose one listed in ${indexPath}`,
      )
    }
  }
  return findings
}
