import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { isCurrentStateDocument, verifyNoWorkItemLinks } from './docs-work-item-links.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'

export async function checkDocsWorkItemLinks(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
): Promise<CheckOutcome> {
  const paths = (await snapshot.files('docs').catch((): string[] => []))
    .filter(isCurrentStateDocument)
    .sort()
  const documents = await Promise.all(
    paths.map(async (file) => ({ file, source: await snapshot.read(file) })),
  )
  const findings = verifyNoWorkItemLinks(documents)
  if (findings.length > 0) {
    return {
      ok: false,
      lines: findings.map(
        (finding) =>
          `${finding.file}:${finding.line}:${finding.column}: 現在状態の文書が work item を参照している（${finding.reference}）。記録ではなく現在の状態を書く。`,
      ),
    }
  }
  return {
    ok: true,
    lines: [
      ...(options.verbose ? documents.map((document) => `ok  ${document.file}`) : []),
      `ok  work-item-references (${documents.length} document(s))`,
    ],
  }
}
