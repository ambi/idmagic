import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { afterAll, describe, expect, it } from 'bun:test'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkBoundaries } from './check-boundaries.ts'
import { checkLinks } from './check-links.ts'

const cleanup: string[] = []
afterAll(async () => {
  for (const path of cleanup) await rm(path, { recursive: true, force: true })
})

/**
 * `.worktrees/wi-999` に別の作業ツリーを置いた仮のリポジトリ。
 * worktree の中身には、検査の対象に入れば必ず指摘される内容を置く。
 */
async function repositoryWithWorktree(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'check-worktree-exclusion-test-'))
  cleanup.push(root)
  const worktree = join(root, '.worktrees', 'wi-999')
  await mkdir(join(root, 'docs'), { recursive: true })
  await mkdir(join(worktree, 'docs'), { recursive: true })
  await writeFile(join(root, 'go.mod'), 'module example.com/sample\n')
  await writeFile(join(root, 'docs', 'README.md'), '# Docs\n')
  await writeFile(join(worktree, 'docs', 'README.md'), '# Docs\n\n[missing](missing.md)\n')
  await writeFile(join(worktree, 'architecture.yaml'), 'layers: []\n')
  return root
}

describe('repository-root scans', () => {
  it('check-links does not inspect Markdown inside work-item worktrees', async () => {
    const outcome = await checkLinks(createWorkspaceSnapshot(await repositoryWithWorktree()))

    expect(outcome).toEqual({ ok: true, lines: ['ok  Markdown links (1 document(s))'] })
  })

  it('check-boundaries does not inspect files inside work-item worktrees', async () => {
    const outcome = await checkBoundaries(createWorkspaceSnapshot(await repositoryWithWorktree()))

    expect(outcome.ok).toBe(true)
  })
})
