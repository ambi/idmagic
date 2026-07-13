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
  await mkdir(join(root, 'spec', 'contexts'), { recursive: true })
  await mkdir(join(root, 'work-items', 'done'), { recursive: true })
  await mkdir(join(root, 'decisions'), { recursive: true })
  await writeFile(join(root, 'spec', 'scl.yaml'), 'system: demo\nspec_version: "3.0"\n')
  return root
}

describe('discoverWorkspaceConfig', () => {
  it('discovers the standard app layout without a registry file', async () => {
    const root = await workspace()
    const config = await discoverWorkspaceConfig(root)

    expect(config.apps).toEqual([
      expect.objectContaining({
        name: 'demo',
        root: '.',
        scl: 'spec/scl.yaml',
        contextGlob: 'spec/contexts/*.yaml',
        workItems: 'work-items',
        decisions: 'decisions',
      }),
    ])
  })

  it('discovers and sorts every embedded tool SCL 3.0 spec', async () => {
    const root = await workspace()
    for (const tool of ['zeta', 'alpha']) {
      await mkdir(join(root, 'tools', tool, 'spec'), { recursive: true })
      await writeFile(
        join(root, 'tools', tool, 'spec', 'scl.yaml'),
        `system: ${tool}\nspec_version: "3.0"\n`,
      )
    }

    const config = await discoverWorkspaceConfig(root)
    expect(config.toolSpecs).toEqual(['tools/alpha/spec/scl.yaml', 'tools/zeta/spec/scl.yaml'])
  })

  it('rejects an empty directory with no RA targets', async () => {
    const root = await mkdtemp(join(tmpdir(), 'ra-workspace-test-'))
    cleanup.push(root)
    await expect(discoverWorkspaceConfig(root)).rejects.toThrow('no RA workspace targets')
  })
})
