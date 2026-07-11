import type { Finding } from './lib.ts'

export type WorkItemDependencyRecord = {
  id: string
  path: string
  depends_on: string[]
  depends_on_line?: number
}

export type WorkItemDependencyFinding = Finding & { path: string }

/**
 * Verify the workspace-wide prerequisite graph. A dependency means that the
 * current work item cannot be completed before its target is completed.
 */
export function verifyWorkItemDependencies(
  records: WorkItemDependencyRecord[],
): WorkItemDependencyFinding[] {
  const findings: WorkItemDependencyFinding[] = []
  const byId = new Map(records.map((record) => [record.id, record]))
  const edges = new Map<string, string[]>()

  for (const record of records) {
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
