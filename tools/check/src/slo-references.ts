/**
 * 監視資材が名指しするサービス目標の ID が実在し、page 級のアラートが到達
 * できる runbook を持つことを確かめる。
 *
 * しきい値の数値そのものは照合しない。アラートが判定するのは error budget の
 * 消費速度であり、目標は 30 日の移動窓で評価する別のものだからである。同じ数を
 * 使っているのは現在の設計判断にすぎず、一致を強制すると、窓を変えたときに
 * 検査を通すためだけに目標を動かす誘因が生まれる。
 */

/** 目標の宣言。`docs/capacity.md` の表の 1 行に対応する。 */
export interface Objective {
  id: string
}

/** 監視資材の中で見つかった、目標 ID への言及または runbook への参照。 */
export interface AlertReference {
  alert: string
  severity: string
  objectives: string[]
  runbook?: string
}

export interface Finding {
  path: string
  message: string
}

const OBJECTIVE_ID = /\b((?:SLO|CAP)-[A-Z0-9-]+)\b/g

/** 目標の表から ID を集める。表の行頭に置かれたものだけを宣言とみなす。 */
export function declaredObjectives(source: string): Set<string> {
  const ids = new Set<string>()
  for (const line of source.split('\n')) {
    const cell = line.match(/^\|\s*((?:SLO|CAP)-[A-Z0-9-]+)\s*\|/)
    if (cell?.[1]) ids.add(cell[1])
  }
  return ids
}

/**
 * Prometheus のルール定義からアラートを読む。YAML として解析せず行を追うのは、
 * 資材が 2 つの書式（ブロックとフロー）で annotations を書いており、どちらでも
 * 同じ規則を当てたいためである。
 */
export function alertReferences(source: string): AlertReference[] {
  const alerts: AlertReference[] = []
  let current: AlertReference | undefined
  for (const line of source.split('\n')) {
    const name = line.match(/^\s*-\s*alert:\s*(\S+)/)
    if (name?.[1]) {
      current = { alert: name[1], severity: '', objectives: [] }
      alerts.push(current)
      continue
    }
    if (!current) continue
    const severity = line.match(/severity:\s*([A-Za-z0-9_-]+)/)
    if (severity?.[1]) current.severity = severity[1]
    const runbook = line.match(/runbook_url:\s*"?([^"\s}]+)"?/)
    if (runbook?.[1]) current.runbook = runbook[1]
    for (const [, id] of line.matchAll(OBJECTIVE_ID)) if (id) current.objectives.push(id)
  }
  return alerts
}

/**
 * 名指しした ID が実在すること、`page` のアラートが runbook を持つことを求める。
 * 目標に由来しないアラート（スロットルの発動率、ジョブの滞留）は ID を名乗らない
 * ので、名乗ることは求めない。
 */
export function checkAlerts(
  path: string,
  alerts: readonly AlertReference[],
  declared: ReadonlySet<string>,
  runbookExists: (relativePath: string) => boolean,
): Finding[] {
  const findings: Finding[] = []
  for (const alert of alerts) {
    for (const id of alert.objectives) {
      if (!declared.has(id)) {
        findings.push({ path, message: `${alert.alert} names ${id}, which no objective declares` })
      }
    }
    if (alert.severity === 'page' && !alert.runbook) {
      findings.push({ path, message: `${alert.alert} is severity page and has no runbook_url` })
    }
    if (alert.runbook && !runbookExists(alert.runbook)) {
      findings.push({
        path,
        message: `${alert.alert} points at a missing runbook: ${alert.runbook}`,
      })
    }
  }
  return findings
}
