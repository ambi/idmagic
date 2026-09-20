import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { verifyDocumentLayout } from './document-layout-format.ts'
import type { CheckOutcome } from './runner.ts'

const FORMAT_PATH = 'SPECIFICATION_FORMAT.md'

export async function checkDocumentLayout(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const findings = verifyDocumentLayout(await snapshot.read(FORMAT_PATH))
  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map((finding) => `${FORMAT_PATH}: ${finding.message}`)
        : ['ok  document layout matches canonical document definitions'],
  }
}
