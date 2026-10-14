import { afterEach, describe, expect, it } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { missingArchivedRevisions } from './archived-work-item.ts'
import { checkWorkItems } from './check-work-items.ts'

const temporaryDirectories: string[] = []

async function workspaceWith(name: string, source: string): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-archived-'))
  temporaryDirectories.push(root)
  await mkdir(join(root, 'spec'), { recursive: true })
  await writeFile(join(root, 'spec', 'release-state.json'), '{"phase":"unpublished"}')
  await mkdir(join(root, 'work-items', 'done'), { recursive: true })
  await writeFile(join(root, 'work-items', 'done', name), source)
  return root
}

afterEach(async () => {
  await Promise.all(temporaryDirectories.splice(0).map((path) => rm(path, { recursive: true })))
})

const archived = (frontmatter: string) =>
  `---\n${frontmatter}\n---\n\n# 記録を縮める\n\n縮めた結果の要約。\n`

describe('archived work items', () => {
  it('accepts a completed record collapsed to its title and summary', async () => {
    const root = await workspaceWith(
      'wi-10001-collapse.md',
      archived('status: completed\narchived_from: 0123456789abcdef0123456789abcdef01234567'),
    )
    const outcome = await checkWorkItems(createWorkspaceSnapshot(root))
    expect(outcome.lines).toEqual(['ok  1 work-item dependency record(s)'])
  })

  it('rejects an archived record that keeps fields the full record owns', async () => {
    const root = await workspaceWith(
      'wi-10002-collapse.md',
      archived(
        'status: completed\narchived_from: 0123456789abcdef0123456789abcdef01234567\naffected_spec: []',
      ),
    )
    const outcome = await checkWorkItems(createWorkspaceSnapshot(root))
    expect(outcome.ok).toBe(false)
    expect(outcome.lines.join('\n')).toContain(
      'archived record keeps only status and archived_from',
    )
  })

  it('rejects an archived record that is not completed or cancelled', async () => {
    const root = await workspaceWith(
      'wi-10003-collapse.md',
      archived('status: in_progress\narchived_from: 0123456789abcdef0123456789abcdef01234567'),
    )
    const outcome = await checkWorkItems(createWorkspaceSnapshot(root))
    expect(outcome.ok).toBe(false)
    expect(outcome.lines.join('\n')).toContain('archived record must be completed or cancelled')
  })

  it('reports a revision that does not hold the full record', () => {
    const root = resolve(import.meta.dir, '../../..')
    const head = Bun.spawnSync(['git', 'rev-parse', 'HEAD'], { cwd: root }).stdout.toString().trim()
    expect(
      missingArchivedRevisions(root, [
        { path: 'AGENTS.md', revision: head },
        { path: 'work-items/done/wi-10004-missing.md', revision: head },
      ]),
    ).toEqual([
      `work-items/done/wi-10004-missing.md: archived_from ${head} does not hold this record`,
    ])
  })
})
