import { expect, test } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { agentGuidanceFiles } from './agent-guidance.ts'
import { checkAgentGuidance } from './check-agent-guidance.ts'

/** どの指示ファイルにも求められる目印をすべて持つ、最小の手順。 */
const guidance = [
  '# 手順',
  '',
  '`mise run brief -- wi-1 --target backend --concern contract --concern boundary` を実行する。',
  '',
  '- docs/development/coding-style.md、docs/development/task-context.md、docs/README.md',
  '- docs/design/application/backend.md、docs/modules/<module>/README.md',
  '- spec/modules/<module>/{models,main}.tsp、docs/modules/<module>/features/<feature>.md、<feature>.feature.md',
  '- fault_model',
  '',
].join('\n')

test('手順が呼ぶタスクの実在を、同じ workspace の mise.toml で判定する', async () => {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-guidance-'))
  try {
    for (const file of agentGuidanceFiles) {
      await mkdir(dirname(join(root, file)), { recursive: true })
      await writeFile(join(root, file), guidance)
    }
    await writeFile(join(root, 'mise.toml'), '[tasks.brief]\nrun = "true"\n')
    expect(await checkAgentGuidance(createWorkspaceSnapshot(root))).toMatchObject({ ok: true })

    const file = '.agents/skills/implement-work-item/SKILL.md'
    await writeFile(join(root, file), `${guidance}\nmise run guidance-only-task\n`)
    const missing = await checkAgentGuidance(createWorkspaceSnapshot(root))
    expect(missing.ok).toBe(false)
    expect(missing.lines).toContain(`${file}: calls an undeclared mise task: guidance-only-task`)

    await writeFile(
      join(root, 'mise.toml'),
      '[tasks.brief]\nrun = "true"\n\n[tasks.guidance-only-task]\nrun = "true"\n',
    )
    expect(await checkAgentGuidance(createWorkspaceSnapshot(root))).toMatchObject({ ok: true })
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})
