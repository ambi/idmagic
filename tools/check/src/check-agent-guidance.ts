import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { agentGuidanceFiles, verifyAgentGuidance } from './agent-guidance.ts'
import type { CheckOutcome } from './runner.ts'

/** mise.toml が宣言するタスクの名前。 */
function taskNames(miseToml: string): Set<string> {
  const document = Bun.TOML.parse(miseToml) as { tasks?: Record<string, unknown> }
  return new Set(Object.keys(document.tasks ?? {}))
}

export async function checkAgentGuidance(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const documents = await Promise.all(
    agentGuidanceFiles.map(async (file) => ({ file, source: await snapshot.read(file) })),
  )
  const findings = verifyAgentGuidance(documents, taskNames(await snapshot.read('mise.toml')))
  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map((finding) => `${finding.file}: ${finding.message}`)
        : [`ok  agent guidance (${documents.length} guidance file(s))`],
  }
}
