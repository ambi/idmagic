import { featureDocument } from '../../workspace/src/document-layout.ts'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { CheckOutcome } from './runner.ts'
import { stateMachineFindings } from './state-machines.ts'

export async function checkStateMachines(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const lines: string[] = []
  let count = 0
  for (const path of (await snapshot.files('docs')).filter((path) => {
    const part = featureDocument(path)?.part
    return part === 'specification' || part === 'chapter'
  })) {
    const source = await snapshot.read(path)
    if (!/^## 状態遷移[ \t]*$/m.test(source)) continue
    count += 1
    for (const finding of stateMachineFindings(source, (path) =>
      snapshot.exists(path) ? snapshot.readSync(path) : undefined,
    ))
      lines.push(`${path}: ${finding}`)
  }
  return lines.length
    ? { ok: false, lines }
    : { ok: true, lines: [`ok  ${count} document(s) with generated state machines`] }
}
