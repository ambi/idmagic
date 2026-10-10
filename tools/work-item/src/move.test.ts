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

for (const status of ['completed', 'cancelled']) {
  it(`${status} の移動で共通文書へのリンクを保ち、項目間のリンクを直す`, async () => {
    const root = await mkdtemp(join(tmpdir(), 'idmagic-move-item-'))
    roots.push(root)
    const documents = new Map([
      [
        from,
        `---\nstatus: ${status}\n---\n\n# 項目\n\n[文書](../../docs/README.md)\n[次](wi-10002-two.md)\n`,
      ],
      ['work-items/active/wi-10002-two.md', '# 次\n\n[前](wi-10001-one.md#項目)\n'],
      ['work-items/done/wi-10003-three.md', '# 過去\n\n[前](../active/wi-10001-one.md)\n'],
      ['docs/README.md', '# 文書\n'],
    ])
    for (const [path, source] of documents) {
      await mkdir(dirname(join(root, path)), { recursive: true })
      await writeFile(join(root, path), source)
    }

    const command = Bun.spawnSync(['bun', resolve(import.meta.dir, 'main.ts'), 'wi-10001'], {
      cwd: root,
      env: { ...process.env, SPEC_WORKSPACE_ROOT: root },
    })
    expect(command.exitCode).toBe(0)
    expect(await Bun.file(join(root, from)).exists()).toBe(false)
    documents.delete(from)
    documents.set(to, await readFile(join(root, to), 'utf8'))
    for (const path of documents.keys())
      documents.set(path, await readFile(join(root, path), 'utf8'))
    expect(documents.get(to)).toContain('[文書](../../docs/README.md)')
    expect(documents.get(to)).toContain('[次](../active/wi-10002-two.md)')
    expect(documents.get('work-items/active/wi-10002-two.md')).toContain(
      '../done/wi-10001-one.md#項目',
    )
    expect(documents.get('work-items/done/wi-10003-three.md')).toContain('(wi-10001-one.md)')
    expect(verifyMarkdownLinks(documents, new Set(documents.keys()))).toEqual([])
  })
}

it('未完了と移動先の衝突を、書き込みの前に拒否する', async () => {
  const root = await mkdtemp(join(tmpdir(), 'idmagic-move-item-'))
  roots.push(root)
  await mkdir(join(root, 'work-items/active'), { recursive: true })
  await mkdir(join(root, 'work-items/done'), { recursive: true })
  const pending = '---\nstatus: pending\n---\n# 未完了\n'
  await writeFile(join(root, from), pending)
  await expect(moveWorkItem(root, 'wi-10001')).rejects.toThrow('completed or cancelled')
  expect(await readFile(join(root, from), 'utf8')).toBe(pending)
  await writeFile(join(root, from), pending.replace('pending', 'completed'))
  await writeFile(join(root, to), '# 既存\n')
  await expect(moveWorkItem(root, 'wi-10001')).rejects.toThrow('already exists')
  expect(await readFile(join(root, to), 'utf8')).toBe('# 既存\n')
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
