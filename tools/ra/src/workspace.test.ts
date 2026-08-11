import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { discoverWorkspaceConfig } from './workspace.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

async function workspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'ra-workspace-test-'))
  cleanup.push(root)
  await mkdir(join(root, 'spec', 'contexts', 'demo'), { recursive: true })
  await mkdir(join(root, 'work-items', 'done'), { recursive: true })
  await mkdir(join(root, 'decisions'), { recursive: true })
  await writeFile(join(root, 'spec', 'main.tsp'), 'namespace Demo;\n')
  await writeFile(join(root, 'spec', 'requirements.md'), '# Requirements\n')
  await writeFile(join(root, 'spec', 'contexts', 'demo', 'requirements.md'), '# Demo\n')
  await writeFile(join(root, 'ARCHITECTURE.md'), '# Architecture\n')
  return root
}

describe('discoverWorkspaceConfig', () => {
  it('discovers the standard layout without a registry file', async () => {
    const root = await workspace()
    const config = await discoverWorkspaceConfig(root)
    expect(config.specification).toBe('spec/main.tsp')
    expect(config.requirements).toEqual([
      'spec/contexts/demo/requirements.md',
      'spec/requirements.md',
    ])
    expect(config.workItems).toBe('work-items')
    expect(config.decisions).toBe('decisions')
    expect(config.architectureDocs).toEqual(['ARCHITECTURE.md'])
  })

  it('rejects an empty directory with no RA targets', async () => {
    const root = await mkdtemp(join(tmpdir(), 'ra-workspace-test-'))
    cleanup.push(root)
    await expect(discoverWorkspaceConfig(root)).rejects.toThrow(
      'no Regenerative Architecture workspace targets',
    )
  })
})
