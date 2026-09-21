import type { Finding } from './lib.ts'

export type WorkItemDependencyRecord = {
  id: string
  path: string
  depends_on: string[]
  depends_on_line?: number
}

export type WorkItemDependencyFinding = Finding & { path: string }

export type IdentifierRange = { min: number; max: number }

/**
 * 新規の作業項目が識別番号を選ぶ範囲。
 *
 * 既存の記録は `wi-1` から `wi-634` までの 3 桁以下に収まっているので、
 * 5 桁に限れば番号空間が重ならない。既存のファイル名、`depends_on`、リリース
 * 文書名、ブランチ名を一つも動かさずに採番規則を差し替えられるのはこのためである。
 */
export const IDENTIFIER_RANGE: IdentifierRange = { min: 10_000, max: 99_999 }

/** 空き枠がこれを下回ったら、桁を増やす判断に要る観測として報告する。 */
export const CAPACITY_WARNING_THRESHOLD = 10_000

export type WorkItemIdentifierReport = {
  findings: WorkItemDependencyFinding[]
  warnings: string[]
}

/** 記録の id から識別番号を読む。id はファイル名の stem なので、先頭が番号である。 */
export function workItemNumber(id: string): number | undefined {
  const digits = id.match(/^wi-(\d+)-/)?.[1]
  return digits === undefined ? undefined : Number(digits)
}

/**
 * 記録の集合を識別番号の側から見る。
 *
 * 番号の衝突は id の重複とは別物である。id は拡張子を除いたファイル名全体なので、
 * 題名が違えば `verifyWorkItemDependencies` は別の記録として通す。番号は
 * `depends_on`、リリース文書名、ブランチ名、ワークツリー名にも現れるため、衝突を
 * 事後に改名で直すにはそれらを同時に張り替えなければならない。統合より前に止める。
 * 例外は置かない。連番で生まれた 3 組は、この検査を入れるのと同じ変更で解消した。
 *
 * 空き枠の警告を所見と同じ呼び出しから返すのは、`checkWorkItems` から見た配線を
 * 一箇所に保つためである。閾値を割るには 80000 件を超える記録が要り、実ファイルを
 * 並べる検査では起こせない。
 */
export function verifyWorkItemIdentifiers(
  records: readonly WorkItemDependencyRecord[],
): WorkItemIdentifierReport {
  const byNumber = new Map<number, WorkItemDependencyRecord[]>()
  for (const record of records) {
    const number = workItemNumber(record.id)
    if (number === undefined) continue
    const group = byNumber.get(number)
    if (group) group.push(record)
    else byNumber.set(number, [record])
  }

  const findings: WorkItemDependencyFinding[] = []
  let usedInRange = 0
  for (const [number, group] of byNumber) {
    if (number >= IDENTIFIER_RANGE.min && number <= IDENTIFIER_RANGE.max) usedInRange++
    // id 順に並べてから組を作る。報告の順序と、報告する位置が入力順で変わると、
    // 同じ衝突でも実行ごとに違う行が出る。
    const sorted = [...group].sort((one, other) => one.id.localeCompare(other.id))
    for (let i = 0; i < sorted.length; i++) {
      for (let j = i + 1; j < sorted.length; j++) {
        const first = sorted[i]
        const second = sorted[j]
        if (!first || !second) continue
        findings.push({
          path: second.path,
          line: 1,
          column: 1,
          message: `work-item identifier: ${number} is shared by '${first.id}' and '${second.id}'`,
        })
      }
    }
  }

  const capacity = IDENTIFIER_RANGE.max - IDENTIFIER_RANGE.min + 1
  const remaining = capacity - usedInRange
  return {
    findings,
    warnings:
      remaining < CAPACITY_WARNING_THRESHOLD
        ? [
            `work-item identifier capacity: ${remaining} of ${capacity} five-digit numbers remain, below the ${CAPACITY_WARNING_THRESHOLD} warning threshold`,
          ]
        : [],
  }
}

/**
 * Verify the workspace-wide prerequisite graph. A dependency means that the
 * current work item cannot be completed before its target is completed.
 */
export function verifyWorkItemDependencies(
  records: WorkItemDependencyRecord[],
): WorkItemDependencyFinding[] {
  const findings: WorkItemDependencyFinding[] = []
  const byId = new Map<string, WorkItemDependencyRecord>()
  for (const record of records) {
    const previous = byId.get(record.id)
    if (previous) {
      findings.push({
        path: record.path,
        line: 1,
        column: 1,
        message: `duplicate work item '${record.id}'; also declared by ${previous.path}`,
      })
      continue
    }
    byId.set(record.id, record)
  }
  const edges = new Map<string, string[]>()

  for (const record of byId.values()) {
    const targets: string[] = []
    for (const target of record.depends_on) {
      if (target === record.id) {
        findings.push({
          path: record.path,
          line: record.depends_on_line ?? 1,
          column: 1,
          message: `work-item dependency: '${record.id}' must not depend on itself`,
        })
        continue
      }
      if (!byId.has(target)) {
        findings.push({
          path: record.path,
          line: record.depends_on_line ?? 1,
          column: 1,
          message: `work-item dependency: '${record.id}' references unknown work item '${target}'`,
        })
        continue
      }
      targets.push(target)
    }
    edges.set(record.id, targets)
  }

  const cycle = findCycle(edges)
  if (cycle) {
    const source = byId.get(cycle[0] ?? '')
    if (source) {
      findings.push({
        path: source.path,
        line: source.depends_on_line ?? 1,
        column: 1,
        message: `work-item dependency cycle detected: ${cycle.join(' -> ')}`,
      })
    }
  }
  return findings
}

function findCycle(edges: Map<string, string[]>): string[] | null {
  const visiting = new Set<string>()
  const visited = new Set<string>()
  const stack: string[] = []

  const visit = (id: string): string[] | null => {
    visiting.add(id)
    stack.push(id)
    for (const next of edges.get(id) ?? []) {
      if (visiting.has(next)) return [...stack.slice(stack.indexOf(next)), next]
      if (!visited.has(next)) {
        const cycle = visit(next)
        if (cycle) return cycle
      }
    }
    stack.pop()
    visiting.delete(id)
    visited.add(id)
    return null
  }

  for (const id of edges.keys()) {
    if (visited.has(id)) continue
    const cycle = visit(id)
    if (cycle) return cycle
  }
  return null
}
