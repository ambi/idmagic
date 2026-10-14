import { afterEach, expect, it } from 'bun:test'
import { mkdir, mkdtemp, readFile, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { dirname, join, posix, resolve } from 'node:path'
import MarkdownIt from 'markdown-it'
import { verifyMarkdownLinks } from '../../check/src/markdown-links.ts'
import { moveWorkItem, relocateMarkdownLinks } from './move.ts'

const roots: string[] = []
afterEach(async () => {
  await Promise.all(roots.splice(0).map((root) => rm(root, { recursive: true })))
})

const item = 'wi-10001-one.md'
const from = `work-items/active/${item}`
const to = `work-items/done/${item}`

async function workspace(documents: Map<string, string>): Promise<string> {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-move-item-'))
  roots.push(root)
  await mkdir(join(root, 'spec'), { recursive: true })
  await writeFile(join(root, 'spec/release-state.json'), '{"phase":"unpublished"}')
  for (const [path, source] of documents) {
    await mkdir(dirname(join(root, path)), { recursive: true })
    await writeFile(join(root, path), source)
  }
  return root
}

const record = (status: string, summary = '\n## 要約\n\n文書の表記を直した。\n') =>
  `---\nstatus: ${status}\nauthors: [tn]\nrisk: low\ncreated_at: 2026-10-15\ndepends_on: []\nchange_kind: docs\nspec_impact: { kind: none, reason: "文書の表記だけを直し、HTTP 応答、永続状態、イベントは変えない。" }\n---\n\n# 項目\n\n## 動機\n\n[文書](../../docs/README.md) を直す。\n\n## 対象範囲\n\n- 表記\n\n## 対象外\n\n- 内容\n\n## 検証\n\n- check-links\n\n## リスク\n\nない。\n${summary}`

for (const [flags, status] of [
  [[], 'completed'],
  [['--cancel'], 'cancelled'],
] as const) {
  it(`${status} の移動で要約に縮め、ほかの文書から記録へのリンクを直す`, async () => {
    const documents = new Map([
      [from, record('in_progress')],
      ['work-items/active/wi-10002-two.md', '# 次\n\n[前](wi-10001-one.md#項目)\n'],
      ['docs/README.md', '# 文書\n'],
    ])
    const root = await workspace(documents)

    const command = Bun.spawnSync(
      ['bun', resolve(import.meta.dir, 'main.ts'), 'wi-10001', ...flags],
      { cwd: root, env: { ...process.env, SPEC_WORKSPACE_ROOT: root } },
    )
    expect(command.stderr.toString()).toBe('')
    expect(command.exitCode).toBe(0)
    expect(await Bun.file(join(root, from)).exists()).toBe(false)
    const moved = await readFile(join(root, to), 'utf8')
    expect(moved).toBe(
      [
        '---',
        `status: ${status}`,
        '---',
        '',
        '# 項目',
        '',
        '文書の表記を直した。',
        '',
        '全文は `git log -1 -- work-items/active/wi-10001-one.md` が示すコミットの親、またはそのコミットで読む。',
        '',
      ].join('\n'),
    )
    documents.delete(from)
    documents.set(to, moved)
    documents.set(
      'work-items/active/wi-10002-two.md',
      await readFile(join(root, 'work-items/active/wi-10002-two.md'), 'utf8'),
    )
    expect(documents.get('work-items/active/wi-10002-two.md')).toContain(
      '../done/wi-10001-one.md#項目',
    )
    expect(verifyMarkdownLinks(documents, new Set(documents.keys()))).toEqual([])
  })
}

it('要約のない記録と移動先の衝突を、書き込みの前に拒否する', async () => {
  const withoutSummary = record('in_progress', '')
  const root = await workspace(new Map([[from, withoutSummary]]))
  await expect(moveWorkItem(root, 'wi-10001')).rejects.toThrow('## 要約')
  expect(await readFile(join(root, from), 'utf8')).toBe(withoutSummary)
  await writeFile(join(root, from), record('in_progress'))
  await mkdir(join(root, 'work-items/done'), { recursive: true })
  await writeFile(join(root, to), '# 既存\n')
  await expect(moveWorkItem(root, 'wi-10001')).rejects.toThrow('already exists')
  expect(await readFile(join(root, to), 'utf8')).toBe('# 既存\n')
})

it('完了の検査に通らない記録を、書き込みの前に拒否する', async () => {
  const broken = `---\nstatus: in_progress\nchange_kind: bugfix\naffected_spec:\n  - { path: docs/modules/demo/README.md, requirement: REQ-DEMO-001 }\n---\n\n# 項目\n\n## 要約\n\n直した。\n`
  const root = await workspace(new Map([[from, broken]]))
  await expect(moveWorkItem(root, 'wi-10001')).rejects.toThrow('affected_spec path does not exist')
  expect(await readFile(join(root, from), 'utf8')).toBe(broken)
})

it('参照定義、画像、表、引用、タイトルとコードを区別して移行する', () => {
  const old = 'work-items/wi-10001-one.md'
  const migrated = 'work-items/active/wi-10001-one.md'
  const source = [
    '# 項目',
    '',
    '[文書](../docs/README.md "説明")',
    '[参照][r]',
    '',
    '[r]: <../docs/README.md#文書> "タイトル"',
    '',
    '![画像](../docs/picture.png)',
    '',
    '| 文書 | 文書 |',
    '| --- | --- |',
    '| [一](../docs/README.md) | [一](../docs/README.md) |',
    '',
    '> [引用](../docs/README.md)',
    '',
    '`[例](../docs/README.md)`',
    '',
    '`[例](../docs/README.md)` [文書](../docs/README.md)',
    '',
    '```md',
    '[例](../docs/README.md)',
    '```',
    '',
  ].join('\n')
  const result = relocateMarkdownLinks(source, old, migrated, new Map([[old, migrated]]))
  expect(result).toContain('[文書](../../docs/README.md "説明")')
  expect(result).toContain('[r]: <../../docs/README.md#文書> "タイトル"')
  expect(result).toContain('![画像](../../docs/picture.png)')
  expect(result).toContain('| [一](../../docs/README.md) | [一](../../docs/README.md) |')
  expect(result).toContain('> [引用](../../docs/README.md)')
  expect(result).toContain('`[例](../docs/README.md)`')
  expect(result).toContain('`[例](../docs/README.md)` [文書](../../docs/README.md)')
  expect(result).toContain('```md\n[例](../docs/README.md)\n```')
})

it('外部 URL、絶対 URL、不正なエンコードとリンクでない文字列を維持する', () => {
  const source = '[外部](https://example.com/a) [絶対](/docs/a) [不正](%XX.md)\n`../docs/a`\n'
  expect(relocateMarkdownLinks(source, `work-items/${item}`, from, new Map())).toBe(source)
})

it('リンク先のバックスラッシュを、エスケープとして読まれないよう書き戻す', () => {
  const source = '[文書](../docs/a.md#a\\\\_b) [山括弧](<../docs/a.md#a\\\\_b>)\n'
  expect(relocateMarkdownLinks(source, `work-items/${item}`, from, new Map())).toBe(
    '[文書](../../docs/a.md#a\\\\_b) [山括弧](<../../docs/a.md#a\\\\_b>)\n',
  )
})

it('同じ対象の正常なリンクがあっても、成立しないリンク構文を書き換えない', () => {
  const source = '[正常](../docs/a.md) [未成立](../docs/a.md "閉じない)\n'
  expect(relocateMarkdownLinks(source, `work-items/${item}`, from, new Map())).toBe(
    '[正常](../../docs/a.md) [未成立](../docs/a.md "閉じない)\n',
  )
})

it('fuzz: 移動前後でリンクの対象を保存し、往復で本文を復元する', () => {
  const markdown = new MarkdownIt()
  let seed = 35829
  for (let index = 0; index < 300; index++) {
    seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0
    const target = `docs/${encodeURIComponent(`資料 (${seed})`)}.md`
    const suffix = `?view=${seed}#見出し`
    const old = `work-items/${item}`
    const source = `[文書](<../${target}${suffix}> "説明")\n`
    const result = relocateMarkdownLinks(source, old, to, new Map([[old, to]]))
    const href = markdown
      .parseInline(result, {})[0]
      ?.children?.find((token) => token.type === 'link_open')
      ?.attrGet('href')
    expect(
      posix.normalize(posix.join(posix.dirname(to), decodeURI(String(href).split('?')[0]!))),
    ).toBe(decodeURI(target))
    expect(relocateMarkdownLinks(result, to, old, new Map([[to, old]]))).toBe(source)
  }
})
