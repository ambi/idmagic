import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { agentGuidanceFiles, verifyAgentGuidance } from './agent-guidance.ts'
import { taskNames } from './command-map.ts'
import type { CheckOutcome } from './runner.ts'

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
