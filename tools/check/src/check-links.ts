import { posix } from 'node:path'
import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { verifyMarkdownLinks } from './markdown-links.ts'
import type { CheckOutcome } from './runner.ts'

function excluded(path: string): boolean {
  const segments = path.split('/')
  return (
    segments.includes('.git') ||
    segments.includes('node_modules') ||
    path === 'spec/generated' ||
    path.startsWith('spec/generated/')
  )
}

export async function checkLinks(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const documents = new Map<string, string>()
  const existingPaths = new Set<string>()
  for (const path of await snapshot.files('', ['.git', 'node_modules'])) {
    if (excluded(path)) continue
    existingPaths.add(path)
    let parent = posix.dirname(path)
    while (parent !== '.') {
      existingPaths.add(parent)
      parent = posix.dirname(parent)
    }
    if (path.endsWith('.md')) documents.set(path, await snapshot.read(path))
  }
  const findings = verifyMarkdownLinks(documents, existingPaths)
  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map(
            (finding) => `${finding.file}:${finding.line}: ${finding.message} (${finding.target})`,
          )
        : [`ok  Markdown links (${documents.size} document(s))`],
  }
}
