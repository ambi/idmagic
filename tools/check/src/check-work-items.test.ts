import { afterEach, describe, expect, it } from 'bun:test'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { createWorkspaceSnapshot } from '../../workspace/src/workspace.ts'
import { loadWorkItems } from './check-work-items.ts'
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
