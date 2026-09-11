import type { WorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { verifyCommandMap } from './command-map.ts'
import type { CheckOutcome } from './runner.ts'

export async function checkCommandMap(snapshot: WorkspaceSnapshot): Promise<CheckOutcome> {
  const workflows: Array<{ file: string; source: string }> = []
  try {
    for (const entry of await snapshot.list('.github/workflows')) {
      if (!entry.isFile() || !/\.ya?ml$/.test(entry.name)) continue
      const file = `.github/workflows/${entry.name}`
      workflows.push({ file, source: await snapshot.read(file) })
    }
  } catch {
    // workflow を持たないリポジトリには、mise と不一致になる呼び出しもない。
  }
  const findings = verifyCommandMap(await snapshot.read('mise.toml'), workflows)
  return {
    ok: findings.length === 0,
    lines:
      findings.length > 0
        ? findings.map(
            (finding) =>
              `${finding.file}: workflow calls a task mise.toml does not define: ${finding.task}`,
          )
        : [`ok  command map (${workflows.length} workflow file(s))`],
  }
}
