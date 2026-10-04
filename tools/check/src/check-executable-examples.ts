import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { updateExampleBlocks } from './executable-examples.ts'
import type { CheckOutcome } from './runner.ts'

export async function checkExecutableExamples(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const lines: string[] = []
  let count = 0
  for (const path of (await snapshot.files('docs')).filter((path) =>
    path.endsWith('acceptance.feature.md'),
  )) {
    const source = await snapshot.read(path)
    if (!source.includes('spec:examples')) continue
    count += 1
    try {
      const generated = updateExampleBlocks(source, (path) =>
        snapshot.exists(path) ? snapshot.readSync(path) : undefined,
      )
      if (generated !== source)
        lines.push(`${path}: executable examples drift; run mise run generate-spec-examples`)
    } catch (error) {
      lines.push(`${path}: ${error instanceof Error ? error.message : String(error)}`)
    }
  }
  return lines.length
    ? { ok: false, lines }
    : { ok: true, lines: [`ok  ${count} executable example document(s)`] }
}
