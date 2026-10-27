import { featureDocument } from '../../workspace/src/document-layout.ts'
import { writeFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { updateExampleBlocks } from './executable-examples.ts'
import { updateStateMachineBlocks } from './state-machines.ts'

const snapshot = createWorkspaceSnapshot()
const read = (path: string) => (snapshot.exists(path) ? snapshot.readSync(path) : undefined)
for (const path of await snapshot.files('docs')) {
  const part = featureDocument(path)?.part
  if (part === undefined) continue
  const source = await snapshot.read(path)
  const generated =
    part === 'examples' ? updateExampleBlocks(source, read) : updateStateMachineBlocks(source, read)
  if (generated !== source) {
    await writeFile(resolve(snapshot.root, path), generated)
    console.log(`generated ${path}`)
  }
}
