import { expect, test } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, resolve } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { agentGuidanceFiles } from './agent-guidance.ts'
import { checkAgentGuidance } from './check-agent-guidance.ts'

test('手順が呼ぶタスクの実在を、同じ workspace の mise.toml で判定する', async () => {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-guidance-'))
  const repository = resolve(import.meta.dir, '../../..')
  try {
    for (const file of [...agentGuidanceFiles, 'mise.toml']) {
      await mkdir(dirname(join(root, file)), { recursive: true })
      await writeFile(join(root, file), await Bun.file(join(repository, file)).text())
    }
    expect((await checkAgentGuidance(createWorkspaceSnapshot(root))).ok).toBe(true)

    const file = '.agents/skills/implement-work-item/SKILL.md'
    const source = await Bun.file(join(root, file)).text()
    await writeFile(join(root, file), `${source}\nmise run guidance-only-task\n`)
    const missing = await checkAgentGuidance(createWorkspaceSnapshot(root))
    expect(missing.ok).toBe(false)
    expect(missing.lines).toContain(`${file}: calls an undeclared mise task: guidance-only-task`)

    await writeFile(
      join(root, 'mise.toml'),
      `${await Bun.file(join(root, 'mise.toml')).text()}\n[tasks.guidance-only-task]\nrun = "true"\n`,
    )
    expect((await checkAgentGuidance(createWorkspaceSnapshot(root))).ok).toBe(true)
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})
