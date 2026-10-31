import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { COMPOSITION_NAME, parseBoundaryDebt } from './boundary-fitness.ts'
import {
  analyzeModuleBoundaries,
  BOUNDARY_DEBT_PATH,
  RESPONSIBILITY_TABLE_PATH,
} from './check-boundaries.ts'
import { moduleDependencyMatrix, moduleGroups } from './module-dependency-diagram.ts'
import type { CheckOutcome } from './runner.ts'

/** 責務表と境界検査の参照に、人が表へ記した関係と負債の区別を加える。 */
export async function generatedModuleDependencies(snapshot: WorkspaceSnapshot): Promise<string> {
  const analysis = await analyzeModuleBoundaries(snapshot)
  const source = await snapshot.read(RESPONSIBILITY_TABLE_PATH)
  const debt = parseBoundaryDebt(await snapshot.read(BOUNDARY_DEBT_PATH)).flatMap((entry) =>
    entry.targetModule
      ? [{ sourceModule: entry.sourceModule, targetModule: entry.targetModule }]
      : [],
  )
  return moduleDependencyMatrix(
    moduleGroups(source, COMPOSITION_NAME),
    analysis.dependencies,
    source,
    debt,
  )
}

/** 生成図の入力となる関係の表で、未知のモジュールと重複を拒否する。 */
export async function checkModuleDependencies(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  if (!snapshot.exists('backend'))
    return { ok: true, lines: ['ok  module dependency relationships'] }
  await generatedModuleDependencies(snapshot)
  return { ok: true, lines: ['ok  module dependency relationships'] }
}
