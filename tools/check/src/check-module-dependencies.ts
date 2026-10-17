import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { COMPOSITION_NAME } from './boundary-fitness.ts'
import { analyzeModuleBoundaries, RESPONSIBILITY_TABLE_PATH } from './check-boundaries.ts'
import {
  moduleDependencyDiagram,
  moduleGroups,
  withModuleDependencyDiagram,
} from './module-dependency-diagram.ts'
import type { CheckOutcome } from './runner.ts'

/** モジュール設計文書の入口の依存の図を、現在の責務表と import の辺から作り直す。 */
export async function generatedModuleDependencies(
  snapshot: WorkspaceSnapshot,
): Promise<{ source: string; generated: string | undefined }> {
  const analysis = await analyzeModuleBoundaries(snapshot)
  const source = await snapshot.read(RESPONSIBILITY_TABLE_PATH)
  const generated = withModuleDependencyDiagram(
    source,
    moduleDependencyDiagram(moduleGroups(source, COMPOSITION_NAME), analysis.dependencies),
  )
  return { source, generated }
}

/** 入口の依存の図が、作り直した図と一致することを確かめる。 */
export async function checkModuleDependencies(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  if (!snapshot.exists('backend')) return { ok: true, lines: ['ok  module dependency diagram'] }
  const { source, generated } = await generatedModuleDependencies(snapshot)
  if (generated === undefined) {
    return {
      ok: false,
      lines: [`${RESPONSIBILITY_TABLE_PATH}: the module dependency diagram markers are missing`],
    }
  }
  return generated === source
    ? { ok: true, lines: ['ok  module dependency diagram'] }
    : {
        ok: false,
        lines: [
          `${RESPONSIBILITY_TABLE_PATH}: the module dependency diagram is stale; run mise run generate-module-dependencies`,
        ],
      }
}
