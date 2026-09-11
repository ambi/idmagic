import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { GoFile } from './contract-drift.ts'

const nonTestGoFiles = new WeakMap<WorkspaceSnapshot, Promise<GoFile[]>>()
const allGoFiles = new WeakMap<WorkspaceSnapshot, Promise<GoFile[]>>()

async function loadGoFiles(snapshot: WorkspaceSnapshot, includeTests: boolean): Promise<GoFile[]> {
  const paths = (await snapshot.files('backend', ['vendor', 'dist', 'build', 'generated'])).filter(
    (path) => path.endsWith('.go') && (includeTests || !path.endsWith('_test.go')),
  )
  return Promise.all(paths.map(async (path) => ({ path, source: await snapshot.read(path) })))
}

/** 複数の規則が使う Go ソース集合を snapshot ごとに一度だけ組み立てる。 */
export function repositoryGoFiles(
  snapshot: WorkspaceSnapshot,
  includeTests = false,
): Promise<GoFile[]> {
  const cache = includeTests ? allGoFiles : nonTestGoFiles
  let files = cache.get(snapshot)
  if (!files) {
    files = loadGoFiles(snapshot, includeTests)
    cache.set(snapshot, files)
  }
  return files
}
