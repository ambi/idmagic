import { describe, expect, it } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import MarkdownIt from 'markdown-it'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkExecutableExamples } from './check-executable-examples.ts'
import { renderExampleCases, updateExampleBlocks } from './executable-examples.ts'
import { diffSpecifications } from './spec-diff.ts'

const cases = [
  {
    id: 'EX-DEMO-001-01',
    title: '空白と区切りを含む入力',
    given: {},
    input: { name: ' a|b\n ' },
    expected: { name: 'a|b' },
  },
]
const fixture = 'backend/demo/feature/testdata/values.examples.json'
const source = `## Rule: REQ-DEMO-001 正規化する\n\n<!-- spec:examples ${fixture} -->\n古い例\n<!-- /spec:examples -->\n\n本文を残す。\n`
const specification = [
  'docs/modules/demo/feature/README.md',
  '# 機能\n\n## 操作\n\n### 正規化\n\n#### REQ-DEMO-001 正規化する\n\n- 入力を正規化する。\n',
] as const

describe('実行可能な具体例', () => {
  it('期待値だけの変更も親規則の仕様差分へ返す', () => {
    const path = 'docs/modules/demo/feature/acceptance.feature.md'
    const document = (expected: string) =>
      '# Feature: Demo\n\n' +
      updateExampleBlocks(source, () =>
        JSON.stringify([{ ...cases[0], expected: { name: expected } }]),
      )
    const changed = diffSpecifications(
      new Map([specification, [path, document('a|b')]]),
      new Map([specification, [path, document('changed')]]),
    )
    expect(changed.changedScenarios).toEqual(['REQ-DEMO-001'])
  })
  it('期待値の集合が同じでも、入力との対応の変更を検出する', () => {
    const path = 'docs/modules/demo/feature/acceptance.feature.md'
    const data = ['first', 'second'].map((name, index) => ({
      ...cases[0],
      id: `EX-DEMO-001-0${index + 1}`,
      input: { name },
      expected: { name },
    }))
    const document = (values: unknown) =>
      '# Feature: Demo\n\n' + updateExampleBlocks(source, () => JSON.stringify(values))
    const before = new Map([specification, [path, document(data)]])
    const swapped = data.map((item, index) => ({ ...item, expected: data[1 - index]?.expected }))
    expect(
      diffSpecifications(before, new Map([specification, [path, document(swapped)]]))
        .changedScenarios,
    ).toEqual(['REQ-DEMO-001'])
    expect(
      diffSpecifications(before, new Map([specification, [path, document([...data].reverse())]]))
        .changedScenarios,
    ).toEqual([])
  })
  it('具体値を Markdown の表として読み取れる形で表示する', () => {
    const html = new MarkdownIt().render(renderExampleCases(cases))
    expect(html).toContain('<table>')
    expect(html).toContain('<td>name</td>')
    expect(html).toContain('追加の前提はない')
  })
  it('一次情報の値を表示し、古い派生区間だけを置き換える', () => {
    const updated = updateExampleBlocks(source, () => JSON.stringify(cases))
    expect(updated).toContain('### Example: EX-DEMO-001-01')
    expect(updated).toContain('a\\|b')
    expect(updated).not.toContain('古い例')
    expect(updated).toEndWith('本文を残す。\n')
    expect(updateExampleBlocks(updated, () => JSON.stringify(cases))).toBe(updated)
  })

  it('コード例の marker は実行せず、文書の参照だけを解決する', () => {
    const fenced = `\`\`\`markdown\n${source}\`\`\`\n`
    expect(
      updateExampleBlocks(fenced, () => {
        throw new Error('読んではならない')
      }),
    ).toBe(fenced)
  })

  it('欠落、重複、未知の欄、空の期待結果を拒否する', () => {
    for (const value of [
      [],
      [...cases, ...cases],
      [{ ...cases[0], extra: true }],
      [{ ...cases[0], expected: {} }],
    ]) {
      expect(() => renderExampleCases(value)).toThrow()
    }
    expect(() => updateExampleBlocks(source, () => undefined)).toThrow('does not exist')
    expect(() =>
      updateExampleBlocks(source.replace('<!-- /spec:examples -->', ''), () => '[]'),
    ).toThrow('not closed')
  })

  it('親ディレクトリと絶対パスと見出し注入を拒否する', () => {
    for (const path of [
      '../secret.examples.json',
      '/tmp/values.examples.json',
      'backend/../values.examples.json',
    ]) {
      expect(() => updateExampleBlocks(source.replace(fixture, path), () => '[]')).toThrow('path')
    }
    expect(() =>
      renderExampleCases([{ ...cases[0], title: '例\n## Rule: REQ-DEMO-999' }]),
    ).toThrow()
  })

  it('崩れた marker と、コードフェンス内だけにある閉じ marker を拒否する', () => {
    expect(() => updateExampleBlocks(source.replace(` ${fixture} `, ''), () => '[]')).toThrow(
      'marker',
    )
    expect(() =>
      updateExampleBlocks(
        source.replace('<!-- /spec:examples -->', '```\n<!-- /spec:examples -->\n```'),
        () => '[]',
      ),
    ).toThrow('not closed')
  })

  it('標準検査は文書の改変、一次情報の変更と欠落を検出し、自身では書き換えない', async () => {
    const root = await mkdtemp(join(tmpdir(), 'idmagic-examples-'))
    const doc = 'docs/modules/demo/feature/acceptance.feature.md'
    try {
      await mkdir(dirname(join(root, doc)), { recursive: true })
      await mkdir(dirname(join(root, fixture)), { recursive: true })
      const generated = updateExampleBlocks(source, () => JSON.stringify(cases))
      await writeFile(join(root, fixture), JSON.stringify(cases))
      await writeFile(join(root, doc), generated)
      expect((await checkExecutableExamples(createWorkspaceSnapshot(root))).ok).toBe(true)
      await writeFile(join(root, doc), generated.replace('a\\|b', 'incorrect'))
      expect((await checkExecutableExamples(createWorkspaceSnapshot(root))).ok).toBe(false)
      await writeFile(join(root, doc), generated)
      await writeFile(
        join(root, fixture),
        JSON.stringify([{ ...cases[0], expected: { name: 'changed' } }]),
      )
      const changed = createWorkspaceSnapshot(root)
      expect((await checkExecutableExamples(changed)).ok).toBe(false)
      expect(await changed.read(doc)).toBe(generated)
      await rm(join(root, fixture))
      expect((await checkExecutableExamples(createWorkspaceSnapshot(root))).ok).toBe(false)
    } finally {
      await rm(root, { recursive: true, force: true })
    }
  })

  it('区切りや改行を含む値でも生成と再生成が一致する', () => {
    const alphabet = ['a', '|', '\\', '\n', '\r', '<', '>', '`', '日', '😀']
    let seed = 34029
    const inputs = Array.from({ length: 256 }, () =>
      Array.from({ length: 16 }, () => {
        seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0
        return alphabet[seed % alphabet.length]
      }).join(''),
    )
    for (const name of [
      '',
      'a|b',
      '\\',
      '\n',
      '\r\n',
      '<!-- /spec:examples -->',
      '```',
      '日本語',
      ...inputs,
    ]) {
      const data = JSON.stringify([{ ...cases[0], input: { name } }])
      const once = updateExampleBlocks(source, () => data)
      expect(updateExampleBlocks(once, () => data)).toBe(once)
      expect(once.match(/^### Example:/gm)).toHaveLength(1)
    }
  })
})
