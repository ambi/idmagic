import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import type { CheckOptions, CheckOutcome } from './runner.ts'
import { TERMINOLOGY_ROOT_DOCUMENTS, verifyTerminology } from './terminology.ts'

/**
 * 用語を固定する対象は `docs/` の Markdown と、文書体系が所有する root 直下の
 * Markdown である。`CONFIGURATION.md` と `ROUTE_PRIORITY.md` は生成物なので入れず、
 * `work-items/` は書かれた時点の記録なので後から書き換えない。
 */
async function terminologyDocumentPaths(snapshot: WorkspaceSnapshot): Promise<string[]> {
  const docs = await snapshot.files('docs').catch((): string[] => [])
  return [
    ...docs.filter((path) => path.endsWith('.md')),
    ...TERMINOLOGY_ROOT_DOCUMENTS.filter((path) => snapshot.exists(path)),
  ].sort()
}

export async function checkTerminology(
  snapshot: WorkspaceSnapshot,
  options: CheckOptions,
): Promise<CheckOutcome> {
  const paths = await terminologyDocumentPaths(snapshot)
  const documents = await Promise.all(
    paths.map(async (file) => ({ file, source: await snapshot.read(file) })),
  )
  const findings = verifyTerminology(documents)
  if (findings.length > 0) {
    return {
      ok: false,
      lines: findings.map(
        (finding) => `${finding.file}:${finding.line}:${finding.column}: ${finding.message}`,
      ),
    }
  }
  return {
    ok: true,
    lines: [
      ...(options.verbose ? documents.map((document) => `ok  ${document.file}`) : []),
      `ok  terminology (${documents.length} document(s))`,
    ],
  }
}
