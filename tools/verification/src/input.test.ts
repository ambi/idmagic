import { afterEach, describe, expect, it } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { git, verificationInput } from './input.ts'

const cleanup: string[] = []
afterEach(async () => {
  for (const root of cleanup.splice(0)) await rm(root, { recursive: true, force: true })
})

const workItem = 'work-items/active/wi-1-demo.md'

/** Go のファイルと作業項目を一つずつ持つ、コミット済みの作業ツリーを作る。 */
async function repository(): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'verification-input-test-'))
  cleanup.push(root)
  await mkdir(join(root, 'backend/demo'), { recursive: true })
  await mkdir(join(root, 'work-items/active'), { recursive: true })
  await writeFile(join(root, 'backend/demo/demo.go'), 'package demo\n')
  await writeFile(join(root, workItem), '---\nstatus: in_progress\n---\n')
  git(root, 'init', '-q')
  commit(root, 'initial')
  return root
}

function commit(root: string, message: string) {
  git(root, 'add', '-A')
  git(
    root,
    '-c',
    'user.name=fixture',
    '-c',
    'user.email=fixture@example.invalid',
    'commit',
    '-qm',
    message,
  )
}

const fingerprint = async (root: string, inputSet: 'workspace' | 'test-execution') =>
  (await verificationInput(createWorkspaceSnapshot(root), undefined, inputSet)).fingerprint

describe('検証の入力の同一性', () => {
  it('テストの実行の入力は、作業項目の書き換え、追加、コミットで変わらない', async () => {
    const root = await repository()
    const before = await fingerprint(root, 'test-execution')

    await writeFile(
      join(root, workItem),
      '---\nstatus: in_progress\n---\n\n## 要約\n\n完了した。\n',
    )
    expect(await fingerprint(root, 'test-execution')).toBe(before)

    await writeFile(join(root, 'work-items/active/wi-2-new.md'), '---\nstatus: pending\n---\n')
    expect(await fingerprint(root, 'test-execution')).toBe(before)

    commit(root, 'record the summary')
    expect(await fingerprint(root, 'test-execution')).toBe(before)
  })

  it('テストの実行の入力は、作業項目の外の書き換えと追加で変わる', async () => {
    const root = await repository()
    const before = await fingerprint(root, 'test-execution')

    await writeFile(join(root, 'backend/demo/demo.go'), 'package demo\n\nconst changed = 1\n')
    const edited = await fingerprint(root, 'test-execution')
    expect(edited).not.toBe(before)

    await writeFile(join(root, 'backend/demo/extra.go'), 'package demo\n')
    expect(await fingerprint(root, 'test-execution')).not.toBe(edited)

    // 接頭辞だけが似たパスは作業項目ではない。
    await mkdir(join(root, 'work-items-archive'))
    const added = await fingerprint(root, 'test-execution')
    await writeFile(join(root, 'work-items-archive/note.md'), 'note\n')
    expect(await fingerprint(root, 'test-execution')).not.toBe(added)
  })

  it('作業ツリー全体の入力は、作業項目の書き換えとコミットで変わる', async () => {
    const root = await repository()
    const before = await fingerprint(root, 'workspace')

    await writeFile(join(root, workItem), '---\nstatus: completed\n---\n')
    const edited = await fingerprint(root, 'workspace')
    expect(edited).not.toBe(before)

    commit(root, 'complete')
    expect(await fingerprint(root, 'workspace')).not.toBe(edited)
  })

  it('範囲を記録し、二つの範囲の値を区別する', async () => {
    const root = await repository()
    const snapshot = createWorkspaceSnapshot(root)
    const test = await verificationInput(snapshot, undefined, 'test-execution')
    const workspace = await verificationInput(snapshot)

    expect(test.inputSet).toBe('test-execution')
    expect(workspace.inputSet).toBe('workspace')
    expect(test.fingerprint).not.toBe(workspace.fingerprint)
    expect(test.revision).toBe(workspace.revision)
  })
})
