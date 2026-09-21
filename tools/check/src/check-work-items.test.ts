import { afterEach, describe, expect, it } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { checkWorkItems, loadWorkItems } from './check-work-items.ts'
import { validateMarkdownRecord } from './work-item-markdown.ts'

const temporaryDirectories: string[] = []

afterEach(async () => {
  await Promise.all(temporaryDirectories.splice(0).map((path) => rm(path, { recursive: true })))
})

describe('loadWorkItems', () => {
  it('一覧、本文、解析結果を対象ごとに一度だけ作る', async () => {
    const root = await mkdtemp(join(tmpdir(), 'idmagic-work-items-'))
    temporaryDirectories.push(root)
    await mkdir(join(root, 'work-items', 'done'), { recursive: true })
    const source =
      '---\nstatus: pending\nauthors: [tn]\nrisk: low\ncreated_at: 2026-09-12\n---\n\n# Test\n'
    await writeFile(join(root, 'work-items', 'wi-901-one.md'), source)
    await writeFile(join(root, 'work-items', 'done', 'wi-902-two.md'), source)

    const base = createWorkspaceSnapshot(root)
    const listCount = new Map<string, number>()
    const readCount = new Map<string, number>()
    const snapshot = {
      ...base,
      list: async (directory = '') => {
        listCount.set(directory, (listCount.get(directory) ?? 0) + 1)
        return base.list(directory)
      },
      read: async (path: string) => {
        readCount.set(path, (readCount.get(path) ?? 0) + 1)
        return base.read(path)
      },
    }
    let parseCount = 0
    const records = await loadWorkItems(snapshot, (...args) => {
      parseCount++
      return validateMarkdownRecord(...args)
    })

    expect(records.map((record) => record.id).sort()).toEqual(['wi-901-one', 'wi-902-two'])
    expect(listCount).toEqual(
      new Map([
        ['work-items', 1],
        ['work-items/done', 1],
      ]),
    )
    expect(readCount).toEqual(
      new Map([
        ['work-items/wi-901-one.md', 1],
        ['work-items/done/wi-902-two.md', 1],
      ]),
    )
    expect(parseCount).toBe(2)
  })
})

/** スキーマを満たす最小の記録。番号の衝突だけを唯一の所見として残すために使う。 */
const minimalRecord = (title: string) =>
  [
    '---',
    'status: pending',
    'authors: [tn]',
    'risk: low',
    'created_at: 2026-09-21',
    'priority: p2',
    'depends_on: []',
    'change_kind: tooling',
    'spec_impact: { kind: none, reason: "検査の fixture であり、規範要素を変えない。" }',
    '---',
    '',
    `# ${title}`,
    '',
    '## 動機',
    '',
    '検査の fixture である。',
    '',
    '## 対象範囲',
    '',
    '- 何も変えない。',
    '',
    '## 対象外',
    '',
    '- 何も変えない。',
    '',
    '## 検証',
    '',
    '- `mise run check-work-items`',
    '',
    '## リスク',
    '',
    'なし。',
    '',
  ].join('\n')

describe('checkWorkItems', () => {
  it('題名が違っても識別番号が同じ二つの記録を落とす', async () => {
    const root = await mkdtemp(join(tmpdir(), 'idmagic-work-items-'))
    temporaryDirectories.push(root)
    await mkdir(join(root, 'work-items', 'done'), { recursive: true })
    await writeFile(join(root, 'work-items', 'wi-40318-one.md'), minimalRecord('One'))
    await writeFile(join(root, 'work-items', 'done', 'wi-40318-two.md'), minimalRecord('Two'))

    const outcome = await checkWorkItems(createWorkspaceSnapshot(root))

    expect(outcome.ok).toBe(false)
    expect(outcome.lines.join('\n')).toContain(
      "work-item identifier: 40318 is shared by 'wi-40318-one' and 'wi-40318-two'",
    )
  })

  it('番号が重ならない記録だけの workspace を通す', async () => {
    const root = await mkdtemp(join(tmpdir(), 'idmagic-work-items-'))
    temporaryDirectories.push(root)
    await mkdir(join(root, 'work-items', 'done'), { recursive: true })
    await writeFile(join(root, 'work-items', 'wi-40318-one.md'), minimalRecord('One'))
    await writeFile(join(root, 'work-items', 'done', 'wi-40319-two.md'), minimalRecord('Two'))

    const outcome = await checkWorkItems(createWorkspaceSnapshot(root))

    expect(outcome).toEqual({ ok: true, lines: ['ok  2 work-item dependency record(s)'] })
  })
})
