/** mise.toml のタスク変更から、検査の閉包が届く層を選ぶ。 */

import { isDeepStrictEqual } from 'node:util'
import { type MiseTasks, taskClosure } from '../../check/src/verification-tasks.ts'

export type Layer = 'go' | 'ui' | 'tools' | 'audit'

export type MiseChange = {
  /** タスク以外の変更や読み取り失敗により、全層へ影響する。 */
  all: boolean
  reasons: string[]
  /** all が false のとき、変更タスクに到達する検査の層。 */
  layers: Layer[]
}

/** verify-changed が各層で起動する検査の起点。 */
const ROOTS: Record<Layer, string[]> = {
  go: ['lint-go', 'test-go'],
  ui: ['verify-ui'],
  tools: ['test-tools', 'typecheck-tools', 'lint-tools'],
  audit: ['audit-dependencies'],
}

type Document = Record<string, unknown> & { tasks?: MiseTasks }

function parse(source: string): Document | undefined {
  try {
    return Bun.TOML.parse(source) as Document
  } catch {
    return undefined
  }
}

export function miseChange(before: string, after: string): MiseChange {
  const old = parse(before)
  const current = parse(after)
  if (!old || !current)
    return { all: true, reasons: ['mise.toml cannot be read on both sides'], layers: [] }

  const tables = [...new Set([...Object.keys(old), ...Object.keys(current)])]
    .filter((key) => key !== 'tasks' && !isDeepStrictEqual(old[key], current[key]))
    .sort()
  if (tables.length > 0) {
    return {
      all: true,
      reasons: [
        `mise.toml changes ${tables.map((key) => `[${key}]`).join(', ')}, which may affect every layer`,
      ],
      layers: [],
    }
  }

  const oldTasks = old.tasks ?? {}
  const currentTasks = current.tasks ?? {}
  const changed = [...new Set([...Object.keys(oldTasks), ...Object.keys(currentTasks)])].filter(
    (name) => !isDeepStrictEqual(oldTasks[name], currentTasks[name]),
  )
  // 追加と削除を同じ規則で判定できるよう、変更前後の閉包の和集合を見る。
  const reaches = (layer: Layer) =>
    ROOTS[layer].some((root) => {
      const before = taskClosure(oldTasks, root)
      const after = taskClosure(currentTasks, root)
      return changed.some((name) => before.has(name) || after.has(name))
    })
  return {
    all: false,
    reasons: [],
    layers: (Object.keys(ROOTS) as Layer[]).filter(reaches),
  }
}
