import { describe, expect, it } from 'bun:test'
import { parseWorkItemGroups, verifyWorkItemGroups } from './work-item-groups.ts'

const INDEX = 'docs/development/work-item-groups.md'

const index = [
  '# 作業項目のグループ',
  '',
  '| グループ | 目的 | 基準の優先度 |',
  '| --- | --- | --- |',
  '| `spec-conformance` | 規範と実装の食い違いを直す | p1 |',
  '| `release` | 初回リリースに備える | p3 |',
  '',
].join('\n')

const record = (status: string, group?: unknown) => ({
  path: 'work-items/active/wi-10001-example.md',
  data: group === undefined ? { status } : { status, group },
})

describe('parseWorkItemGroups', () => {
  it('表の最初の列にコードで書いた名前をグループとして読む', () => {
    expect(parseWorkItemGroups(index)).toEqual({
      groups: ['spec-conformance', 'release'],
      findings: [],
    })
  })

  it('同じ名前の二つ目の行を重複として報告する', () => {
    const duplicated = `${index}| \`release\` | 二つ目 | p3 |\n`
    expect(parseWorkItemGroups(duplicated).findings).toEqual([
      "duplicate work item group 'release'",
    ])
  })

  it('グループの行が一つもない索引を報告する', () => {
    expect(parseWorkItemGroups('# 作業項目のグループ\n').findings).toEqual([
      'the index declares no work item group',
    ])
  })
})

describe('verifyWorkItemGroups', () => {
  it('索引にあるグループを持つ未完了の記録を受け入れる', () => {
    const groups = ['spec-conformance', 'release']
    expect(
      verifyWorkItemGroups(
        [record('pending', 'release'), record('in_progress', 'spec-conformance')],
        groups,
        INDEX,
      ),
    ).toEqual([])
  })

  it('グループのない未完了の記録を拒否する', () => {
    expect(verifyWorkItemGroups([record('pending')], ['release'], INDEX)).toEqual([
      `work-items/active/wi-10001-example.md: missing group; choose one listed in ${INDEX}`,
    ])
  })

  it('索引にないグループを拒否する', () => {
    expect(verifyWorkItemGroups([record('pending', 'unknown')], ['release'], INDEX)).toEqual([
      `work-items/active/wi-10001-example.md: unknown group 'unknown'; choose one listed in ${INDEX}`,
    ])
  })

  it('完了と中止の記録にはグループを求めない', () => {
    expect(
      verifyWorkItemGroups([record('completed'), record('cancelled')], ['release'], INDEX),
    ).toEqual([])
  })

  it('索引がない workspace ではグループを求めない', () => {
    expect(verifyWorkItemGroups([record('pending')], undefined, INDEX)).toEqual([])
  })
})
