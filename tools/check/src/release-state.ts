import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'

export type ReleasePhase = 'unpublished' | 'published'

/** 状態を推測すると公開後の検査を免除できるため、明示した二値だけを受け付ける。 */
export async function readReleasePhase(snapshot: WorkspaceSnapshot): Promise<ReleasePhase> {
  const path = 'spec/release-state.json'
  const source = await snapshot.read(path)
  let state: unknown
  try {
    state = JSON.parse(source)
  } catch (error) {
    throw new Error(`${path}: invalid JSON`, { cause: error })
  }
  if (state !== null && typeof state === 'object' && !Array.isArray(state) && 'phase' in state) {
    if (state.phase === 'unpublished' || state.phase === 'published') return state.phase
  }
  throw new Error(`${path}: phase must be unpublished or published`)
}
