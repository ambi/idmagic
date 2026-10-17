import { writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { RESPONSIBILITY_TABLE_PATH } from './check-boundaries.ts'
import { generatedModuleDependencies } from './check-module-dependencies.ts'

const snapshot = createWorkspaceSnapshot()
const { source, generated } = await generatedModuleDependencies(snapshot)
if (generated === undefined) {
  console.error(`${RESPONSIBILITY_TABLE_PATH}: the module dependency diagram markers are missing`)
  process.exit(1)
}
if (generated !== source) {
  await writeFile(resolve(snapshot.root, RESPONSIBILITY_TABLE_PATH), generated)
  console.log(`generated ${RESPONSIBILITY_TABLE_PATH}`)
}
