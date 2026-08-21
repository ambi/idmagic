import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import {
  discoverGeneratedOpenApi,
  discoverOpenApiBaseline,
  discoverWorkspaceConfig,
} from './workspace.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

describe('OpenAPI artifact discovery', () => {
  it('discovers product-neutral filenames from the standard directories', async () => {
    const root = await workspace()
    await mkdir(join(root, 'spec', 'generated', 'openapi'), { recursive: true })
    await writeFile(join(root, 'spec', 'sample.openapi.baseline.json'), '{}\n')
    await writeFile(join(root, 'spec', 'generated', 'openapi', 'sample.openapi.json'), '{}\n')

    expect(await discoverOpenApiBaseline(root)).toBe(
      join(root, 'spec', 'sample.openapi.baseline.json'),
    )
    expect(await discoverGeneratedOpenApi(root)).toBe(
      join(root, 'spec', 'generated', 'openapi', 'sample.openapi.json'),
    )
  })

  it('rejects ambiguous generated OpenAPI documents', async () => {
    const root = await workspace()
    await mkdir(join(root, 'spec', 'generated', 'openapi'), { recursive: true })
    await writeFile(join(root, 'spec', 'generated', 'openapi', 'one.json'), '{}\n')
    await writeFile(join(root, 'spec', 'generated', 'openapi', 'two.json'), '{}\n')

    await expect(discoverGeneratedOpenApi(root)).rejects.toThrow('found 2')
  })
})

async function workspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'spec-workspace-test-'))
  cleanup.push(root)
  await mkdir(join(root, 'spec', 'contexts', 'demo'), { recursive: true })
  await mkdir(join(root, 'work-items', 'done'), { recursive: true })
  await writeFile(join(root, 'spec', 'main.tsp'), 'namespace Demo;\n')
  await writeFile(join(root, 'spec', 'README.md'), '# Specification\n')
  await writeFile(join(root, 'spec', 'authorization.md'), '# Authorization\n')
  await writeFile(join(root, 'spec', 'contexts', 'demo', 'README.md'), '# Demo\n')
  await writeFile(join(root, 'spec', 'contexts', 'demo', 'scenarios.md'), '# Demo Scenarios\n')
  return root
}

describe('discoverWorkspaceConfig', () => {
  it('discovers the standard layout without a registry file', async () => {
    const root = await workspace()
    const config = await discoverWorkspaceConfig(root)
    expect(config.specification).toBe('spec/main.tsp')
    expect(config.documents).toEqual([
      'spec/README.md',
      'spec/authorization.md',
      'spec/contexts/demo/README.md',
      'spec/contexts/demo/scenarios.md',
    ])
    expect(config.workItems).toBe('work-items')
  })

  it('ignores a Markdown file the layout does not name', async () => {
    const root = await workspace()
    await writeFile(join(root, 'spec', 'contexts', 'demo', 'notes.md'), '# Notes\n')
    await writeFile(join(root, 'spec', 'states.md'), '# States\n')
    const config = await discoverWorkspaceConfig(root)
    expect(config.documents).not.toContain('spec/contexts/demo/notes.md')
    expect(config.documents).not.toContain('spec/states.md')
  })

  it('rejects a leftover SPECIFICATION.md as a second source of truth', async () => {
    const root = await workspace()
    await writeFile(join(root, 'spec', 'contexts', 'demo', 'SPECIFICATION.md'), '# Demo\n')
    await expect(discoverWorkspaceConfig(root)).rejects.toThrow(
      'legacy specification documents found: spec/contexts/demo/SPECIFICATION.md',
    )
  })

  it('rejects an empty directory with no specification targets', async () => {
    const root = await mkdtemp(join(tmpdir(), 'spec-workspace-test-'))
    cleanup.push(root)
    await expect(discoverWorkspaceConfig(root)).rejects.toThrow(
      'no specification-first workspace targets',
    )
  })
})
