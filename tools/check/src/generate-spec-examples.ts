import { writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { updateExampleBlocks } from './executable-examples.ts'

const snapshot = createWorkspaceSnapshot()
for (const path of (await snapshot.files('docs')).filter((path) =>
  path.endsWith('examples.feature.md'),
)) {
  const source = await snapshot.read(path)
  const generated = updateExampleBlocks(source, (path) =>
    snapshot.exists(path) ? snapshot.readSync(path) : undefined,
  )
  if (generated !== source) {
    await writeFile(resolve(snapshot.root, path), generated)
    console.log(`generated ${path}`)
  }
}
