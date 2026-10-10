import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterEach, describe, expect, it } from 'bun:test'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkLinks } from './check-links.ts'

const cleanup: string[] = []
afterEach(async () => {
  await Promise.all(cleanup.splice(0).map((path) => rm(path, { recursive: true, force: true })))
})

const BROKEN = '# 記録\n\n[移した文書](../../docs/moved.md)\n'

function git(root: string, ...args: string[]): void {
  const result = Bun.spawnSync(
    [
      'git',
      '-c',
      'user.name=t',
      '-c',
      'user.email=t@example.com',
      '-c',
      'commit.gpgsign=false',
      ...args,
    ],
    { cwd: root },
  )
  if (result.exitCode !== 0) throw new Error(result.stderr.toString())
}

/** `files` を main の最初のコミットにした Git の workspace。 */
async function committedWorkspace(files: Record<string, string>): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-links-test-'))
  cleanup.push(root)
  for (const [path, source] of Object.entries(files)) {
    await mkdir(join(root, path, '..'), { recursive: true })
    await writeFile(join(root, path), source)
  }
  git(root, 'init', '-q', '-b', 'main')
  git(root, 'add', '-A')
  git(root, 'commit', '-q', '-m', 'base')
  return root
}

describe('checkLinks', () => {
  it('基準から変わっていない完了済みの記録のリンクは検査しない', async () => {
    const root = await committedWorkspace({
      'README.md': '# 入口\n',
      'work-items/done/wi-10004-closed.md': BROKEN,
    })

    const outcome = await checkLinks(createWorkspaceSnapshot(root))

    expect(outcome).toEqual({ ok: true, lines: ['ok  Markdown links (1 document(s))'] })
  })

  it('作業ツリーで変わった完了済みの記録のリンクを検査する', async () => {
    const root = await committedWorkspace({ 'work-items/done/wi-10004-closed.md': BROKEN })
    await writeFile(join(root, 'work-items/done/wi-10004-closed.md'), `${BROKEN}\n追記。\n`)

    const outcome = await checkLinks(createWorkspaceSnapshot(root))

    expect(outcome.ok).toBe(false)
    expect(outcome.lines.join('\n')).toContain('work-items/done/wi-10004-closed.md:3:')
  })

  it('リリース文書のリンクは、変わっていなくても検査する', async () => {
    const root = await committedWorkspace({ 'docs/releases/changes/wi-10004-closed.md': BROKEN })

    const outcome = await checkLinks(createWorkspaceSnapshot(root))

    expect(outcome.ok).toBe(false)
    expect(outcome.lines.join('\n')).toContain('docs/releases/changes/wi-10004-closed.md:3:')
  })
})
