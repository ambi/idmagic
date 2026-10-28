import { describe, expect, it } from 'bun:test'
import { repositoryReads } from './tool-test-inputs.ts'

describe('tools のテストがリポジトリの状態を読まないことの検査', () => {
  it('テストの位置から tools の外へ出る相対パスと、根を開くスナップショットを返す', () => {
    const source = [
      "const root = resolve(import.meta.dir, '../../..')",
      "const config = await Bun.file(resolve(import.meta.dir, '../../../mise.toml')).text()",
      'const snapshot = createWorkspaceSnapshot()',
    ].join('\n')
    expect(repositoryReads('tools/check/src/demo.test.ts', source)).toEqual([
      { line: 1, value: '../../..' },
      { line: 2, value: '../../../mise.toml' },
      { line: 3, value: 'createWorkspaceSnapshot()' },
    ])
  })

  it('tools の中の相対パス、fixture の根、コメントは返さない', () => {
    const source = [
      "const runner = resolve(import.meta.dir, 'main.ts')",
      "const verification = resolve(import.meta.dir, '../../verification/src/main.ts')",
      "const fixture = join(import.meta.dir, '../testdata/demo')",
      'const snapshot = createWorkspaceSnapshot(root)',
      "// resolve(import.meta.dir, '../../..') は使わない",
      'expect(html).toContain(\'href="../../../api/index.html"\')',
    ].join('\n')
    expect(repositoryReads('tools/check/src/demo.test.ts', source)).toEqual([])
  })
})
