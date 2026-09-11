import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import {
  createWorkspaceSnapshot,
  discoverGeneratedOpenApi,
  discoverOpenApiBaseline,
  discoverWorkspaceConfig,
  listCanonicalDirectories,
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

describe('createWorkspaceSnapshot', () => {
  it('一回の実行では本文とディレクトリ一覧を同じ状態に保つ', async () => {
    const root = await workspace()
    const snapshot = createWorkspaceSnapshot(root)

    expect(await snapshot.read('docs/README.md')).toBe('# Specification\n')
    expect((await snapshot.list('docs')).map((entry) => entry.name)).toContain('README.md')

    await writeFile(join(root, 'docs', 'README.md'), '# Changed\n')
    await writeFile(join(root, 'docs', 'later.md'), '# Later\n')

    expect(await snapshot.read('docs/README.md')).toBe('# Specification\n')
    expect((await snapshot.list('docs')).map((entry) => entry.name)).not.toContain('later.md')
  })
})

async function workspace(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'spec-workspace-test-'))
  cleanup.push(root)
  await mkdir(join(root, 'spec', 'contexts', 'demo'), { recursive: true })
  await mkdir(join(root, 'docs', 'contexts', 'demo'), { recursive: true })
  await mkdir(join(root, 'docs', 'requirements'), { recursive: true })
  await mkdir(join(root, 'docs', 'design', 'security'), { recursive: true })
  await mkdir(join(root, 'work-items', 'done'), { recursive: true })
  await writeFile(join(root, 'spec', 'main.tsp'), 'namespace Demo;\n')
  await writeFile(join(root, 'docs', 'README.md'), '# Specification\n')
  await writeFile(join(root, 'docs', 'requirements', 'README.md'), '# Requirements\n')
  await writeFile(join(root, 'docs', 'requirements', 'quality.md'), '# Quality Requirements\n')
  await writeFile(join(root, 'docs', 'design', 'security', 'README.md'), '# Security Design\n')
  await writeFile(join(root, 'docs', 'design', 'security', 'authorization.md'), '# Authorization\n')
  await writeFile(join(root, 'docs', 'contexts', 'demo', 'README.md'), '# Demo\n')
  await writeFile(
    join(root, 'docs', 'contexts', 'demo', 'scenarios.feature.md'),
    '# Demo Scenarios\n',
  )
  return root
}

describe('discoverWorkspaceConfig', () => {
  it('discovers the standard layout without a registry file', async () => {
    const root = await workspace()
    const config = await discoverWorkspaceConfig(root)
    expect(config.specification).toBe('spec/main.tsp')
    expect(config.documents).toEqual([
      'docs/README.md',
      'docs/contexts/demo/README.md',
      'docs/contexts/demo/scenarios.feature.md',
      'docs/design/security/README.md',
      'docs/design/security/authorization.md',
      'docs/requirements/README.md',
      'docs/requirements/quality.md',
    ])
    expect(config.workItems).toBe('work-items')
  })

  it('ignores a Markdown file the layout does not name', async () => {
    const root = await workspace()
    await writeFile(join(root, 'docs', 'contexts', 'demo', 'notes.md'), '# Notes\n')
    await writeFile(join(root, 'docs', 'states.md'), '# States\n')
    const config = await discoverWorkspaceConfig(root)
    expect(config.documents).not.toContain('docs/contexts/demo/notes.md')
    expect(config.documents).not.toContain('docs/states.md')
  })

  /**
   * 固定の一覧から段を落とすと、その段の規範文書は集める側からも拒否する側からも
   * 消える。文書は残っているのに検証だけが止まるので、一覧に無い段こそ列挙して
   * 閉じた集合の検査へ渡さなければならない。
   */
  it('lists a directory below docs the fixed layout does not name', async () => {
    const root = await workspace()
    await mkdir(join(root, 'docs', 'design', 'unplanned'), { recursive: true })
    await writeFile(join(root, 'docs', 'design', 'unplanned', 'threat-model.md'), '# Threats\n')

    const listings = await listCanonicalDirectories(root)
    expect(
      listings.find((listing) => listing.directory === 'docs/design/unplanned')?.files,
    ).toEqual(['threat-model.md'])
  })

  it('leaves the freely named directories below docs out of the closed set', async () => {
    const root = await workspace()
    await mkdir(join(root, 'docs', 'runbooks'), { recursive: true })
    await mkdir(join(root, 'docs', 'releases', 'upgrades'), { recursive: true })
    await mkdir(join(root, 'docs', 'development'), { recursive: true })
    await writeFile(join(root, 'docs', 'runbooks', 'anything.md'), '# Anything\n')
    await writeFile(join(root, 'docs', 'releases', 'upgrades', 'wi-1.md'), '# Upgrade\n')
    await writeFile(join(root, 'docs', 'development', 'release.md'), '# Release\n')

    const directories = (await listCanonicalDirectories(root)).map((listing) => listing.directory)
    expect(directories).not.toContain('docs/runbooks')
    expect(directories).not.toContain('docs/releases')
    expect(directories).not.toContain('docs/releases/upgrades')
    expect(directories).not.toContain('docs/development')
  })

  it('rejects a leftover SPECIFICATION.md as a second source of truth', async () => {
    const root = await workspace()
    await writeFile(join(root, 'docs', 'contexts', 'demo', 'SPECIFICATION.md'), '# Demo\n')
    await expect(discoverWorkspaceConfig(root)).rejects.toThrow(
      'legacy specification documents found: docs/contexts/demo/SPECIFICATION.md',
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
